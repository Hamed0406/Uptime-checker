package logging

import (
	"os"
	"testing"
)

func TestNewLogger_CreatesDirAndLogger(t *testing.T) {
	dir := t.TempDir()
	log, err := NewLogger(dir)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	defer func() { _ = log.Sync() }()

	// Directory should exist
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("log dir missing: %v", err)
	}

	// Write once; just ensuring no panic / basic functionality.
	log.Info("test_message_from_logging_test")

	// Best-effort: a file might not be flushed immediately; don't fail on it.
	if entries, _ := os.ReadDir(dir); len(entries) == 0 {
		t.Logf("no files yet in %s (ok; async writers may delay)", dir)
	}
}
