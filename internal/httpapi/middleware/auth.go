package middleware

import (
	"net/http"
	"strings"
)

type Keys struct {
	Public []string
	Admin  []string
}

func readAuth(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	if k := r.Header.Get("X-API-Key"); k != "" {
		return strings.TrimSpace(k)
	}
	return ""
}

func hasKey(given string, set []string) bool {
	if given == "" || len(set) == 0 {
		return false
	}
	for _, k := range set {
		if k == given {
			return true
		}
	}
	return false
}

// RequireAny allows requests that present either a public or admin key.
// If no keys are configured, it allows all requests (handy for local dev).
func RequireAny(keys Keys) func(http.Handler) http.Handler {
	enabled := len(keys.Public) > 0 || len(keys.Admin) > 0
	return func(next http.Handler) http.Handler {
		if !enabled {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := readAuth(r)
			if hasKey(key, keys.Public) || hasKey(key, keys.Admin) {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
		})
	}
}

// RequireAdmin only permits requests that present an admin key.
// If no admin keys are configured, it allows all requests (dev).
func RequireAdmin(keys Keys) func(http.Handler) http.Handler {
	enabled := len(keys.Admin) > 0
	return func(next http.Handler) http.Handler {
		if !enabled {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := readAuth(r)
			if hasKey(key, keys.Admin) {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"forbidden"}`))
		})
	}
}
