package dashboard

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/alexandreroman/temporal-versioning-demo/internal/pizza"
)

const (
	routingTimeout = 10 * time.Second
	recoverTimeout = 30 * time.Second
	keepAlive      = 20 * time.Second
)

// sseRegions are the named regions streamed to the browser on every frame.
var sseRegions = []string{"orders", "versions", "controls"}

// Server exposes the dashboard API and serves the SPA.
type Server struct {
	hub       *Hub
	actions   *Actions
	renderer  *Renderer
	frontend  http.Handler
	generator *GeneratorControl
	logger    *slog.Logger
}

// NewServer builds a Server serving the SPA from the given file system (the
// embedded frontend assets in production).
func NewServer(
	hub *Hub,
	actions *Actions,
	renderer *Renderer,
	frontend fs.FS,
	generator *GeneratorControl,
	logger *slog.Logger,
) *Server {
	return &Server{
		hub:       hub,
		actions:   actions,
		renderer:  renderer,
		frontend:  http.FileServerFS(frontend),
		generator: generator,
		logger:    logger,
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
	mux.HandleFunc("GET /deploy", s.handleDeployModal)
	mux.HandleFunc("GET /rollback", s.handleRollbackModal)
	mux.HandleFunc("GET /deploy/ramp", s.handleDeployRamp)
	mux.HandleFunc("DELETE /modal", s.handleClose)
	mux.HandleFunc("POST /deploy", s.handleDeploy)
	mux.HandleFunc("POST /rollback", s.handleRollback)
	mux.HandleFunc("POST /orders/play", s.handleOrdersPlay)
	mux.HandleFunc("POST /orders/pause", s.handleOrdersPause)
	mux.HandleFunc("POST /orders/{id}/recover", s.handleRecoverOne)
	mux.Handle("/", s.frontend)
	return mux
}

func (s *Server) currentState() DashboardState {
	state := s.hub.Latest()
	if s.generator != nil {
		state.Generator = s.generator.Status()
	}
	return state
}

func (s *Server) renderControls(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.Region(w, "controls", s.currentState()); err != nil {
		s.logger.Warn("render controls failed", "err", err)
		s.writeError(w, http.StatusInternalServerError, "Render failed: "+err.Error())
	}
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

// handleDeployModal renders the Deploy modal fragment from the latest state,
// pre-selecting the Ramping version (else Current, else first) with the ramp
// slider set to match its status.
func (s *Server) handleDeployModal(w http.ResponseWriter, _ *http.Request) {
	view := buildDeployModalView(s.hub.Latest())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.DeployModal(w, view); err != nil {
		s.logger.Warn("render deploy modal failed", "err", err)
		s.writeError(w, http.StatusInternalServerError, "Render failed: "+err.Error())
	}
}

// handleRollbackModal renders the rollback confirmation modal fragment.
func (s *Server) handleRollbackModal(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.RollbackModal(w); err != nil {
		s.logger.Warn("render rollback modal failed", "err", err)
		s.writeError(w, http.StatusInternalServerError, "Render failed: "+err.Error())
	}
}

// handleDeployRamp re-renders only the ramp section. A "stop" query param means
// the slider moved, so we honor the chosen stop; otherwise a radio changed and
// we derive the ramp from the selected version's deployment-card status.
func (s *Server) handleDeployRamp(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var view rampView
	if stop := q.Get("stop"); stop != "" {
		view = rampViewForStop(stop)
	} else {
		view = rampViewFor(s.hub.Latest(), q.Get("version"))
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.DeployRamp(w, view); err != nil {
		s.logger.Warn("render deploy ramp failed", "err", err)
		s.writeError(w, http.StatusInternalServerError, "Render failed: "+err.Error())
	}
}

// handleClose serves DELETE /modal: it empties #modal-host, closing whichever
// modal is open. The empty 200 body (not 204) is what htmx swaps in to clear
// the host.
func (s *Server) handleClose(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

// validVersion guards the friendly labels the UI may send.
func validVersion(v string) bool { _, ok := pizza.ParseVersion(v); return ok }

// handleDeploy applies the modal's selection: ramping to the chosen stop, or
// promoting the version to Current at 100%. On success it returns an empty 200
// body (not 204, which htmx would not swap) so htmx clears #modal-host, closing
// the modal; errors route to #toast and leave the modal open.
func (s *Server) handleDeploy(w http.ResponseWriter, r *http.Request) {
	version := r.FormValue("version")
	if !validVersion(version) {
		s.writeError(w, http.StatusBadRequest, "invalid version")
		return
	}
	_, pct, ok := parseStop(r.FormValue("stop"))
	if !ok {
		s.writeError(w, http.StatusBadRequest, "invalid ramp selection")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), routingTimeout)
	defer cancel()
	if pct == rampFullPct {
		if err := s.actions.Promote(ctx, version); err != nil {
			s.logger.Warn("promote failed", "err", err)
			s.writeError(w, http.StatusInternalServerError, "Promote failed: "+err.Error())
			return
		}
	} else {
		if err := s.actions.Ramp(ctx, version, float32(pct)); err != nil {
			s.logger.Warn("ramp failed", "err", err)
			s.writeError(w, http.StatusInternalServerError, "Ramp failed: "+err.Error())
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

// handleRollback reverts traffic to the previously Current version. Like
// handleDeploy, success is an empty 200 body that clears #modal-host.
func (s *Server) handleRollback(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), routingTimeout)
	defer cancel()
	if err := s.actions.Rollback(ctx); err != nil {
		s.logger.Warn("rollback failed", "err", err)
		s.writeError(w, http.StatusInternalServerError, "Rollback failed: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleOrdersPlay(w http.ResponseWriter, _ *http.Request) {
	if s.generator == nil {
		s.writeError(w, http.StatusInternalServerError, "Order generator is unavailable")
		return
	}
	s.generator.Play()
	s.renderControls(w)
}

func (s *Server) handleOrdersPause(w http.ResponseWriter, _ *http.Request) {
	if s.generator == nil {
		s.writeError(w, http.StatusInternalServerError, "Order generator is unavailable")
		return
	}
	s.generator.Pause()
	s.renderControls(w)
}

func (s *Server) handleRecoverOne(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx, cancel := context.WithTimeout(r.Context(), recoverTimeout)
	defer cancel()
	if err := s.actions.RecoverOne(ctx, id); err != nil {
		s.logger.Warn("recover failed", "workflowId", id, "err", err)
		s.writeError(w, http.StatusInternalServerError, "Recover failed: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderer.Toast(w, fmt.Sprintf("Recovered %s", id)); err != nil {
		s.logger.Warn("render recover toast failed", "err", err)
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
