package saga

import (
	"fmt"
	"strings"
)

// Visualize returns an ASCII tree of the workflow structure without running it.
func Visualize(root nodeRunner) (string, error) {
	if root == nil {
		return "", ErrorRootEmpty
	}

	var b strings.Builder
	if err := writeTree(&b, root, "", ""); err != nil {
		return "", err
	}

	return b.String(), nil
}

func writeTree(b *strings.Builder, n nodeRunner, linePrefix, childPrefix string) error {
	label, err := nodeLabel(n)
	if err != nil {
		return err
	}

	b.WriteString(linePrefix + label + "\n")

	children := n.describe().children
	for i, child := range children {
		connector, indent := "├── ", "│   "
		if i == len(children)-1 {
			connector, indent = "└── ", "    "
		}

		if err := writeTree(b, child, childPrefix+connector, childPrefix+indent); err != nil {
			return err
		}
	}

	return nil
}

func nodeLabel(n nodeRunner) (string, error) {
	d := n.describe()

	var label string

	switch d.kind {
	case Task:
		if !d.executable {
			return "", ErrorNodeNoExecuteFunc(d.name)
		}
		label = d.name
	case SequenceGroup:
		label = "sequence"
		if d.name != "" {
			label = "nested " + d.name
		}
	case ParallelGroup:
		label = "parallel"
	default:
		return "", ErrorNoneKind
	}

	if d.compensations == 0 {
		return label, nil
	}

	switch d.mode {
	case SequentialMode:
		label += fmt.Sprintf(" [compensate: sequential x%d]", d.compensations)
	case ParallelMode:
		label += fmt.Sprintf(" [compensate: parallel x%d]", d.compensations)
	default:
		return "", ErrorNoneMode
	}

	return label, nil
}
