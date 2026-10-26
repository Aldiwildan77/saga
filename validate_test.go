package saga

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
)

func okTask(name string) *Node[struct{}, struct{}] {
	return Do(name, func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	})
}

func TestValidateValidFlow(t *testing.T) {
	flow := Sequence(
		okTask("validate"),
		Parallel(okTask("charge"), okTask("ship")),
		Nested("finalize", okTask("confirm")),
	)

	if err := Validate(flow); err != nil {
		t.Fatalf("expected valid flow, got %v", err)
	}
}

func TestValidateNilRoot(t *testing.T) {
	if err := Validate(nil); !errors.Is(err, ErrorRootEmpty) {
		t.Fatalf("expected root empty, got %v", err)
	}
}

func TestValidateTaskWithoutExecuteFunc(t *testing.T) {
	err := Validate(Sequence(Do[struct{}, struct{}]("missing", nil)))

	want := ErrorNodeNoExecuteFunc("missing").Error()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("got %v, want it to contain %q", err, want)
	}
}

func TestValidateEmptySequence(t *testing.T) {
	err := Validate(Sequence())

	if !errors.Is(err, ErrorGroupEmpty) {
		t.Fatalf("expected empty group error, got %v", err)
	}

	if !strings.Contains(err.Error(), "sequence") {
		t.Fatalf("expected group name in error, got %v", err)
	}
}

func TestValidateEmptyParallel(t *testing.T) {
	err := Validate(Parallel())

	if !errors.Is(err, ErrorGroupEmpty) {
		t.Fatalf("expected empty group error, got %v", err)
	}

	if !strings.Contains(err.Error(), "parallel") {
		t.Fatalf("expected parallel in error, got %v", err)
	}
}

func TestValidateEmptyNestedNamesGroup(t *testing.T) {
	err := Validate(Sequence(Nested("prepare-order")))

	if !errors.Is(err, ErrorGroupEmpty) {
		t.Fatalf("expected empty group error, got %v", err)
	}

	if !strings.Contains(err.Error(), "prepare-order") {
		t.Fatalf("expected nested name in error, got %v", err)
	}
}

func TestValidateNilChild(t *testing.T) {
	err := Validate(Sequence(okTask("first"), nil))

	if !errors.Is(err, ErrorNilChild) {
		t.Fatalf("expected nil child error, got %v", err)
	}
}

func TestValidateUnsupportedKind(t *testing.T) {
	node := &Node[struct{}, struct{}]{name: "bogus", kind: Kind(99)}

	if err := Validate(node); !errors.Is(err, ErrorNoneKind) {
		t.Fatalf("expected unsupported kind, got %v", err)
	}
}

func TestValidateUnsupportedCompensationMode(t *testing.T) {
	node := okTask("reserve").Compensate(CompensationPlan[struct{}]{
		mode: Mode(99),
		actions: []Action[struct{}, any]{
			func(context.Context, struct{}) (any, error) { return nil, nil },
		},
	})

	if err := Validate(node); !errors.Is(err, ErrorNoneMode) {
		t.Fatalf("expected unsupported mode, got %v", err)
	}
}

func TestValidateNilCompensationAction(t *testing.T) {
	node := okTask("reserve").Compensate(SequentialCompensations(
		func(context.Context, struct{}) (any, error) { return nil, nil },
		nil,
	))

	err := Validate(node)

	if !errors.Is(err, ErrorCompensationNilFunc) {
		t.Fatalf("expected nil compensation error, got %v", err)
	}

	if !strings.Contains(err.Error(), "reserve") {
		t.Fatalf("expected task name in error, got %v", err)
	}
}

func TestValidateReportsAllProblems(t *testing.T) {
	flow := Sequence(
		Do[struct{}, struct{}]("missing", nil),
		Parallel(),
		Nested("empty"),
	)

	err := Validate(flow)

	for _, want := range []error{
		ErrorNodeNoExecuteFunc("missing"),
		ErrorGroupEmpty,
	} {
		if !strings.Contains(err.Error(), want.Error()) && !errors.Is(err, want) {
			t.Fatalf("expected %q in joined error, got %v", want, err)
		}
	}

	if strings.Count(err.Error(), "has no children") < 2 {
		t.Fatalf("expected both empty groups reported, got %v", err)
	}
}

func TestValidateDoesNotRunTasks(t *testing.T) {
	var ran atomic.Int32

	flow := Sequence(Do("step", func(context.Context, struct{}) (struct{}, error) {
		ran.Add(1)
		return struct{}{}, nil
	}))

	if err := Validate(flow); err != nil {
		t.Fatal(err)
	}

	if ran.Load() != 0 {
		t.Fatal("validate must not run tasks")
	}
}
