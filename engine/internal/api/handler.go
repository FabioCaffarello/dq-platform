// path: engine/internal/api/handler.go

package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"

	"dq-platform/engine/internal/alerts"
	"dq-platform/engine/internal/loader"
	"dq-platform/engine/internal/results"
	"dq-platform/engine/internal/runner"
)

// Dispatcher is the runner-shaped dependency the handler invokes
// for each accepted trigger. *runner.Runner satisfies this
// interface; tests substitute a fake to assert dispatcher inputs
// and inject deterministic completion signaling.
type Dispatcher interface {
	Run(ctx context.Context, trigger runner.TriggerRequest) (*results.ExecutionRow, error)
}

// ErrEntityNotInManifest is the sentinel a ResolveChecks closure
// returns when the requested entity has no rule in the active
// manifest. The handler maps it to HTTP 404 with the
// ENTITY_NOT_IN_MANIFEST error code. All other resolver errors
// map to HTTP 502 (upstream config failure) per the W3-P6d
// wiring; the resolver is treated as an upstream dependency that
// the handler cannot fix in-band.
var ErrEntityNotInManifest = errors.New("api: entity not in active manifest")

// ResolveChecksFunc is the signature of the closure the engine
// binary supplies to translate a trigger entity into the list of
// CheckSpecs the runner should evaluate. Implementation reads
// the entity's YAML body from the object store and parses it
// (see engine/internal/dsl/spec). The closure runs on the
// HTTP request path and respects the request context.
//
// Return contract:
//
//   - (specs, nil) — checks resolved; handler passes them through
//     to the runner.
//   - (nil, ErrEntityNotInManifest) — no rule for the entity;
//     handler returns 404.
//   - (_, other err) — system failure (object-store read,
//     parser error, etc.); handler returns 502.
//
// A nil ResolveChecksFunc means "Phase-4e behavior" — handler
// passes Checks: nil to the runner (the runner's check loop
// runs zero iterations and writes a terminal error row per
// ADR-0004 CC2 branch 2). Used by tests that exercise the
// handler in isolation.
type ResolveChecksFunc func(ctx context.Context, entity string) ([]runner.CheckSpec, error)

// Handler is the HTTP trigger handler scaffolded by W3-P4e. It
// exposes the three endpoints committed by ADR-0014: POST
// /v1/trigger, GET /healthz, GET /readyz.
//
// The handler is the data-plane boundary for the runner. It
// captures the active manifest at trigger acceptance (ADR-0007 §3
// in-flight isolation), strictly decodes the request body
// (ADR-0014 §2), mints the attempt_id (ADR-0003 §4), computes the
// execution_id (ADR-0002 §1 formula), and dispatches the runner
// asynchronously — the response carries `status: "running"` per
// ADR-0014 §3 and the runner produces the terminal row out of
// band.
type Handler struct {
	dispatcher     Dispatcher
	activeManifest func() *loader.Manifest
	resolveChecks  ResolveChecksFunc
	engineCtx      context.Context
	now            func() time.Time
	logger         *slog.Logger
	attemptID      func() string
	onComplete     func(executionID string, err error)
	publisher      alerts.Publisher
}

// HandlerConfig groups the dependencies passed to NewHandler.
// All required fields are validated at construction. Optional
// fields (Now, Logger, AttemptID, OnComplete) default to
// production values when zero.
type HandlerConfig struct {
	// Dispatcher is the runner instance (or test fake) the
	// handler dispatches accepted triggers to. Required.
	Dispatcher Dispatcher

	// ActiveManifest is the closure that returns the engine's
	// current active manifest. Captured at trigger acceptance to
	// honor ADR-0007 §3 (in-flight executions isolated against
	// the manifest active at plan creation). Required.
	ActiveManifest func() *loader.Manifest

	// EngineCtx is the engine's signal context. The async
	// dispatcher uses it (not the HTTP request context) so the
	// runner finishes its work even after the client
	// disconnects, but shuts down cleanly on SIGTERM/SIGINT.
	// Required.
	EngineCtx context.Context

	// Now is the clock used for the response's `accepted_at`
	// timestamp. Optional; defaults to time.Now.
	Now func() time.Time

	// Logger is the structured logger. Optional; defaults to a
	// discarding handler.
	Logger *slog.Logger

	// AttemptID overrides the UUID minter (test injection point).
	// Optional; defaults to uuid.NewString (UUID v4 per ADR-0003
	// §4).
	AttemptID func() string

	// OnComplete is invoked from the dispatcher goroutine after
	// Run returns. Tests use this to wait for completion without
	// racing on the store. Optional; nil in production. The
	// panic-recovery path also calls OnComplete so tests waiting
	// on it do not deadlock when a dispatcher panics.
	OnComplete func(executionID string, err error)

	// Publisher emits the operational alert when the dispatcher
	// goroutine panics (defer-recover path per P3 — ownership
	// visibility into failures). Optional; defaults to
	// alerts.NoopPublisher (no alerts emitted; panics still
	// recorded via Logger).
	Publisher alerts.Publisher

	// ResolveChecks translates a trigger entity into the
	// []runner.CheckSpec the runner evaluates. The engine binary
	// supplies a closure that reads the entity's YAML body from
	// the object store and parses it via dsl/spec.Parse.
	//
	// Optional; nil preserves Phase-4e behavior (handler passes
	// Checks: nil to the runner). See ResolveChecksFunc for the
	// full return contract.
	ResolveChecks ResolveChecksFunc
}

