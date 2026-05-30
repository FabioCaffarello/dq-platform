// path: engine/internal/api/handler_test.go

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"dq-platform/engine/internal/loader"
	"dq-platform/engine/internal/results"
	"dq-platform/engine/internal/runner"
)

// --- Test scaffolding ---------------------------------------------------

// captureDispatcher records every Run invocation. Tests use it
// instead of a real *runner.Runner so the handler's call
// arguments can be asserted without spinning up the full
// runner+store stack.
type captureDispatcher struct {
	mu       sync.Mutex
	calls    []runner.TriggerRequest
	runErr   error
	runDelay time.Duration
}

func (c *captureDispatcher) Run(ctx context.Context, trigger runner.TriggerRequest) (*results.ExecutionRow, error) {
	if c.runDelay > 0 {
		select {
		case <-time.After(c.runDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	c.mu.Lock()
	c.calls = append(c.calls, trigger)
	c.mu.Unlock()
	if c.runErr != nil {
		return nil, c.runErr
	}
	return &results.ExecutionRow{}, nil
}

func (c *captureDispatcher) callCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.calls)
}

func (c *captureDispatcher) lastCall() runner.TriggerRequest {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls[len(c.calls)-1]
}

// testHandler builds a Handler backed by captureDispatcher and a
// synthetic active manifest. The completion channel fires once
// the dispatcher goroutine returns so tests can assert on the
// dispatch outcome without racing.
func testHandler(t *testing.T) (*Handler, *captureDispatcher, *loader.Manifest, chan struct{}) {
	t.Helper()
	m := &loader.Manifest{
		ManifestVersion: 1,
		RulesetVersion:  "rules-v1.0.0",
		Hash:            "deadbeef",
	}
	d := &captureDispatcher{}
	complete := make(chan struct{}, 8)
	h, err := NewHandler(HandlerConfig{
		Dispatcher:     d,
		ActiveManifest: func() *loader.Manifest { return m },
		EngineCtx:      context.Background(),
		Now: func() time.Time {
			return time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
		},
		AttemptID: func() string { return "00000000-0000-0000-0000-000000000001" },
		OnComplete: func(_ string, _ error) {
			complete <- struct{}{}
		},
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	return h, d, m, complete
}

func waitForDispatch(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("dispatcher goroutine did not complete within 2s")
	}
}

func post(t *testing.T, h http.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/trigger", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h(w, req)
	return w
}

// validBody returns a JSON body that passes every decoder
// invariant — tests mutate it via string replace to exercise
// each rejection path.
const validBody = `{
  "entity": "customer",
  "window_start": "2026-05-22T14:00:00Z",
  "window_end": "2026-05-22T15:00:00Z",
  "trigger_source": "scheduler"
}`

// --- Happy-path tests --------------------------------------------------

func TestHandleTrigger_HappyPath_ReturnsRunningStatus(t *testing.T) {
	h, d, manifest, complete := testHandler(t)
	w := post(t, h.HandleTrigger, validBody)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200. body=%s", w.Code, w.Body.String())
	}
	var resp TriggerHTTPResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v body=%s", err, w.Body.String())
	}
	if resp.Status != "running" {
		t.Errorf("status = %q; want %q", resp.Status, "running")
	}
	if len(resp.ExecutionID) != 64 {
		t.Errorf("execution_id length = %d; want 64", len(resp.ExecutionID))
	}
	if resp.AttemptID != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("attempt_id = %q; want injected UUID", resp.AttemptID)
	}
	if resp.AcceptedAt != "2026-05-22T12:00:00Z" {
		t.Errorf("accepted_at = %q; want injected clock UTC Z", resp.AcceptedAt)
	}
	if resp.Self != "/v1/executions/"+resp.ExecutionID {
		t.Errorf("self = %q; want /v1/executions/{execution_id}", resp.Self)
	}
	waitForDispatch(t, complete)
	if d.callCount() != 1 {
		t.Fatalf("dispatcher call count = %d; want 1", d.callCount())
	}
	got := d.lastCall()
	if got.Entity != "customer" {
		t.Errorf("dispatcher Entity = %q; want %q", got.Entity, "customer")
	}
	if got.RulesetVersion != manifest.RulesetVersion {
		t.Errorf("dispatcher RulesetVersion = %q; want %q", got.RulesetVersion, manifest.RulesetVersion)
	}
	if got.AttemptID == nil || *got.AttemptID != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("dispatcher AttemptID = %v; want injected UUID", got.AttemptID)
	}
	if got.TriggerSource != results.TriggerScheduler {
		t.Errorf("dispatcher TriggerSource = %q; want scheduler", got.TriggerSource)
	}
	if len(got.Checks) != 0 {
		t.Errorf("dispatcher Checks = %v; want empty (P6 wires real checks)", got.Checks)
	}
}

