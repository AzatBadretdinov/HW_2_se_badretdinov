
package di

import (
	"fmt"
	"reflect"
)

type entry struct {
	provider  func(*Container) (any, error)
	value     any
	ready     bool
	resolving bool
}

type Container struct{ entries map[reflect.Type]*entry }

func New() *Container { return &Container{entries: make(map[reflect.Type]*entry)} }

func typeOf[T any]() reflect.Type { return reflect.TypeOf((*T)(nil)).Elem() }

func Provide[T any](c *Container, provider func(*Container) (T, error)) error {
	t := typeOf[T]()
	if _, ok := c.entries[t]; ok {
		return fmt.Errorf("DI: тип %s уже зарегистрирован", t)
	}
	c.entries[t] = &entry{provider: func(c *Container) (any, error) { return provider(c) }}
	return nil
}

func Resolve[T any](c *Container) (T, error) {
	var zero T
	t := typeOf[T]()
	e, ok := c.entries[t]
	if !ok {
		return zero, fmt.Errorf("DI: тип %s не зарегистрирован", t)
	}
	if e.resolving {
		return zero, fmt.Errorf("DI: циклическая зависимость при разрешении %s", t)
	}
	if !e.ready {
		e.resolving = true
		defer func() { e.resolving = false }()
		value, err := e.provider(c)
		if err != nil {
			return zero, fmt.Errorf("DI %s: %w", t, err)
		}
		e.value, e.ready = value, true
	}
	value, ok := e.value.(T)
	if !ok {
		return zero, fmt.Errorf("DI: провайдер %s вернул нетипизированный nil", t)
	}
	return value, nil
}
