package saga

import "context"

type Action[I, O any] func(ctx context.Context, fn I) (O, error)

type Mode uint8

const (
	SequentialMode Mode = iota
	ParallelMode
)

type Kind uint8

const (
	Task Kind = iota
	SequenceGroup
	ParallelGroup
)

type CompensationPlan[I, O any] struct {
	mode    Mode
	actions []Action[I, O]
}

type Node[I, O any] struct {
	name       string
	kind       Kind
	execute    Action[I, O]
	compensate CompensationPlan[I, O]
	children   []*Node[I, O]
}

func (n *Node[I, O]) Compensate(plan CompensationPlan[I, O]) *Node[I, O] {
	clone := *n
	clone.children = append([]*Node[I, O](nil), n.children...)
	clone.compensate = CompensationPlan[I, O]{
		mode:    plan.mode,
		actions: append([]Action[I, O](nil), plan.actions...),
	}

	return &clone
}

func group[I, O any](kind Kind, nodes ...*Node[I, O]) *Node[I, O] {
	return &Node[I, O]{
		kind:     kind,
		children: append([]*Node[I, O](nil), nodes...),
	}
}

func Do[I, O any](name string, fn Action[I, O]) *Node[I, O] {
	return &Node[I, O]{
		name:    name,
		kind:    Task,
		execute: fn,
	}
}

func Sequence[I, O any](nodes ...*Node[I, O]) *Node[I, O] {
	return group(SequenceGroup, nodes...)
}

func Parallel[I, O any](nodes ...*Node[I, O]) *Node[I, O] {
	return group(ParallelGroup, nodes...)
}

func SequentialCompensations[I, O any](
	actions ...Action[I, O],
) CompensationPlan[I, O] {
	return CompensationPlan[I, O]{
		mode:    SequentialMode,
		actions: append([]Action[I, O](nil), actions...),
	}
}

func ParallelCompensations[I, O any](
	actions ...Action[I, O],
) CompensationPlan[I, O] {
	return CompensationPlan[I, O]{
		mode:    ParallelMode,
		actions: append([]Action[I, O](nil), actions...),
	}
}

func Nested[I, O any](
	name string,
	nodes ...*Node[I, O],
) *Node[I, O] {
	node := group(SequenceGroup, nodes...)
	node.name = name
	return node
}
