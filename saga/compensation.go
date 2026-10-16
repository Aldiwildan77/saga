package saga

import (
	"context"
	"errors"
	"slices"
	"sync"
)

func rollback[I, O any](
	ctx context.Context,
	record *Record[I, O],
	input I,
) error {
	if record == nil {
		return nil
	}

	var errs []error

	switch record.Node.kind {
	case Task:
	case SequenceGroup:
		for _, v := range slices.Backward(record.Children) {
			if err := rollback(ctx, v, input); err != nil {
				errs = append(errs, err)
			}
		}
	case ParallelGroup:
		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, child := range record.Children {
			wg.Go(func() {
				if err := rollback(ctx, child, input); err != nil {
					mu.Lock()
					errs = append(errs, err)
					mu.Unlock()
				}
			})
		}

		wg.Wait()
	}

	if record.Succeeded {
		if err := runCompensations(ctx, record.Node, input); err != nil {
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
