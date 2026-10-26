package saga

import (
	"errors"
	"fmt"
)

// Validate checks a workflow's structure without running it. It reports every
// problem it finds, joined into one error, so all issues can be fixed at once.
func Validate(root nodeRunner) error {
	if root == nil {
		return ErrorRootEmpty
	}

	return validateNode(root)
}

func validateNode(n nodeRunner) error {
	d := n.describe()

	var errs []error

	switch d.kind {
	case Task:
		if !d.executable {
			errs = append(errs, ErrorNodeNoExecuteFunc(d.name))
		}
	case SequenceGroup, ParallelGroup:
		if len(d.children) == 0 {
			errs = append(errs, fmt.Errorf("%w: %s", ErrorGroupEmpty, groupName(d)))
		}

		for _, child := range d.children {
			if child == nil {
				errs = append(errs, fmt.Errorf("%w: %s", ErrorNilChild, groupName(d)))
				continue
			}

			if err := validateNode(child); err != nil {
				errs = append(errs, err)
			}
		}
	default:
		errs = append(errs, ErrorNoneKind)
	}

	if d.compensations > 0 {
		if d.mode != SequentialMode && d.mode != ParallelMode {
			errs = append(errs, ErrorNoneMode)
		}

		if d.nilActions > 0 {
			errs = append(errs, fmt.Errorf("%w: %s", ErrorCompensationNilFunc, groupName(d)))
		}
	}

	return errors.Join(errs...)
}

func groupName(d nodeDescription) string {
	switch {
	case d.name != "":
		return d.name
	case d.kind == ParallelGroup:
		return "parallel"
	case d.kind == Task:
		return "task"
	default:
		return "sequence"
	}
}