func TestHandleTrigger_ExecutionIDReproducible(t *testing.T) {
	// ADR-0002 §1: the formula is deterministic. The same body
	// twice produces the same execution_id.
	h, _, _, complete := testHandler(t)
	w1 := post(t, h.HandleTrigger, validBody)
	waitForDispatch(t, complete)
	w2 := post(t, h.HandleTrigger, validBody)
	waitForDispatch(t, complete)

	var r1, r2 TriggerHTTPResponse
	_ = json.Unmarshal(w1.Body.Bytes(), &r1)
	_ = json.Unmarshal(w2.Body.Bytes(), &r2)
	if r1.ExecutionID != r2.ExecutionID {
		t.Errorf("execution_id differs across identical bodies: %q vs %q", r1.ExecutionID, r2.ExecutionID)
	}
}

// --- Decoder-rejection tests (ADR-0014 §2) -----------------------------

type rejectionCase struct {
	name     string
	body     string
	wantCode string
	wantField string
}

func TestHandleTrigger_DecoderRejections(t *testing.T) {
	cases := []rejectionCase{
		{
			name:     "unknown field",
			body:     `{"entity":"x","window_start":"2026-05-22T14:00:00Z","window_end":"2026-05-22T15:00:00Z","trigger_source":"scheduler","extra":"nope"}`,
			wantCode: ErrCodeUnknownField,
		},
		{
			name:     "missing entity",
			body:     strings.Replace(validBody, `"entity": "customer"`, `"entity": ""`, 1),
			wantCode: ErrCodeMissingField,
			wantField: "entity",
		},
		{
			name:     "pipe in entity",
			body:     strings.Replace(validBody, `"customer"`, `"cust|omer"`, 1),
			wantCode: ErrCodePipeInInput,
			wantField: "entity",
		},
		{
			name:     "+00:00 instead of Z",
			body:     strings.Replace(validBody, `"2026-05-22T14:00:00Z"`, `"2026-05-22T14:00:00+00:00"`, 1),
			wantCode: ErrCodeInvalidWindowFormat,
			wantField: "window_start",
		},
		{
			name:     "non-RFC3339 timestamp",
			body:     strings.Replace(validBody, `"2026-05-22T14:00:00Z"`, `"2026-05-22 14:00:00"`, 1),
			wantCode: ErrCodeInvalidWindowFormat,
			wantField: "window_start",
		},
		{
			name:     "window_end equal to window_start",
			body:     strings.Replace(validBody, `"window_end": "2026-05-22T15:00:00Z"`, `"window_end": "2026-05-22T14:00:00Z"`, 1),
			wantCode: ErrCodeInvalidWindowOrder,
			wantField: "window_end",
		},
		{
			name:     "trigger_source operator-rerun rejected at data-plane path",
			body:     strings.Replace(validBody, `"scheduler"`, `"operator-rerun"`, 1),
			wantCode: ErrCodeInvalidTriggerSrc,
			wantField: "trigger_source",
		},
		{
			name:     "unknown trigger_source value",
			body:     strings.Replace(validBody, `"scheduler"`, `"webhook"`, 1),
			wantCode: ErrCodeInvalidTriggerSrc,
			wantField: "trigger_source",
		},
		{
			name:     "malformed JSON",
			body:     `{not json}`,
			wantCode: ErrCodeDecodeError,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			h, d, _, _ := testHandler(t)
			w := post(t, h.HandleTrigger, tc.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d; want 400. body=%s", w.Code, w.Body.String())
			}
			var env ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
				t.Fatalf("unmarshal envelope: %v body=%s", err, w.Body.String())
			}
			if env.Code != tc.wantCode {
				t.Errorf("code = %q; want %q. message=%s", env.Code, tc.wantCode, env.Message)
			}
			if tc.wantField != "" && env.Field != tc.wantField {
				t.Errorf("field = %q; want %q", env.Field, tc.wantField)
			}
			if d.callCount() != 0 {
				t.Errorf("dispatcher invoked on rejection (call count = %d)", d.callCount())
			}
		})
	}
}

