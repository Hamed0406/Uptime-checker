// cmd/preflight/main.go
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	fail := func(msg string) {
		fmt.Fprintln(os.Stderr, "✖", msg)
		os.Exit(1)
	}
	warn := func(msg string) { fmt.Fprintln(os.Stderr, "⚠", msg) }
	ok := func(msg string) { fmt.Println("✔", msg) }

	admin := strings.TrimSpace(os.Getenv("ADMIN_API_KEYS"))
	pub := strings.TrimSpace(os.Getenv("PUBLIC_API_KEYS"))
	apiAddr := strings.TrimSpace(os.Getenv("ADDR"))
	db := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	allowed := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))

	if admin == "" {
		fail("ADMIN_API_KEYS is empty (admin routes will 403).")
	}
	if pub == "" {
		fail("PUBLIC_API_KEYS is empty (read routes will 401).")
	}

	// Normalize and sanity-check lists (no spaces around commas).
	for name, v := range map[string]string{"ADMIN_API_KEYS": admin, "PUBLIC_API_KEYS": pub} {
		if strings.Contains(v, " ") {
			warn(name + " contains spaces; use comma-separated with no spaces, e.g. key1,key2")
		}
	}

	if apiAddr == "" {
		warn("ADDR is empty; default in your app may be used.")
	} else {
		ok("ADDR=" + apiAddr)
	}

	if db == "" {
		warn("DATABASE_URL empty — API will use in-memory stores unless overridden at runtime.")
	} else {
		ok("DATABASE_URL present")
	}

	if allowed == "" {
		warn("ALLOWED_ORIGINS empty — browser will be blocked by CORS for cross-origin requests.")
	} else {
		ok("ALLOWED_ORIGINS=" + allowed)
	}

	ok("preflight passed")
}
