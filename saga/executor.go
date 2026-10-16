package saga

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

func Run[I, O any](
	ctx context.Context,
	root *Node[I, O],
	input I,
) error {
	if root == nil {
		return ErrorRootEmpty
	}

	record, err := execute(ctx, root, input)
	if err == nil {
		return nil
	}

	rollbackCtx := context.WithoutCancel(ctx)
	rollbackErr := rollback(rollbackCtx, record, input)

	return errors.Join(err, rollbackErr)
}

func execute[I, O any](
	ctx context.Context,
	node *Node[I, O],
	input I,
) (*Record[I, O], error) {
	record := &Record[I, O]{Node: node}

	if err := ctx.Err(); err != nil {
		return record, err
	}

	switch node.kind {
	case Task:
		if node.execute == nil {
			return record, ErrorNodeNoExecuteFunc(node.name)
		}

		output, err := node.execute(ctx, input)
		if err != nil {
			return record, ErrorNodeExecutedFunc(node.name, err)
		}

		record.Succeeded = true
		record.Result = output
		return record, nil
	case SequenceGroup:
		for _, child := range node.children {
			childRecord, err := execute(ctx, child, input)
			record.Children = append(record.Children, childRecord)

			if err != nil {
				return record, fmt.Errorf("%w: %w", ErrorSequenceType, err)
			}
		}

		record.Succeeded = true
		return record, nil
	case ParallelGroup:
		results := make([]*Record[I, O], len(node.children))
		errs := make([]error, len(node.children))

		var wg sync.WaitGroup

		for i, child := range node.children {
			wg.Go(func() {
				results[i], errs[i] = execute(ctx, child, input)
			})
		}

		wg.Wait()
		record.Children = results

		var failures []error

		for i, err := range errs {
			if err != nil {
				failures = append(failures, fmt.Errorf("branch %d: %w", i, err))
			}
		}

		if len(failures) > 0 {
			return record, fmt.Errorf(
				"%w: %w",
				ErrorParallelType,
				errors.Join(failures...),
			)
		}

		record.Succeeded = true
		return record, nil
	default:
		return record, ErrorNoneKind
	}
}
