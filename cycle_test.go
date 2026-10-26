package saga

import (
	"context"
	"errors"
	"testing"
)

func cycleTask(name string) *Node[struct{}, struct{}] {
	return Do(name, func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	})
}

func TestDetectCyclesValidFlow(t *testing.T) {
	flow := Sequence(
		cycleTask("validate"),
		Parallel(cycleTask("charge"), cycleTask("ship")),
		Nested("finalize", cycleTask("confirm")),
	)

	if err := DetectCycles(flow); err != nil {
		t.Fatalf("expected no cycle, got %v", err)
	}
}

func TestDetectCyclesNilRoot(t *testing.T) {
	if err := DetectCycles(nil); !errors.Is(err, ErrorRootEmpty) {
		t.Fatalf("expected root empty, got %v", err)
	}
}

func TestDetectCyclesSharedNodeIsNotCycle(t *testing.T) {
	shared := cycleTask("notify")

	// The same node appears in two sibling branches. That is reuse, not a cycle.
	flow := Sequence(
		Parallel(shared, cycleTask("other")),
		Nested("again", shared),
	)

	if err := DetectCycles(flow); err != nil {
		t.Fatalf("shared node should not be a cycle, got %v", err)
	}
}

func TestDetectCyclesSelfContainment(t *testing.T) {
	group := Sequence(cycleTask("step"))
	group.children = append(group.children, group)

	err := DetectCycles(group)

	if !errors.Is(err, ErrorCycle) {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestDetectCyclesDeepCycle(t *testing.T) {
	inner := Nested("inner", cycleTask("leaf"))
	outer := Sequence(inner)

	// inner contains outer, which contains inner: a cycle two levels down.
	inner.children = append(inner.children, outer)

	err := DetectCycles(outer)

	if !errors.Is(err, ErrorCycle) {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestDetectCyclesSkipsNilChild(t *testing.T) {
	flow := Sequence(cycleTask("first"), nil)

	if err := DetectCycles(flow); err != nil {
		t.Fatalf("nil child should be skipped, got %v", err)
	}
}