// NewHandler validates the config and returns a Handler. Returns
// an error when any required field is missing.
func NewHandler(cfg HandlerConfig) (*Handler, error) {
	if cfg.Dispatcher == nil {
		return nil, errors.New("api: Dispatcher is required")
	}
	if cfg.ActiveManifest == nil {
		return nil, errors.New("api: ActiveManifest is required")
	}
	if cfg.EngineCtx == nil {
		return nil, errors.New("api: EngineCtx is required")
	}
	h := &Handler{
		dispatcher:     cfg.Dispatcher,
		activeManifest: cfg.ActiveManifest,
		resolveChecks:  cfg.ResolveChecks,
		engineCtx:      cfg.EngineCtx,
		now:            cfg.Now,
		logger:         cfg.Logger,
		attemptID:      cfg.AttemptID,
		onComplete:     cfg.OnComplete,
		publisher:      cfg.Publisher,
	}
	if h.now == nil {
		h.now = time.Now
	}
	if h.logger == nil {
		h.logger = slog.Default()
	}
	if h.attemptID == nil {
		h.attemptID = uuid.NewString
	}
	if h.publisher == nil {
		h.publisher = alerts.NoopPublisher{}
	}
	return h, nil
}

// HandleTrigger serves POST /v1/trigger. The five steps in order:
//
//  1. Strict-decode the body per ADR-0014 §2. Reject with 400 on
//     unknown fields, missing fields, invalid UTF-8, ASCII pipe,
//     malformed timestamps, non-Z timezone, closed-enum violation,
//     or window-order violation.
//  2. Read the active manifest via the closure (ADR-0007 §3).
//  3. Compute execution_id (ADR-0002 §1 formula).
//  4. Mint attempt_id (ADR-0003 §4 — UUID v4 by default).
//  5. Spawn the runner goroutine using the engine context so
//     shutdown signals propagate; respond with the DTO carrying
//     `status: "running"` per ADR-0014 §3.
//
// The handler does not write any rows directly; the runner is
// responsible for both the running and terminal rows.
func (h *Handler) HandleTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, &ErrorResponse{
			Code:    ErrCodeMethodNotAllowed,
			Message: fmt.Sprintf("method %s not allowed on /v1/trigger; use POST", r.Method),
		})
		return
	}

	req, decodeErr := DecodeRequest(r.Body)
	if decodeErr != nil {
		writeError(w, http.StatusBadRequest, decodeErr)
		return
	}

	manifest := h.activeManifest()
	if manifest == nil {
		// Per ADR-0014 §1 the listener is bound only after the
		// initial load completes, so in practice this is
		// unreachable. The guard exists so tests and any future
		// listener-without-loader topology fail loud.
		writeError(w, http.StatusServiceUnavailable, &ErrorResponse{
			Code:    ErrCodeInternal,
			Message: "active manifest is unavailable",
		})
		return
	}

	windowStart := mustParseValidatedWindow(req.WindowStart)
	windowEnd := mustParseValidatedWindow(req.WindowEnd)
	triggerSource := results.TriggerSource(req.TriggerSource)

	// ADR-0002 §1: execution_id formula. The handler computes it
	// at acceptance so the response DTO and the persisted rows
	// carry the same identifier.
	executionID, err := runner.Compute(
		manifest.RulesetVersion,
		req.Entity,
		windowStart,
		windowEnd,
		triggerSource,
	)
	if err != nil {
		// runner.Compute can only fail on inputs that the
		// decoder already rejected (pipe in input). Treat as
		// internal — the decoder invariant should have caught it.
		h.logger.Error("execution_id compute failed after decode",
			"error", err.Error(),
			"entity", req.Entity,
		)
		writeError(w, http.StatusInternalServerError, &ErrorResponse{
			Code:    ErrCodeInternal,
			Message: "execution_id compute failed",
		})
		return
	}

	// W3-P6d: resolve the entity's check list from the active
	// manifest before computing the response DTO. The closure
	// reads the YAML body from the object store and parses it
	// (see engine/internal/dsl/spec); errors short-circuit the
	// trigger so the runner never starts on an unresolvable
	// entity. When ResolveChecks is nil the handler preserves
	// Phase-4e behavior (Checks: nil) for tests that exercise
	// the handler in isolation.
	var checks []runner.CheckSpec
	if h.resolveChecks != nil {
		var resolveErr error
		checks, resolveErr = h.resolveChecks(r.Context(), req.Entity)
		if resolveErr != nil {
			if errors.Is(resolveErr, ErrEntityNotInManifest) {
				writeError(w, http.StatusNotFound, &ErrorResponse{
					Code:    ErrCodeEntityNotInManifest,
					Field:   "entity",
					Message: fmt.Sprintf("no rule for entity %q in active manifest", req.Entity),
				})
				return
			}
			h.logger.Error("check resolution failed",
				"entity", req.Entity,
				"error", resolveErr.Error(),
			)
			writeError(w, http.StatusBadGateway, &ErrorResponse{
				Code:    ErrCodeCheckResolutionFailed,
				Message: "check resolution upstream failure",
			})
			return
		}
	}

	attemptID := h.attemptID()
	acceptedAt := h.now().UTC()

	runnerReq := runner.TriggerRequest{
		Entity:         req.Entity,
		WindowStart:    windowStart,
		WindowEnd:      windowEnd,
		TriggerSource:  triggerSource,
		RulesetVersion: manifest.RulesetVersion,
		AttemptID:      &attemptID,
		// W3-P6d wires manifest-driven check resolution above.
		// When ResolveChecks is nil (test path), Checks stays
		// nil and the runner's loop runs zero iterations.
		Checks: checks,
	}

	// Dispatch the runner under the engine context so the work
	// outlives the client connection but still terminates on
	// engine shutdown. Goroutines are unbounded per accepted
	// trigger — rate limiting and concurrency budgets are
	// deferred to a follow-up ADR per ADR-0014 OQ-CC.2.
	go func() {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			stack := debug.Stack()
			h.logger.Error("trigger handler goroutine panic",
				"execution_id", executionID,
				"attempt_id", attemptID,
				"entity", req.Entity,
				"panic", fmt.Sprintf("%v", rec),
				"stack", string(stack),
			)
			// Best-effort operational alert per P3 (ownership
			// explicit everywhere). Publisher failure is logged
			// but does not propagate — alerting is best-effort
			// out-of-band signal (ADR-0006 CC5 pattern).
			sev := alerts.SeverityCritical
			errSum := fmt.Sprintf("trigger handler panic: %v", rec)
			alertCtx, cancel := context.WithTimeout(h.engineCtx, 5*time.Second)
			defer cancel()
			if pubErr := h.publisher.Publish(alertCtx, alerts.Event{
				ExecutionID:  &executionID,
				AttemptID:    &attemptID,
				Entity:       req.Entity,
				Category:     alerts.CategoryOperational,
				Severity:     &sev,
				EventSource:  alerts.SourceTriggerHandler,
				RecordedAt:   h.now().UTC(),
				ErrorSummary: &errSum,
			}); pubErr != nil {
				h.logger.Warn("panic-alert publish failed",
					"execution_id", executionID,
					"error", pubErr.Error(),
				)
			}
			if h.onComplete != nil {
				h.onComplete(executionID, fmt.Errorf("panic: %v", rec))
			}
		}()

		_, err := h.dispatcher.Run(h.engineCtx, runnerReq)
		if err != nil {
			h.logger.Warn("dispatcher returned error post-acceptance",
				"execution_id", executionID,
				"attempt_id", attemptID,
				"entity", req.Entity,
				"error", err.Error(),
			)
		}
		if h.onComplete != nil {
			h.onComplete(executionID, err)
		}
	}()

	h.logger.Info("trigger accepted",
		"execution_id", executionID,
		"attempt_id", attemptID,
		"entity", req.Entity,
		"trigger_source", req.TriggerSource,
		"ruleset_version", manifest.RulesetVersion,
	)

	resp := TriggerHTTPResponse{
		ExecutionID: executionID,
		AttemptID:   attemptID,
		Status:      string(results.StatusRunning),
		AcceptedAt:  acceptedAt.Format(timestampLayout),
		Self:        "/v1/executions/" + executionID,
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleHealthz serves GET /healthz. Returns 200 OK while the
// process is up; does not depend on manifest state. Liveness
// probes use this endpoint per ADR-0014 §4.
func (h *Handler) HandleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleReadyz serves GET /readyz. Returns 200 OK while the
// listener is reachable. The listener is bound only after the
// first successful manifest load (ADR-0014 §1), so /readyz being
// reachable already implies readiness — no in-band 503 path is
// needed. Refresh failures do not flip /readyz per ADR-0014 §4
// (the escalation signal goes out of band via ADR-0006 alerting).
func (h *Handler) HandleReadyz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// writeJSON marshals v as JSON and writes the response. Marshal
// errors are logged but not propagated — by this point the
// status code has been written and the client is committed.
func writeJSON(w http.ResponseWriter, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		// Best-effort: try to surface the failure. The header
		// may already have been written; in that case the client
		// gets a truncated body, which is the least-bad outcome.
		http.Error(w, `{"code":"INTERNAL_ERROR","message":"response marshal failed"}`,
			http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// writeError marshals an ErrorResponse and writes the response
// with the given status code. The envelope shape is committed
// by ADR-0014 §"Consequences" item 6.
func writeError(w http.ResponseWriter, status int, env *ErrorResponse) {
	writeJSON(w, status, env)
}
