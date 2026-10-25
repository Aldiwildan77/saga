package saga

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
)

func (n *Node[I, O]) compensateNode(
	ctx context.Context,
	input any,
) error {
	if len(n.compensate.actions) == 0 {
		return nil
	}

	typedInput, ok := input.(I)
	if !ok {
		return fmt.Errorf(
			"compensation for %q: input type mismatch",
			n.name,
		)
	}

	return runCompensations(ctx, n, typedInput)
}

func rollback(ctx context.Context, record *Record) error {
	if record == nil {
		return nil
	}

	var errs []error

	switch record.Node.nodeKind() {
	case Task:
	case SequenceGroup:
		for _, v := range slices.Backward(record.Children) {
			if err := rollback(ctx, v); err != nil {
				errs = append(errs, err)
			}
		}
	case ParallelGroup:
		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, child := range record.Children {
			wg.Go(func() {
				if err := rollback(ctx, child); err != nil {
					mu.Lock()
					errs = append(errs, err)
					mu.Unlock()
				}
			})
		}

		wg.Wait()
	}

	if record.Succeeded {
		if err := record.Node.compensateNode(ctx, record.Input); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func runCompensations[I, O any](
	ctx context.Context,
	node *Node[I, O],
	input I,
) error {
	actions := node.compensate.actions

	run := func(i int) error {
		_, err := actions[i](ctx, input)
		if err != nil {
			return ErrorCompensationActionFunc(node.name, err)
		}

		return nil
	}

	switch node.compensate.mode {
	case SequentialMode:
		var errs []error

		for i := range slices.Backward(actions) {
			if err := run(i); err != nil {
				errs = append(errs, err)
			}
		}

		return errors.Join(errs...)
	case ParallelMode:
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errs []error

		for i := range actions {
			wg.Go(func() {
				if err := run(i); err != nil {
					mu.Lock()
					errs = append(errs, err)
					mu.Unlock()
				}
			})
		}

		wg.Wait()
		return errors.Join(errs...)
	default:
		return ErrorNoneMode
	}
}
