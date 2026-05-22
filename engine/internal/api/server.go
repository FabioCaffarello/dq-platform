// path: engine/internal/api/server.go

package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// defaultReadHeaderTimeout protects the listener against
// slow-header attacks. The trigger handler's request bodies are
// small and bounded by the per-field length ceilings in the
// decoder; the read header timeout is the only listener-level
// safety primitive in scope for P4e (broader rate-limiting and
// payload-size enforcement are deferred per ADR-0014 OQ-CC.2 and
// OQ-MD-2.2).
const defaultReadHeaderTimeout = 5 * time.Second

// Server is the HTTP server wrapper for the trigger handler. It
// assembles the mux, owns the underlying *http.Server, and exposes
// a graceful Shutdown method that integrates with the engine
// binary's sync.WaitGroup-based shutdown path.
type Server struct {
	server *http.Server
	logger *slog.Logger
}

// NewServer assembles the mux from the Handler and returns a
// Server bound to addr. It does not start the listener — call
// ListenAndServe (typically in a goroutine).
//
// The mux uses Go 1.22+ method+path patterns so each route binds
// to its committed HTTP method exactly. Patterns:
//   - POST /v1/trigger
//   - GET  /healthz
//   - GET  /readyz
//
// Any other path returns 404 via the default ServeMux behavior;
// any wrong-method request on a registered path returns 405 (Go's
// ServeMux handles method-specific patterns this way out of the
// box).
func NewServer(addr string, handler *Handler, logger *slog.Logger) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/trigger", handler.HandleTrigger)
	mux.HandleFunc("GET /healthz", handler.HandleHealthz)
	mux.HandleFunc("GET /readyz", handler.HandleReadyz)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{server: srv, logger: logger}
}

// ListenAndServe starts the listener and serves requests until
// Shutdown is called or the listener errors. Returns nil on
// normal shutdown (http.ErrServerClosed), or the underlying error
// otherwise. The engine binary invokes this in a goroutine and
// surfaces failures via the same shutdown path.
func (s *Server) ListenAndServe() error {
	s.logger.Info("http server listening", "addr", s.server.Addr)
	err := s.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully drains in-flight requests and closes the
// listener. The engine binary invokes this on signal-context
// cancellation, before sync.WaitGroup.Wait. The provided ctx
// bounds the drain window; long-running requests beyond ctx
// deadline are forcibly closed.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("http server shutting down")
	return s.server.Shutdown(ctx)
}

// Addr returns the configured listener address. Useful for tests
// that need to construct a client URL.
func (s *Server) Addr() string { return s.server.Addr }
