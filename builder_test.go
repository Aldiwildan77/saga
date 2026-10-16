package saga

import (
	"context"
	"testing"
)

func TestBuildSequence(t *testing.T) {
	taskNameReserve := "reserve"
	taskNameCharge := "charge"

	firstTask := Do(
		taskNameReserve,
		func(context.Context, struct{}) (any, error) {
			return nil, nil
		},
	)

	secondTask := Do(taskNameCharge, func(context.Context, struct{}) (any, error) {
		return nil, nil
	})

	nodes := []*Node[struct{}, any]{firstTask, secondTask}
	flow := Sequence(nodes...)

	if flow.kind != SequenceGroup {
		t.Fatal("expected sequence group")
	}

	if len(flow.children) != len(nodes) {
		t.Fatalf("expected %d children, got %d", len(nodes), len(flow.children))
	}

	if flow.children[0].name != taskNameReserve {
		t.Fatal("unexpected first step")
	}
}

func TestBuildParallel(t *testing.T) {
	taskNameReserve := "reserve"
	taskNameCharge := "charge"

	firstTask := Do(taskNameReserve, func(context.Context, struct{}) (any, error) {
		return nil, nil
	})

	secondTask := Do(taskNameCharge, func(context.Context, struct{}) (any, error) {
		return nil, nil
	})

	nodes := []*Node[struct{}, any]{firstTask, secondTask}
	flow := Parallel(nodes...)

	if flow.kind != ParallelGroup {
		t.Fatal("expected parallel group")
	}

	if len(flow.children) != len(nodes) {
		t.Fatalf("expected %d children, got %d", len(nodes), len(flow.children))
	}

	if flow.children[0].name != taskNameReserve {
		t.Fatal("unexpected first step")
	}
}
