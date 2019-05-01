package api

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
)

func NodeNames(nodes []*StepNode) []string {
	var names []string
	for _, node := range nodes {
		name := node.Step.Name()
		if len(name) == 0 {
			name = fmt.Sprintf("<%T>", node.Step)
		}
		names = append(names, name)
	}
	return names
}

func LinkNames(links []StepLink) []string {
	var names []string
	for _, link := range links {
		name := fmt.Sprintf("<%#v>", link)
		names = append(names, name)
	}
	return names
}

func TopologicalSort(nodes []*StepNode) ([]*StepNode, error) {
	var sortedNodes []*StepNode
	var satisfied []StepLink
	seen := make(map[Step]struct{})
	for len(nodes) > 0 {
		var changed bool
		var waiting []*StepNode
		for _, node := range nodes {
			for _, child := range node.Children {
				if _, ok := seen[child.Step]; !ok {
					waiting = append(waiting, child)
				}
			}
			if _, ok := seen[node.Step]; ok {
				continue
			}
			if !HasAllLinks(node.Step.Requires(), satisfied) {
				waiting = append(waiting, node)
				continue
			}
			satisfied = append(satisfied, node.Step.Creates()...)
			sortedNodes = append(sortedNodes, node)
			seen[node.Step] = struct{}{}
			changed = true
		}
		if !changed && len(waiting) > 0 {
			for _, node := range waiting {
				var missing []StepLink
				for _, link := range node.Step.Requires() {
					if !HasAllLinks([]StepLink{link}, satisfied) {
						missing = append(missing, link)
					}
					log.Printf("step <%T> is missing dependencies: %s", node.Step, strings.Join(LinkNames(missing), ", "))
				}
			}
			return nil, errors.New("steps are missing dependencies")
		}
		nodes = waiting
	}
	return sortedNodes, nil
}

func PrintDigraph(w io.Writer, steps []Step) error {
	for _, step := range steps {
		for _, other := range steps {
			if step == other {
				continue
			}
			if HasAnyLinks(step.Requires(), other.Creates()) {
				if _, err := fmt.Fprintf(w, "%s %s\n", step.Name(), other.Name()); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
