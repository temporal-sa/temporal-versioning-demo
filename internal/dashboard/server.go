package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

const (
	routingTimeout = 10 * time.Second
	recoverTimeout = 30 * time.Second
	keepAlive      = 20 * time.Second
)

// Server exposes the dashboard API and serves the SPA.
type Server struct {
	hub      *Hub
	actions  *Actions
	frontend http.Handler
	logger   *slog.Logger
}

// NewServer builds a Server serving the SPA from frontendDir.
func NewServer(hub *Hub, actions *Actions, frontendDir string, logger *slog.Logger) *Server {
	return &Server{hub: hub, actions: actions, frontend: http.FileServer(http.Dir(frontendDir)), logger: logger}
}

// Routes returns the configured HTTP handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /api/state", s.handleState)
	mux.HandleFunc("GET /events", s.handleSSE)
	mux.HandleFunc("POST /api/ramp", s.handleRamp)
	mux.HandleFunc("POST /api/promote", s.handleAction(s.actions.Promote))
	mux.HandleFunc("POST /api/rollback", s.handleAction(s.actions.Rollback))
	mux.HandleFunc("POST /api/recover", s.handleRecover)
	mux.Handle("/", s.frontend)
	return mux
}

func (s *Server) handleState(w http.ResponseWriter, _ *http.Request) {
	state, ok := s.hub.Latest()
	w.Header().Set("Content-Type", "application/json")
	if !ok {
		_, _ = w.Write([]byte("{}"))
		return
	}
	if err := json.NewEncoder(w).Encode(state); err != nil {
		s.logger.Warn("encode state failed", "err", err)
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
			data, err := json.Marshal(state)
			if err != nil {
				s.logger.Warn("marshal SSE frame failed", "err", err)
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (s *Server) handleRamp(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Percentage float32 `json:"percentage"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Percentage < 0 || body.Percentage > 100 {
		s.writeError(w, http.StatusBadRequest, "percentage must be between 0 and 100")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), routingTimeout)
	defer cancel()
	if err := s.actions.Ramp(ctx, body.Percentage); err != nil {
		s.logger.Warn("ramp failed", "err", err)
		s.writeError(w, http.StatusInternalServerError, err.Error())
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
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]int{"recovered": recovered}); err != nil {
		s.logger.Warn("encode recover response failed", "err", err)
	}
}

// handleAction wraps a zero-arg action that returns only an error.
func (s *Server) handleAction(action func(context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), routingTimeout)
		defer cancel()
		if err := action(ctx); err != nil {
			s.logger.Warn("action failed", "err", err)
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": msg}); err != nil {
		s.logger.Warn("encode error response failed", "err", err)
	}
}
