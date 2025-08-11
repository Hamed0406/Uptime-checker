package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	Checker *probe.HTTPChecker
}

func NewServer(l *zap.Logger, ts repo.TargetStore, rs repo.ResultStore, c *probe.HTTPChecker) *Server {
	return &Server{Logger: l, Targets: ts, Results: rs, Checker: c}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(cors.AllowAll().Handler)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
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
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil || p.URL == "" {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}

	t := &domain.Target{URL: p.URL, CreatedAt: time.Now().UTC()}
	if err := s.Targets.Add(r.Context(), t); err != nil {
		http.Error(w, "could not add", http.StatusInternalServerError)
		return
	}

	// Run a single check synchronously for immediate feedback
	out := s.Checker.Check(p.URL)

	// If HTTP check fails, run DNS check
	dnsClass := ""
	if !out.Up {
		host := extractHost(p.URL)
		dns := probe.CheckDNS(host)
		dnsClass = dns.Class

		// Log detailed DNS info
		s.Logger.Info("dns_check",
			zap.String("domain", dns.Domain),
			zap.String("class", dns.Class),
			zap.Bool("has_a_or_aaaa", dns.HasAOrAAAA),
			zap.Strings("nameservers", dns.Nameservers),
			zap.String("cname", dns.CNAME),
			zap.String("resolver_error", dns.ResolverError),
		)
	}

	// Combine reason with DNS class (if any)
	reason := out.Reason
	if dnsClass != "" {
		reason = strings.TrimSpace(fmt.Sprintf("%s dns=%s", out.Reason, dnsClass))
	}

	cr := &domain.CheckResult{
		TargetID:   t.ID,
		Up:         out.Up,
		HTTPStatus: out.StatusCode,
		LatencyMS:  out.LatencyMS,
		Reason:     reason,
		CheckedAt:  time.Now().UTC(),
	}
	_ = s.Results.Append(r.Context(), cr)

	s.Logger.Info("added_target",
		zap.String("url", p.URL),
		zap.Bool("up", out.Up),
		zap.Float64("latency_ms", out.LatencyMS),
	)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"target": t, "result": cr,
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

// extractHost pulls the hostname from a URL string
func extractHost(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return raw
	}
	return u.Hostname()
}
