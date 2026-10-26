package saga

import (
	"context"
)

type Action[I, O any] func(ctx context.Context, input I) (O, error)

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

type nodeRunner interface {
	nodeKind() Kind
	nodeName() string
	run(context.Context, any) (*Record, error)
	compensateNode(context.Context, any) error
	describe() nodeDescription
}

type nodeDescription struct {
	name          string
	kind          Kind
	children      []nodeRunner
	executable    bool
	mode          Mode
	compensations int
	nilActions    int
}

func (n *Node[I, O]) describe() nodeDescription {
	nilActions := 0
	for _, action := range n.compensate.actions {
		if action == nil {
			nilActions++
		}
	}

	return nodeDescription{
		name:          n.name,
		kind:          n.kind,
		children:      n.children,
		executable:    n.execute != nil,
		mode:          n.compensate.mode,
		compensations: len(n.compensate.actions),
		nilActions:    nilActions,
	}
}

func (n *Node[I, O]) nodeKind() Kind {
	return n.kind
}

func (n *Node[I, O]) nodeName() string {
	return n.name
}

type CompensationPlan[I any] struct {
	mode    Mode
	actions []Action[I, any]
}

type Node[I, O any] struct {
	name       string
	kind       Kind
	execute    Action[I, O]
	compensate CompensationPlan[I]
	children   []nodeRunner
}

func (n *Node[I, O]) Compensate(plan CompensationPlan[I]) *Node[I, O] {
	clone := *n
	clone.children = append([]nodeRunner(nil), n.children...)
	clone.compensate = CompensationPlan[I]{
		mode:    plan.mode,
		actions: append([]Action[I, any](nil), plan.actions...),
	}

	return &clone
}

func group(kind Kind, nodes ...nodeRunner) *Node[any, any] {
	return &Node[any, any]{
		kind:     kind,
		children: append([]nodeRunner(nil), nodes...),
	}
}

func Do[I, O any](name string, fn Action[I, O]) *Node[I, O] {
	return &Node[I, O]{
		name:    name,
		kind:    Task,
		execute: fn,
	}
}

func Sequence(nodes ...nodeRunner) *Node[any, any] {
	return group(SequenceGroup, nodes...)
}

func Parallel(nodes ...nodeRunner) *Node[any, any] {
	return group(ParallelGroup, nodes...)
}

func SequentialCompensations[I any](
	actions ...Action[I, any],
) CompensationPlan[I] {
	return CompensationPlan[I]{
		mode:    SequentialMode,
		actions: append([]Action[I, any](nil), actions...),
	}
}

func ParallelCompensations[I any](
	actions ...Action[I, any],
) CompensationPlan[I] {
	return CompensationPlan[I]{
		mode:    ParallelMode,
		actions: append([]Action[I, any](nil), actions...),
	}
}

func Nested(
	name string,
	nodes ...nodeRunner,
) *Node[any, any] {
	node := group(SequenceGroup, nodes...)
	node.name = name
	return node
}
