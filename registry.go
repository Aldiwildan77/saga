package saga

import (
	"fmt"
	"sync"
)

// Registry stores reusable task functions by name. The zero value is ready to use.
// It is safe for concurrent use.
type Registry struct {
	mu    sync.RWMutex
	funcs map[string]any
}

func NewRegistry() *Registry {
	return &Registry{funcs: make(map[string]any)}
}

// Register adds fn under name. Each name can be registered only once.
func Register[I, O any](r *Registry, name string, fn Action[I, O]) error {
	if r == nil {
		return ErrorRegistryNil
	}

	if name == "" {
		return ErrorRegistryName
	}

	if fn == nil {
		return ErrorRegistryNilFunc
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.funcs == nil {
		r.funcs = make(map[string]any)
	}

	if _, ok := r.funcs[name]; ok {
		return fmt.Errorf("%w: %q", ErrorRegistryDuplicate, name)
	}

	r.funcs[name] = fn
	return nil
}

// Resolve returns the function registered under name with the given input and output types.
func Resolve[I, O any](r *Registry, name string) (Action[I, O], error) {
	if r == nil {
		return nil, ErrorRegistryNil
	}

	r.mu.RLock()
	value, ok := r.funcs[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrorRegistryNotFound, name)
	}

	fn, ok := value.(Action[I, O])
	if !ok {
		return nil, fmt.Errorf("%w: %q is %T", ErrorRegistryTypeMismatch, name, value)
	}

	return fn, nil
}

// DoFrom builds a task named name from the function registered under that name.
func DoFrom[I, O any](r *Registry, name string) (*Node[I, O], error) {
	fn, err := Resolve[I, O](r, name)
	if err != nil {
		return nil, err
	}

	return Do(name, fn), nil
}
