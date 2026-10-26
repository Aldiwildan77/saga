package saga

import "fmt"

// DetectCycles reports ErrorCycle when a node appears inside its own subtree.
// A node used in two places that do not contain each other is not a cycle.
func DetectCycles(root nodeRunner) error {
	if root == nil {
		return ErrorRootEmpty
	}

	return detectCycles(root, make(map[nodeRunner]bool))
}

// path holds the nodes on the current route from the root. Seeing one again
// on that route means the node contains itself.
func detectCycles(n nodeRunner, path map[nodeRunner]bool) error {
	if path[n] {
		return fmt.Errorf("%w: %q", ErrorCycle, n.nodeName())
	}

	path[n] = true
	defer delete(path, n)

	for _, child := range n.describe().children {
		if child == nil {
			continue
		}

		if err := detectCycles(child, path); err != nil {
			return err
		}
	}

	return nil
}
