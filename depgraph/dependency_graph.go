package depgraph

import (
	"errors"
	"fmt"
)

type node struct {
	id         string
	next       []string
	marked     bool
	markedPerm bool
}

// DepGraph resolves dependency graph
type DepGraph struct {
	nodes map[string]node
}

// ErrCyclicGraph is an error indicating that the dependency graph is cyclic
// and therefore can't be resolved
var ErrCyclicGraph = errors.New("The dependency graph is cyclic")

// Add a dependency to the dependency graph
func (dg *DepGraph) Add(id string, dependsOn []string) {
	if dg.nodes == nil {
		dg.nodes = make(map[string]node)
	}

	if _, ok := dg.nodes[id]; !ok {
		dg.nodes[id] = node{
			next: make([]string, 0),
		}
	}

	n := dg.nodes[id]
	n.id = id
	dg.nodes[id] = n

	for _, depID := range dependsOn {
		if _, ok := dg.nodes[depID]; !ok {
			dg.nodes[depID] = node{
				next: make([]string, 0),
			}
		}

		n := dg.nodes[depID]
		n.next = append(n.next, id)
		dg.nodes[depID] = n
	}
}

// Resolve dependencies added via Add method
func (dg *DepGraph) Resolve() ([]string, error) {
	resolved := make([]string, 0, len(dg.nodes))

	for id, n := range dg.nodes {

		if n.id == "" {
			return resolved, fmt.Errorf("Bad input to dependency resolver. Node with id '%s' has not been registered", id)
		}

		err := dg.visit(n, &resolved)

		if err != nil {
			return resolved, err
		}
	}

	for i, j := 0, len(resolved)-1; i < j; i, j = i+1, j-1 {
		resolved[i], resolved[j] = resolved[j], resolved[i]
	}
	return resolved, nil
}

func (dg *DepGraph) visit(n node, resolved *[]string) error {
	if n.markedPerm {
		return nil
	}

	if n.marked {
		return ErrCyclicGraph
	}

	n.marked = true
	dg.nodes[n.id] = n

	for _, nextID := range n.next {
		err := dg.visit(dg.nodes[nextID], resolved)

		if err != nil {
			return err
		}
	}

	n.markedPerm = true
	dg.nodes[n.id] = n

	*resolved = append(*resolved, n.id)
	return nil
}