func TestHandleTrigger_EntityLengthCeiling(t *testing.T) {
	overlong := strings.Repeat("a", maxStringFieldLen+1)
	body := strings.Replace(validBody, `"customer"`, `"`+overlong+`"`, 1)
	h, d, _, _ := testHandler(t)
	w := post(t, h.HandleTrigger, body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want 400. body=%s", w.Code, w.Body.String())
	}
	var env ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if env.Code != ErrCodeInvalidFieldLength {
		t.Errorf("code = %q; want %q", env.Code, ErrCodeInvalidFieldLength)
	}
	if d.callCount() != 0 {
		t.Errorf("dispatcher invoked on length rejection")
	}
}

// --- Method / non-trigger endpoints ------------------------------------

func TestHandleTrigger_MethodNotAllowed(t *testing.T) {
	h, d, _, _ := testHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/trigger", nil)
	w := httptest.NewRecorder()
	h.HandleTrigger(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d; want 405", w.Code)
	}
	if d.callCount() != 0 {
		t.Errorf("dispatcher invoked on wrong-method request")
	}
}

func TestHandleHealthz(t *testing.T) {
	h, _, _, _ := testHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h.HandleHealthz(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d; want 200", w.Code)
	}
}

func TestHandleReadyz(t *testing.T) {
	h, _, _, _ := testHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	h.HandleReadyz(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d; want 200", w.Code)
	}
}

func TestHandleTrigger_ManifestUnavailable_Returns503(t *testing.T) {
	d := &captureDispatcher{}
	h, err := NewHandler(HandlerConfig{
		Dispatcher:     d,
		ActiveManifest: func() *loader.Manifest { return nil },
		EngineCtx:      context.Background(),
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	w := post(t, h.HandleTrigger, validBody)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d; want 503", w.Code)
	}
	if d.callCount() != 0 {
		t.Errorf("dispatcher invoked with nil manifest")
	}
}

// --- Server mux wiring (Go 1.22+ method+path routing) -------------------

func TestServer_MuxRoutes(t *testing.T) {
	h, _, _, _ := testHandler(t)
	srv := NewServer(":0", h, nil, nil)
	// Drive the server through httptest by serving its underlying handler.
	cases := []struct {
		method   string
		path     string
		body     io.Reader
		wantCode int
	}{
		{http.MethodPost, "/v1/trigger", strings.NewReader(validBody), http.StatusOK},
		{http.MethodGet, "/v1/trigger", nil, http.StatusMethodNotAllowed},
		{http.MethodGet, "/healthz", nil, http.StatusOK},
		{http.MethodGet, "/readyz", nil, http.StatusOK},
		{http.MethodGet, "/does-not-exist", nil, http.StatusNotFound},
		{http.MethodGet, "/metrics", nil, http.StatusNotFound}, // ADR-0055 §Clause 2: absent when metricsHandler is nil
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, tc.body)
		w := httptest.NewRecorder()
		srv.server.Handler.ServeHTTP(w, req)
		if w.Code != tc.wantCode {
			t.Errorf("%s %s -> %d; want %d", tc.method, tc.path, w.Code, tc.wantCode)
		}
	}
}

// TestServer_MetricsRouteWired asserts that when a metricsHandler
// is supplied, GET /metrics serves it (ADR-0055 §Clause 2). The
// route is method-bound — any other method on /metrics returns
// 405, matching the other routes' shape.
func TestServer_MetricsRouteWired(t *testing.T) {
	h, _, _, _ := testHandler(t)
	probe := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte("# probe\ndq_runs_total{entity=\"x\",mode=\"set\",status=\"success\",trigger_source=\"scheduler\"} 0\n"))
	})
	srv := NewServer(":0", h, nil, probe)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /metrics: status %d, want 200", w.Code)
	}
	if got := w.Body.String(); !strings.Contains(got, "dq_runs_total") {
		t.Errorf("GET /metrics body missing probe content:\n%s", got)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/metrics", nil)
	w2 := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /metrics: status %d, want 405", w2.Code)
	}
}

