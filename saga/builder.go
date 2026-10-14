package saga

import "context"

type Action[C any] func(ctx context.Context, fn C) error

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

type CompensationPlan[C any] struct {
	mode    Mode
	actions []Action[C]
}

type Node[C any] struct {
	name       string
	kind       Kind
	execute    Action[C]
	compensate CompensationPlan[C]
	children   []*Node[C]
}

func (n *Node[C]) Compensate(plan CompensationPlan[C]) *Node[C] {
	clone := *n
	clone.children = append([]*Node[C](nil), n.children...)
	clone.compensate = CompensationPlan[C]{
		mode:    plan.mode,
		actions: append([]Action[C](nil), plan.actions...),
	}

	return &clone
}

func group[C any](kind Kind, nodes ...*Node[C]) *Node[C] {
	return &Node[C]{
		kind:     kind,
		children: append([]*Node[C](nil), nodes...),
	}
}

func Do[C any](name string, fn Action[C]) *Node[C] {
	return &Node[C]{
		name:    name,
		kind:    Task,
		execute: fn,
	}
}

func Sequence[C any](nodes ...*Node[C]) *Node[C] {
	return group(SequenceGroup, nodes...)
}

func Parallel[C any](nodes ...*Node[C]) *Node[C] {
	return group(ParallelGroup, nodes...)
}

func SequentialCompensations[C any](
	actions ...Action[C],
) CompensationPlan[C] {
	return CompensationPlan[C]{
		mode:    SequentialMode,
		actions: append([]Action[C](nil), actions...),
	}
}

func ParallelCompensations[C any](
	actions ...Action[C],
) CompensationPlan[C] {
	return CompensationPlan[C]{
		mode:    ParallelMode,
		actions: append([]Action[C](nil), actions...),
	}
}

func Nested[C any](
	name string,
	nodes ...*Node[C],
) *Node[C] {
	node := group(SequenceGroup, nodes...)
	node.name = name
	return node
}
