package dashboard

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	routingTimeout = 10 * time.Second
	recoverTimeout = 30 * time.Second
	keepAlive      = 20 * time.Second
)

// sseRegions are the named regions streamed to the browser on every frame.
var sseRegions = []string{"dep", "kpis", "orders", "versions", "controls"}

// Server exposes the dashboard API and serves the SPA.
type Server struct {
	hub      *Hub
	actions  *Actions
	renderer *Renderer
	frontend http.Handler
	logger   *slog.Logger
}

// NewServer builds a Server serving the SPA from the given file system (the
// embedded frontend assets in production).
func NewServer(hub *Hub, actions *Actions, renderer *Renderer, frontend fs.FS, logger *slog.Logger) *Server {
	return &Server{
		hub:      hub,
		actions:  actions,
		renderer: renderer,
		frontend: http.FileServerFS(frontend),
		logger:   logger,
	}
}

// Routes returns the configured HTTP handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /events", s.handleSSE)
	mux.HandleFunc("POST /api/ramp", s.handleRamp)
	mux.HandleFunc("POST /api/promote", s.handleAction(s.actions.Promote))
	mux.HandleFunc("POST /api/rollback", s.handleAction(s.actions.Rollback))
	mux.HandleFunc("POST /api/recover", s.handleRecover)
	mux.Handle("/", s.frontend)
	return mux
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	frames, unsubscribe := s.hub.Subscribe()
	defer unsubscribe()

	ticker := time.NewTicker(keepAlive)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ":keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case state := <-frames:
			if !s.writeFrame(w, state) {
				return
			}
			flusher.Flush()
		}
	}
}

// writeFrame renders every SSE region for state and writes each as a named SSE
// event. It returns false if writing to the client fails (connection closed).
func (s *Server) writeFrame(w http.ResponseWriter, state DashboardState) bool {
	var buf bytes.Buffer
	for _, region := range sseRegions {
		buf.Reset()
		if err := s.renderer.Region(&buf, region, state); err != nil {
			s.logger.Warn("render SSE region failed", "region", region, "err", err)
			continue
		}
		if err := writeSSEEvent(w, region, buf.String()); err != nil {
			return false
		}
	}
	return true
}

// writeSSEEvent writes a named SSE event whose data is (possibly multi-line)
// HTML. Per the SSE spec each physical line is prefixed with "data: ", and the
// event is terminated by a blank line.
func writeSSEEvent(w http.ResponseWriter, event, body string) error {
	if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
		return err
	}
	for _, line := range strings.Split(body, "\n") {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(w, "\n")
	return err
}

func (s *Server) handleRamp(w http.ResponseWriter, r *http.Request) {
	// HTMX posts hx-vals as application/x-www-form-urlencoded.
	pct, err := strconv.ParseFloat(r.FormValue("percentage"), 32)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid percentage")
		return
	}
	if pct < 0 || pct > 100 {
		s.writeError(w, http.StatusBadRequest, "percentage must be between 0 and 100")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), routingTimeout)
	defer cancel()
	if err := s.actions.Ramp(ctx, float32(pct)); err != nil {
		s.logger.Warn("ramp failed", "err", err)
		s.writeError(w, http.StatusInternalServerError, "Ramp failed: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRecover(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recoverTimeout)
	defer cancel()
	recovered, err := s.actions.Recover(ctx)
	if err != nil {
		s.logger.Warn("recover failed", "err", err)
		s.writeError(w, http.StatusInternalServerError, "Recover failed: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.Toast(w, fmt.Sprintf("Recovered %d stuck order(s)", recovered)); err != nil {
		s.logger.Warn("render recover toast failed", "err", err)
	}
}

// handleAction wraps a zero-arg action that returns only an error.
func (s *Server) handleAction(action func(context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), routingTimeout)
		defer cancel()
		if err := action(ctx); err != nil {
			s.logger.Warn("action failed", "err", err)
			s.writeError(w, http.StatusInternalServerError, "Action failed: "+err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// writeError returns a toast fragment as the error body so the response-targets
// extension can swap it into #toast.
func (s *Server) writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	if err := s.renderer.Toast(w, msg); err != nil {
		s.logger.Warn("render error toast failed", "err", err)
	}
}
