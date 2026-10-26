package saga

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func noop(name string) *Node[struct{}, struct{}] {
	return Do(name, func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	})
}

func TestVisualizeNestedSequenceAndParallel(t *testing.T) {
	flow := Sequence(
		noop("validate"),
		Parallel(
			noop("charge"),
			noop("ship"),
		),
		Nested("finalize",
			noop("confirm"),
		),
	)

	got, err := Visualize(flow)
	if err != nil {
		t.Fatal(err)
	}

	want := "sequence\n" +
		"├── validate\n" +
		"├── parallel\n" +
		"│   ├── charge\n" +
		"│   └── ship\n" +
		"└── nested finalize\n" +
		"    └── confirm\n"

	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestVisualizeSingleTask(t *testing.T) {
	got, err := Visualize(noop("reserve"))
	if err != nil {
		t.Fatal(err)
	}

	if got != "reserve\n" {
		t.Fatalf("got %q", got)
	}
}

func TestVisualizeShowsCompensation(t *testing.T) {
	seq := noop("reserve").Compensate(SequentialCompensations(
		func(context.Context, struct{}) (any, error) { return nil, nil },
		func(context.Context, struct{}) (any, error) { return nil, nil },
	))

	par := noop("charge").Compensate(ParallelCompensations(
		func(context.Context, struct{}) (any, error) { return nil, nil },
	))

	got, err := Visualize(Sequence(seq, par))
	if err != nil {
		t.Fatal(err)
	}

	want := "sequence\n" +
		"├── reserve [compensate: sequential x2]\n" +
		"└── charge [compensate: parallel x1]\n"

	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestVisualizeDoesNotExecuteTasks(t *testing.T) {
	var ran atomic.Int32

	flow := Sequence(
		Do("step", func(context.Context, struct{}) (struct{}, error) {
			ran.Add(1)
			return struct{}{}, nil
		}),
	)

	if _, err := Visualize(flow); err != nil {
		t.Fatal(err)
	}

	if ran.Load() != 0 {
		t.Fatal("visualize must not run tasks")
	}
}

func TestVisualizeNilRoot(t *testing.T) {
	_, err := Visualize(nil)

	if !errors.Is(err, ErrorRootEmpty) {
		t.Fatalf("expected root empty error, got %v", err)
	}
}

func TestVisualizeTaskWithoutExecuteFunc(t *testing.T) {
	_, err := Visualize(Sequence(Do[struct{}, struct{}]("missing", nil)))

	// ErrorNodeNoExecuteFunc builds a new error per call, so compare messages.
	want := ErrorNodeNoExecuteFunc("missing").Error()
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %q", err, want)
	}
}

func TestVisualizeUnsupportedKind(t *testing.T) {
	node := &Node[struct{}, struct{}]{name: "bogus", kind: Kind(99)}

	_, err := Visualize(node)

	if !errors.Is(err, ErrorNoneKind) {
		t.Fatalf("expected unsupported kind error, got %v", err)
	}
}

func TestVisualizeUnsupportedCompensationMode(t *testing.T) {
	node := noop("reserve").Compensate(CompensationPlan[struct{}]{
		mode: Mode(99),
		actions: []Action[struct{}, any]{
			func(context.Context, struct{}) (any, error) { return nil, nil },
		},
	})

	_, err := Visualize(node)

	if !errors.Is(err, ErrorNoneMode) {
		t.Fatalf("expected unsupported mode error, got %v", err)
	}
}
