/*
Copyright 2021 GramLabs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package filters

import (
	"strings"

	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// FilterOne is the opposite of kio.FilterAll, useful if you have a filter that
// is optimized for filtering batches of nodes but you just need to call `Pipe`
// on a single node.
func FilterOne(f kio.Filter) yaml.Filter {
	return yaml.FilterFunc(func(node *yaml.RNode) (*yaml.RNode, error) {
		nodes, err := f.Filter([]*yaml.RNode{node})
		if err != nil {
			return nil, err
		}

		if len(nodes) == 1 {
			return nodes[0], nil
		}

		return nil, nil
	})
}

// FilterAll is similar to `kio.FilterAll` except instead of evaluating for side
// effects, only the non-nil nodes returned by the filter are preserved.
func FilterAll(f yaml.Filter) kio.Filter {
	return kio.FilterFunc(func(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
		var result []*yaml.RNode
		for i := range nodes {
			n, err := f.Filter(nodes[i])
			if err != nil {
				return nil, err
			}
			if yaml.IsMissingOrNull(n) {
				continue
			}

			result = append(result, n)
		}
		return result, nil
	})
}

// Pipeline wraps a KYAML pipeline but doesn't allow writers: instead the
// resulting resource nodes are returned directly. This is useful for applying
// filters to readers in memory. A pipeline can also be used as a reader in
// larger pipelines.
type Pipeline struct {
	Inputs                []kio.Reader
	Filters               []kio.Filter
	ContinueOnEmptyResult bool
}

// Execute this pipeline, returning the resulting resource nodes directly.
func (p *Pipeline) Read() ([]*yaml.RNode, error) {
	var result []*yaml.RNode

	pp := kio.Pipeline{
		Inputs:                p.Inputs,
		Filters:               p.Filters,
		ContinueOnEmptyResult: p.ContinueOnEmptyResult,
		Outputs: []kio.Writer{kio.WriterFunc(func(nodes []*yaml.RNode) error {
			result = nodes
			return nil
		})},
	}

	if err := pp.Execute(); err != nil {
		return nil, err
	}

	return result, nil
}

// ReaderFunc is an adapter to allow the use of ordinary functions as a kio.Reader.
type ReaderFunc func() ([]*yaml.RNode, error)

// Read evaluates the typed function.
func (f ReaderFunc) Read() ([]*yaml.RNode, error) { return f() }

// RestoreVerticalWhiteSpace tries to put back blank lines eaten by the parser.
// It's not perfect (it only restores blank lines on the top level), but it helps
// prevent some changes to YAML sources that contain extra blank lines.
func RestoreVerticalWhiteSpace() kio.Filter {
	return kio.FilterAll(yaml.FilterFunc(func(node *yaml.RNode) (*yaml.RNode, error) {
		n := node.YNode()
		for i := range n.Content {
			// No need to insert VWS if we are still on the same line
			if i == 0 || n.Content[i].Line == n.Content[i-1].Line {
				continue
			}

			// Assume all lines before this node's head comment are blank and work back from there
			ll := n.Content[i].Line - 1
			if len(n.Content[i].HeadComment) > 0 {
				ll -= strings.Count(n.Content[i].HeadComment, "\n") + 1
			}

			// The previous node will have accounted for all the blanks above it
			ll -= lastLine(n.Content[i-1])

			// The foot comment will be stored two nodes back if this is a mapping node
			footComment := n.Content[i-1].FootComment
			if footComment == "" && n.Kind == yaml.MappingNode && i-2 >= 0 {
				footComment = n.Content[i-2].FootComment
			}
			if len(footComment) > 0 {
				ll -= strings.Count(footComment, "\n") + 2
			}

			// Check if all the lines are accounted for
			if ll <= 0 {
				continue
			}

			// Prefix the head comment with blank lines
			n.Content[i].HeadComment = strings.Repeat("\n", ll) + n.Content[i].HeadComment
		}

		return node, nil
	}))
}

// lastLine returns the largest line number from the supplied node.
func lastLine(n *yaml.Node) int {
	line := n.Line
	for i := range n.Content {
		if ll := lastLine(n.Content[i]); ll > line {
			line = ll
		}
	}
	return line
}
