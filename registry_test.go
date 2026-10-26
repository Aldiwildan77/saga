package saga

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func double(_ context.Context, n int) (int, error) {
	return n * 2, nil
}

func TestRegisterResolveAndRun(t *testing.T) {
	reg := NewRegistry()

	if err := Register(reg, "double", Action[int, int](double)); err != nil {
		t.Fatal(err)
	}

	var got int

	node, err := DoFrom[int, int](reg, "double")
	if err != nil {
		t.Fatal(err)
	}

	flow := Sequence(
		node,
		Do("capture", func(_ context.Context, n int) (int, error) {
			got = n
			return n, nil
		}),
	)

	if err := Run(context.Background(), flow, 21); err != nil {
		t.Fatal(err)
	}

	if got != 42 {
		t.Fatalf("got %d, want 42", got)
	}
}

func TestZeroValueRegistryIsUsable(t *testing.T) {
	var reg Registry

	if err := Register(&reg, "double", Action[int, int](double)); err != nil {
		t.Fatal(err)
	}

	if _, err := Resolve[int, int](&reg, "double"); err != nil {
		t.Fatal(err)
	}
}

func TestRegisterNilRegistry(t *testing.T) {
	err := Register[int, int](nil, "double", double)

	if !errors.Is(err, ErrorRegistryNil) {
		t.Fatalf("expected registry nil error, got %v", err)
	}
}

func TestRegisterEmptyName(t *testing.T) {
	err := Register(NewRegistry(), "", Action[int, int](double))

	if !errors.Is(err, ErrorRegistryName) {
		t.Fatalf("expected name error, got %v", err)
	}
}

func TestRegisterNilFunc(t *testing.T) {
	err := Register[int, int](NewRegistry(), "double", nil)

	if !errors.Is(err, ErrorRegistryNilFunc) {
		t.Fatalf("expected nil func error, got %v", err)
	}
}

func TestRegisterDuplicate(t *testing.T) {
	reg := NewRegistry()

	if err := Register(reg, "double", Action[int, int](double)); err != nil {
		t.Fatal(err)
	}

	err := Register(reg, "double", Action[int, int](double))

	if !errors.Is(err, ErrorRegistryDuplicate) {
		t.Fatalf("expected duplicate error, got %v", err)
	}

	if !strings.Contains(err.Error(), `"double"`) {
		t.Fatalf("expected name in error, got %v", err)
	}
}

func TestResolveNilRegistry(t *testing.T) {
	_, err := Resolve[int, int](nil, "double")

	if !errors.Is(err, ErrorRegistryNil) {
		t.Fatalf("expected registry nil error, got %v", err)
	}
}

func TestResolveNotFound(t *testing.T) {
	_, err := Resolve[int, int](NewRegistry(), "missing")

	if !errors.Is(err, ErrorRegistryNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}

	if !strings.Contains(err.Error(), `"missing"`) {
		t.Fatalf("expected name in error, got %v", err)
	}
}

func TestResolveTypeMismatch(t *testing.T) {
	reg := NewRegistry()

	if err := Register(reg, "double", Action[int, int](double)); err != nil {
		t.Fatal(err)
	}

	_, err := Resolve[string, string](reg, "double")

	if !errors.Is(err, ErrorRegistryTypeMismatch) {
		t.Fatalf("expected type mismatch error, got %v", err)
	}
}

func TestDoFromPropagatesResolveError(t *testing.T) {
	node, err := DoFrom[int, int](NewRegistry(), "missing")

	if !errors.Is(err, ErrorRegistryNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}

	if node != nil {
		t.Fatal("expected nil node on error")
	}
}

func TestDoFromNilRegistry(t *testing.T) {
	_, err := DoFrom[int, int](nil, "double")

	if !errors.Is(err, ErrorRegistryNil) {
		t.Fatalf("expected registry nil error, got %v", err)
	}
}

func TestRegistryConcurrentUse(t *testing.T) {
	reg := NewRegistry()

	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(2)

		name := fmt.Sprintf("fn-%d", i)

		go func() {
			defer wg.Done()
			_ = Register(reg, name, Action[int, int](double))
		}()

		go func() {
			defer wg.Done()
			_, _ = Resolve[int, int](reg, name)
		}()
	}

	wg.Wait()

	for i := range 50 {
		if _, err := Resolve[int, int](reg, fmt.Sprintf("fn-%d", i)); err != nil {
			t.Fatalf("fn-%d: %v", i, err)
		}
	}
}
