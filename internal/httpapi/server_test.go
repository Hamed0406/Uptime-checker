package httpapi

import "testing"

func TestIsValidHTTPURL(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"https://example.com", true},
		{"http://EXAMPLE.com", true},
		{"ftp://x", false},
		{"", false},
		{"https://", false},
	}
	for _, c := range cases {
		if got := isValidHTTPURL(c.in); got != c.want {
			t.Fatalf("isValidHTTPURL(%q)=%v want %v", c.in, got, c.want)
		}
	}
}

func TestNormalizeHTTPURL(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"https://EXAMPLE.com/", "https://example.com"},
		{"http://example.com:80", "http://example.com"},
		{"https://example.com:443/", "https://example.com"},
		{"https://example.com/p/", "https://example.com/p/"},
	}
	for _, c := range cases {
		if got := normalizeHTTPURL(c.in); got != c.want {
			t.Fatalf("normalizeHTTPURL(%q)=%q want %q", c.in, got, c.want)
		}
	}
}
