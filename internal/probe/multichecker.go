package probe

import "context"

type MultiChecker struct {
	Checkers []Checker
}

func NewMultiChecker(checkers ...Checker) *MultiChecker {
	return &MultiChecker{Checkers: checkers}
}

func (m *MultiChecker) Run(ctx context.Context, target string) []CheckResult {
	results := make([]CheckResult, 0, len(m.Checkers))
	for _, c := range m.Checkers {
		results = append(results, c.Check(ctx, target))
	}
	return results
}
