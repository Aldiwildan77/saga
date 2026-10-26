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

	nodes := []nodeRunner{firstTask, secondTask}
	flow := Sequence(nodes...)

	if flow.kind != SequenceGroup {
		t.Fatal("expected sequence group")
	}

	if len(flow.children) != len(nodes) {
		t.Fatalf("expected %d children, got %d", len(nodes), len(flow.children))
	}

	if flow.children[0].nodeName() != taskNameReserve {
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

	nodes := []nodeRunner{firstTask, secondTask}
	flow := Parallel(nodes...)

	if flow.kind != ParallelGroup {
		t.Fatal("expected parallel group")
	}

	if len(flow.children) != len(nodes) {
		t.Fatalf("expected %d children, got %d", len(nodes), len(flow.children))
	}

	if flow.children[0].nodeName() != taskNameReserve {
		t.Fatal("unexpected first step")
	}
}

func TestBuildNestedNamesGroup(t *testing.T) {
	inner := Do("reserve", func(context.Context, struct{}) (any, error) {
		return nil, nil
	})

	group := Nested("fulfillment", inner)

	if group.kind != SequenceGroup {
		t.Fatal("expected nested group to be a sequence")
	}

	if group.nodeName() != "fulfillment" {
		t.Fatalf("got name %q, want %q", group.nodeName(), "fulfillment")
	}

	if len(group.children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(group.children))
	}
}

func TestCompensateReturnsCopy(t *testing.T) {
	base := Do("reserve", func(context.Context, struct{}) (any, error) {
		return nil, nil
	})

	plan := SequentialCompensations(
		func(context.Context, struct{}) (any, error) {
			return nil, nil
		},
	)

	compensated := base.Compensate(plan)

	if compensated == base {
		t.Fatal("Compensate should return a new node")
	}

	if len(base.compensate.actions) != 0 {
		t.Fatal("original node should not gain compensation actions")
	}

	if len(compensated.compensate.actions) != 1 {
		t.Fatalf("expected 1 compensation action, got %d", len(compensated.compensate.actions))
	}
}

func TestCompensationPlanModes(t *testing.T) {
	action := func(context.Context, struct{}) (any, error) { return nil, nil }

	if SequentialCompensations(action).mode != SequentialMode {
		t.Fatal("expected sequential mode")
	}

	if ParallelCompensations(action).mode != ParallelMode {
		t.Fatal("expected parallel mode")
	}
}
