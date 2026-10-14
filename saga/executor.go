package saga

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

func Run[C any](
	ctx context.Context,
	root *Node[C],
	input C,
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

func execute[C any](
	ctx context.Context,
	node *Node[C],
	input C,
) (*Record[C], error) {
	record := &Record[C]{Node: node}

	if err := ctx.Err(); err != nil {
		return record, err
	}

	switch node.kind {
	case Task:
		if node.execute == nil {
			return record, ErrorNodeNoExecuteFunc(node.name)
		}

		if err := node.execute(ctx, input); err != nil {
			return record, ErrorNodeExecutedFunc(node.name, err)
		}

		record.Succeeded = true
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
		results := make([]*Record[C], len(node.children))
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
