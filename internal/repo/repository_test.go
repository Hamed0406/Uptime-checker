package repo_test

import (
	"testing"

	"github.com/hamed0406/uptimechecker/internal/repo"
	"github.com/hamed0406/uptimechecker/internal/repo/memory"
	pg "github.com/hamed0406/uptimechecker/internal/repo/postgres"
)

// Compile-time interface satisfaction checks.
// Using external test package avoids import cycle.
func TestInterfaceSatisfaction(t *testing.T) {
	var _ repo.TargetStore = memory.New()
	var _ repo.ResultStore = memory.New()

	// Postgres store types compile against the interfaces, too.
	var _ repo.TargetStore = (*pg.Store)(nil)
	var _ repo.ResultStore = (*pg.Store)(nil)
}
