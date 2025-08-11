package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
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
	Checker *probe.MultiChecker
}

func NewServer(l *zap.Logger, ts repo.TargetStore, rs repo.ResultStore, c *probe.MultiChecker) *Server {
	return &Server{
		Logger:  l,
		Targets: ts,
		Results: rs,
		Checker: c,
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(cors.AllowAll().Handler)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Get("/api/targets", s.handleListTargets)
	r.Post("/api/targets", s.handleAddTarget)

	return r
}

type addPayload struct {
	URL string `json:"url"`
}

func (s *Server) handleAddTarget(w http.ResponseWriter, r *http.Request) {
	var p addPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil || strings.TrimSpace(p.URL) == "" {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}

	t := &domain.Target{URL: p.URL, CreatedAt: time.Now().UTC()}
	if err := s.Targets.Add(r.Context(), t); err != nil {
		http.Error(w, "could not add", http.StatusInternalServerError)
		return
	}

	// Run all registered checks (HTTP, DNS, ...).
	results := s.Checker.Run(context.Background(), p.URL)

	// Decide overall UP/DOWN: require HTTP success.
	up := false
	var httpLatency float64
	for _, res := range results {
		if res.Name == "HTTP" {
			up = res.Success
			httpLatency = res.LatencyMS
			break
		}
	}

	// Combine messages for a human-readable reason.
	var parts []string
	for _, res := range results {
		parts = append(parts, res.Name+"="+res.Message)
	}
	reason := strings.Join(parts, " ")

	cr := &domain.CheckResult{
		TargetID:  t.ID,
		Up:        up,
		LatencyMS: httpLatency, // HTTP latency if available
		Reason:    reason,
		CheckedAt: time.Now().UTC(),
	}
	_ = s.Results.Append(r.Context(), cr)

	s.Logger.Info("added_target",
		zap.String("url", p.URL),
		zap.Bool("up", up),
		zap.Float64("latency_ms", httpLatency),
		zap.String("reason", reason),
	)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"target":  t,
		"results": results, // detailed per-check results
		"summary": cr,      // overall interpretation
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
