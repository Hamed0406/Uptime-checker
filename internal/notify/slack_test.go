package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSlack_OK(t *testing.T) {
	var got string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		_ = json.NewDecoder(r.Body).Decode(&payload)
		got = payload["text"]
		w.WriteHeader(200)
	}))
	defer ts.Close()

	s := NewSlack(ts.URL)
	if s == nil {
		t.Fatal("expected slack client")
	}
	err := s.Send(context.Background(), "Title", "Hello")
	if err != nil {
		t.Fatalf("send err: %v", err)
	}
	if got == "" || got[0] != '*' { // starts with "*Title*"
		t.Fatalf("payload not as expected: %q", got)
	}
}

func TestSlack_Non2xx(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer ts.Close()

	s := NewSlack(ts.URL)
	err := s.Send(context.Background(), "X", "Y")
	if err == nil {
		t.Fatalf("expected error on non-2xx")
	}
}
