package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAdmin_AllowsAdminKey_BlocksPublicKey(t *testing.T) {
	keys := Keys{
		Public: []string{"pub_key"},
		Admin:  []string{"adm_key"},
	}

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Admin key -> 200
	reqAdm := httptest.NewRequest(http.MethodGet, "/admin", nil)
	reqAdm.Header.Set("X-API-Key", "adm_key")
	recAdm := httptest.NewRecorder()
	RequireAdmin(keys)(okHandler).ServeHTTP(recAdm, reqAdm)
	if recAdm.Code != http.StatusOK {
		t.Fatalf("admin key should pass; got %d", recAdm.Code)
	}

	// Public key -> 403
	reqPub := httptest.NewRequest(http.MethodGet, "/admin", nil)
	reqPub.Header.Set("X-API-Key", "pub_key")
	recPub := httptest.NewRecorder()
	RequireAdmin(keys)(okHandler).ServeHTTP(recPub, reqPub)
	if recPub.Code != http.StatusForbidden {
		t.Fatalf("public key should be forbidden; got %d", recPub.Code)
	}

	// Missing key -> 401 (optional check; adjust if your middleware returns something else)
	reqNone := httptest.NewRequest(http.MethodGet, "/admin", nil)
	recNone := httptest.NewRecorder()
	RequireAdmin(keys)(okHandler).ServeHTTP(recNone, reqNone)
	if recNone.Code != http.StatusUnauthorized && recNone.Code != http.StatusForbidden {
		t.Fatalf("missing key should be 401/403; got %d", recNone.Code)
	}
}
