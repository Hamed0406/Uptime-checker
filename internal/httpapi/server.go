package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"go.uber.org/zap"

	apimw "github.com/hamed0406/uptimechecker/internal/httpapi/middleware"

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

// Router wires CORS (from allowlist) and API-key auth.
// - Public/read routes require ANY key if keys are configured (else open in dev).
// - Admin/write routes require an ADMIN key if configured (else open in dev).
func (s *Server) Router(keys apimw.Keys, allowedOrigins []string) http.Handler {
	r := chi.NewRouter()

	// CORS: allowlist if provided; otherwise allow all (handy for local dev)
	if len(allowedOrigins) > 0 {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   allowedOrigins,
			AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: false,
			MaxAge:           300,
		}))
	} else {
		r.Use(cors.AllowAll().Handler)
	}

	// Public/read routes
	r.Group(func(pub chi.Router) {
		pub.Use(apimw.RequireAny(keys))

		pub.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})

		pub.Get("/api/targets", s.handleListTargets)
		pub.Get("/api/results/latest", s.handleLatest)
	})

	// Admin/write routes
	r.Group(func(adm chi.Router) {
		adm.Use(apimw.RequireAdmin(keys))
		adm.Post("/api/targets", s.handleAddTarget)
	})

	return r
}

type addPayload struct {
	URL string `json:"url"`
}

func (s *Server) handleAddTarget(w http.ResponseWriter, r *http.Request) {
	var p addPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	p.URL = strings.TrimSpace(p.URL)
	if !isValidHTTPURL(p.URL) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid url"})
		return
	}

	t := &domain.Target{URL: p.URL, CreatedAt: time.Now().UTC()}
	if err := s.Targets.Add(r.Context(), t); err != nil {
		// If you later want to map duplicate URL errors to 409, do it here.
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "could not add target"})
		return
	}

	// Probe once immediately (checker has its own timeout; we also guard with a context timeout)
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

	writeJSON(w, http.StatusOK, map[string]any{
		"target":  t,
		"summary": cr,
	})
}

func (s *Server) handleListTargets(w http.ResponseWriter, r *http.Request) {
	ts, err := s.Targets.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "list error"})
		return
	}
	writeJSON(w, http.StatusOK, ts)
}

func (s *Server) handleLatest(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Results.Latest(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "latest error"})
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// --- helpers ---

func isValidHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if u.Host == "" {
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
