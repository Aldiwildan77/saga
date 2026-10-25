package saga

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

func Run[I any](
	ctx context.Context,
	root nodeRunner,
	input I,
) error {
	if root == nil {
		return ErrorRootEmpty
	}

	if ctx == nil {
		ctx = context.Background()
	}

	yieldCtx := NewYieldContext(ctx)
	record, err := root.run(yieldCtx, input)
	if err == nil {
		return nil
	}

	rollbackCtx := context.WithoutCancel(ctx)
	rollbackErr := rollback(rollbackCtx, record)

	return errors.Join(err, rollbackErr)
}

func (n *Node[I, O]) run(
	ctx context.Context,
	input any,
) (*Record, error) {
	record := &Record{
		Node:  n,
		Input: input,
	}

	if err := ctx.Err(); err != nil {
		return record, err
	}

	switch n.kind {
	case Task:
		if n.execute == nil {
			return record, ErrorNodeNoExecuteFunc(n.name)
		}

		typedInput, ok := input.(I)
		if !ok {
			return record, fmt.Errorf(
				"step %q: input type mismatch: got %T",
				n.name,
				input,
			)
		}

		output, err := n.execute(ctx, typedInput)
		if err != nil {
			return record, ErrorNodeExecutedFunc(n.name, err)
		}

		record.Result = output
		record.Succeeded = true
		return record, nil

	case SequenceGroup:
		current := input

		for _, child := range n.children {
			childRecord, err := child.run(ctx, current)
			record.Children = append(record.Children, childRecord)

			if err != nil {
				return record, fmt.Errorf(
					"%w: %w",
					ErrorSequenceType,
					err,
				)
			}

			current = childRecord.Result
		}

		record.Result = current
		record.Succeeded = true
		return record, nil

	case ParallelGroup:
		results := make([]*Record, len(n.children))
		errs := make([]error, len(n.children))

		var wg sync.WaitGroup

		for i, child := range n.children {
			wg.Add(1)

			go func(i int, child nodeRunner) {
				defer wg.Done()
				results[i], errs[i] = child.run(ctx, input)
			}(i, child)
		}

		wg.Wait()
		record.Children = results

		var failures []error

		for i, err := range errs {
			if err != nil {
				failures = append(
					failures,
					fmt.Errorf("branch %d: %w", i, err),
				)
			}
		}

		if len(failures) > 0 {
			return record, fmt.Errorf(
				"%w: %w",
				ErrorParallelType,
				errors.Join(failures...),
			)
		}

		outputs := make([]any, len(results))
		for i, childRecord := range results {
			outputs[i] = childRecord.Result
		}

		record.Result = outputs
		record.Succeeded = true
		return record, nil

	default:
		return record, ErrorNoneKind
	}
}