// --- NewHandler validation ---------------------------------------------

func TestNewHandler_RequiresDispatcher(t *testing.T) {
	_, err := NewHandler(HandlerConfig{
		ActiveManifest: func() *loader.Manifest { return nil },
		EngineCtx:      context.Background(),
	})
	if err == nil {
		t.Fatal("NewHandler accepted nil Dispatcher")
	}
}

func TestNewHandler_RequiresActiveManifest(t *testing.T) {
	_, err := NewHandler(HandlerConfig{
		Dispatcher: &captureDispatcher{},
		EngineCtx:  context.Background(),
	})
	if err == nil {
		t.Fatal("NewHandler accepted nil ActiveManifest")
	}
}

func TestNewHandler_RequiresEngineCtx(t *testing.T) {
	_, err := NewHandler(HandlerConfig{
		Dispatcher:     &captureDispatcher{},
		ActiveManifest: func() *loader.Manifest { return nil },
	})
	if err == nil {
		t.Fatal("NewHandler accepted nil EngineCtx")
	}
}

// --- Dispatch ordering: dispatcher receives a real runner.TriggerRequest -

func TestDispatcherInputs_MatchAcceptedTrigger(t *testing.T) {
	h, d, manifest, complete := testHandler(t)
	body := strings.Replace(validBody, `"customer"`, `"orders"`, 1)
	w := post(t, h.HandleTrigger, body)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", w.Code)
	}
	waitForDispatch(t, complete)
	got := d.lastCall()

	wantStart := time.Date(2026, 5, 22, 14, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 5, 22, 15, 0, 0, 0, time.UTC)
	if !got.WindowStart.Equal(wantStart) {
		t.Errorf("WindowStart = %v; want %v", got.WindowStart, wantStart)
	}
	if !got.WindowEnd.Equal(wantEnd) {
		t.Errorf("WindowEnd = %v; want %v", got.WindowEnd, wantEnd)
	}
	if got.Entity != "orders" {
		t.Errorf("Entity = %q; want orders", got.Entity)
	}
	if got.RulesetVersion != manifest.RulesetVersion {
		t.Errorf("RulesetVersion = %q; want %q", got.RulesetVersion, manifest.RulesetVersion)
	}
	if got.SupersedesExecutionID != nil {
		t.Errorf("SupersedesExecutionID = %v; want nil for scheduler trigger", got.SupersedesExecutionID)
	}
}

// --- Empty body handling -----------------------------------------------

func TestHandleTrigger_EmptyBody_Returns400(t *testing.T) {
	h, _, _, _ := testHandler(t)
	w := post(t, h.HandleTrigger, "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want 400", w.Code)
	}
	var env ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if env.Code != ErrCodeDecodeError {
		t.Errorf("code = %q; want DECODE_ERROR", env.Code)
	}
}

// --- writeJSON sets Content-Type ---------------------------------------

func TestResponseContentType(t *testing.T) {
	h, _, _, complete := testHandler(t)
	w := post(t, h.HandleTrigger, validBody)
	waitForDispatch(t, complete)
	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q; want application/json", got)
	}
}

// --- ResolveChecks (W3-P6d) ---------------------------------------------

