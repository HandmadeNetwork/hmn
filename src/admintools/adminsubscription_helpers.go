package admintools

import (
	"fmt"
)

type scenarioCtx struct {
	name  string
	steps int
	index int
}

func newScenarioCtx(name string, steps int) *scenarioCtx {
	return &scenarioCtx{
		name:  name,
		steps: steps,
	}
}

func (s *scenarioCtx) step(msg string, fn func() error) error {
	s.index++
	fmt.Printf("[%d/%d] %s\n", s.index, s.steps, msg)
	return fn()
}

func (s *scenarioCtx) printf(format string, args ...any) {
	fmt.Printf("      "+format, args...)
}
