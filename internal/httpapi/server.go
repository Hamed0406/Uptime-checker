package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"go.uber.org/zap"

	"github.com/hamed0406/uptimechecker/internal/domain"
	"github.com/hamed0406/uptimechecker/internal/probe"
	"github.com/hamed0406/uptimechecker/internal/repo"
)

type Server struct {
	Logger  *zap.Logger
	Targets repo.TargetStore
	Results repo.ResultStore
	Checker probe.Checker
}

func NewServer(l *zap.Logger, ts repo.TargetStore, rs repo.ResultStore, c probe.Checker) *Server {
	return &Server{Logger: l, Targets: ts, Results: rs, Checker: c}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(cors.AllowAll().Handler)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Get("/api/targets", s.handleListTargets)
	r.Post("/api/targets", s.handleAddTarget)
	r.Get("/api/results/latest", s.handleLatest)

	return r
}

type addPayload struct {
	URL string `json:"url"`
}

func (s *Server) handleAddTarget(w http.ResponseWriter, r *http.Request) {
	var p addPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil || p.URL == "" {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}

	t := &domain.Target{URL: p.URL, CreatedAt: time.Now().UTC()}
	if err := s.Targets.Add(r.Context(), t); err != nil {
		http.Error(w, "could not add", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	out := s.Checker.Check(ctx, p.URL)

	cr := &domain.CheckResult{
		TargetID:   t.ID,
		Up:         out.Success,
		HTTPStatus: 0, // not provided by current checker
		LatencyMS:  out.LatencyMS,
		Reason:     out.Message,
		CheckedAt:  time.Now().UTC(),
	}
	_ = s.Results.Append(r.Context(), cr)

	s.Logger.Info("added_target",
		zap.String("url", p.URL),
		zap.Bool("up", out.Success),
		zap.Float64("latency_ms", out.LatencyMS),
		zap.String("reason", out.Message),
	)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"target":  t,
		"summary": cr,
	})
}

func (s *Server) handleListTargets(w http.ResponseWriter, r *http.Request) {
	ts, err := s.Targets.List(r.Context())
	if err != nil {
		http.Error(w, "list error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ts)
}

func (s *Server) handleLatest(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Results.Latest(r.Context())
	if err != nil {
		http.Error(w, "latest error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rows)
}