func TestHandleTrigger_ResolveChecks_PassesChecksToRunner(t *testing.T) {
	manifest := &loader.Manifest{RulesetVersion: "rules-v1.0.0"}
	d := &captureDispatcher{}
	complete := make(chan struct{}, 8)
	h, err := NewHandler(HandlerConfig{
		Dispatcher:     d,
		ActiveManifest: func() *loader.Manifest { return manifest },
		EngineCtx:      context.Background(),
		AttemptID:      func() string { return "00000000-0000-0000-0000-00000000aaaa" },
		OnComplete:     func(_ string, _ error) { complete <- struct{}{} },
		ResolveChecks: func(_ context.Context, entity string) ([]runner.CheckSpec, error) {
			if entity != "customer" {
				t.Errorf("resolver called with entity %q; want customer", entity)
			}
			return []runner.CheckSpec{
				{CheckID: "row_count_positive", Kind: "row_count_positive"},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	w := post(t, h.HandleTrigger, validBody)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200. body=%s", w.Code, w.Body.String())
	}
	waitForDispatch(t, complete)
	got := d.lastCall()
	if len(got.Checks) != 1 {
		t.Fatalf("dispatcher Checks length = %d; want 1", len(got.Checks))
	}
	if got.Checks[0].Kind != "row_count_positive" {
		t.Errorf("dispatcher Check Kind = %q; want row_count_positive", got.Checks[0].Kind)
	}
}

func TestHandleTrigger_ResolveChecks_EntityNotInManifest_Returns404(t *testing.T) {
	manifest := &loader.Manifest{RulesetVersion: "rules-v1.0.0"}
	d := &captureDispatcher{}
	h, err := NewHandler(HandlerConfig{
		Dispatcher:     d,
		ActiveManifest: func() *loader.Manifest { return manifest },
		EngineCtx:      context.Background(),
		ResolveChecks: func(_ context.Context, _ string) ([]runner.CheckSpec, error) {
			return nil, ErrEntityNotInManifest
		},
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	w := post(t, h.HandleTrigger, validBody)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d; want 404", w.Code)
	}
	var env ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if env.Code != ErrCodeEntityNotInManifest {
		t.Errorf("code = %q; want %q", env.Code, ErrCodeEntityNotInManifest)
	}
	if env.Field != "entity" {
		t.Errorf("field = %q; want entity", env.Field)
	}
	if d.callCount() != 0 {
		t.Errorf("dispatcher invoked despite entity-not-in-manifest rejection")
	}
}

func TestHandleTrigger_ResolveChecks_SystemError_Returns502(t *testing.T) {
	manifest := &loader.Manifest{RulesetVersion: "rules-v1.0.0"}
	d := &captureDispatcher{}
	h, err := NewHandler(HandlerConfig{
		Dispatcher:     d,
		ActiveManifest: func() *loader.Manifest { return manifest },
		EngineCtx:      context.Background(),
		ResolveChecks: func(_ context.Context, _ string) ([]runner.CheckSpec, error) {
			return nil, errors.New("simulated object-store read failure")
		},
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	w := post(t, h.HandleTrigger, validBody)
	if w.Code != http.StatusBadGateway {
		t.Fatalf("status = %d; want 502", w.Code)
	}
	var env ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if env.Code != ErrCodeCheckResolutionFailed {
		t.Errorf("code = %q; want %q", env.Code, ErrCodeCheckResolutionFailed)
	}
	if d.callCount() != 0 {
		t.Errorf("dispatcher invoked despite resolver system error")
	}
}

func TestHandleTrigger_ResolveChecks_NilFunc_PreservesP4eBehavior(t *testing.T) {
	// When ResolveChecks is nil, the handler must pass Checks: nil
	// to the runner — preserving the P4e contract for tests that
	// exercise the handler in isolation.
	h, d, _, complete := testHandler(t)
	w := post(t, h.HandleTrigger, validBody)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", w.Code)
	}
	waitForDispatch(t, complete)
	got := d.lastCall()
	if got.Checks != nil {
		t.Errorf("Checks = %v; want nil (P4e backwards-compat)", got.Checks)
	}
}

// --- Decoder UTF-8 check exercised directly -----------------------------

func TestDecodeRequest_InvalidUTF8Entity(t *testing.T) {
	// Construct a JSON body where "entity" is a string containing
	// an invalid UTF-8 sequence (0xff). encoding/json itself
	// would normally reject this at parse time, but we want the
	// validate() path to be exercised as well, so we test the
	// validator directly with an already-parsed request.
	req := TriggerHTTPRequest{
		Entity:        string([]byte{0x66, 0xff, 0x6f}),
		WindowStart:   "2026-05-22T14:00:00Z",
		WindowEnd:     "2026-05-22T15:00:00Z",
		TriggerSource: "scheduler",
	}
	if env := validate(&req); env == nil || env.Code != ErrCodeInvalidUTF8 {
		t.Errorf("validate(invalid UTF-8) = %+v; want INVALID_UTF8", env)
	}
}

// --- Sanity: validBody is actually valid -------------------------------

func TestValidBodyIsValid(t *testing.T) {
	_, env := DecodeRequest(bytes.NewReader([]byte(validBody)))
	if env != nil {
		t.Fatalf("validBody rejected: %+v", env)
	}
}
