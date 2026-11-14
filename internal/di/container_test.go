package di_test

import (
	"testing"

	"hw2kpo/internal/di"
)

func TestSingletonResolutionAndCycleDetection(t *testing.T) {
	c := di.New()
	calls := 0
	if err := di.Provide[*int](c, func(*di.Container) (*int, error) {
		calls++
		value := 42
		return &value, nil
	}); err != nil {
		t.Fatal(err)
	}
	a, err := di.Resolve[*int](c)
	if err != nil {
		t.Fatal(err)
	}
	b, err := di.Resolve[*int](c)
	if err != nil || a != b || calls != 1 {
		t.Fatal("provider was not cached", err)
	}
	if _, err := di.Resolve[string](c); err == nil {
		t.Fatal("missing registration was accepted")
	}
	if err := di.Provide[string](c, func(c *di.Container) (string, error) { return di.Resolve[string](c) }); err != nil {
		t.Fatal(err)
	}
	if _, err := di.Resolve[string](c); err == nil {
		t.Fatal("dependency cycle was not detected")
	}
}
