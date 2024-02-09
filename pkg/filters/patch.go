package filters

import (
	"bytes"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge2"
	yaml2 "sigs.k8s.io/yaml"
)

// UnsupportedPatchError is raised when a patch format is not recognized.
type UnsupportedPatchError struct {
	PatchType string
}

func (e *UnsupportedPatchError) Error() string {
	return fmt.Sprintf("unsupported patch type: %q", e.PatchType)
}

// PatchFilter is used to apply an arbitrary patch.
type PatchFilter struct {
	// The media type of the patch being applied.
	PatchType string
	// The actual raw patch.
	PatchData []byte
}

// Filter applies the configured patch.
func (f *PatchFilter) Filter(node *yaml.RNode) (*yaml.RNode, error) {
	switch f.PatchType {
	case "application/strategic-merge-patch+json", "strategic", "application/merge-patch+json", "merge", "":
		// The patch is likely JSON, parse it as YAML and just clear the style
		patchNode := yaml.NewRNode(&yaml.Node{})
		if err := yaml.NewDecoder(bytes.NewReader(f.PatchData)).Decode(patchNode.YNode()); err != nil {
			return nil, err
		}
		f.resetNodeStyle(patchNode.YNode())

		// Strategic Merge/Merge Patch is just the merge2 logic
		opts := yaml.MergeOptions{
			ListIncreaseDirection: yaml.MergeOptionsListPrepend,
		}
		return merge2.Merge(patchNode, node, opts)

	case "application/json-patch+json", "json":
		// The patch is likely JSON, but might be YAML that needs to be converted to JSON
		patchData := f.PatchData
		if !bytes.HasPrefix(patchData, []byte("[")) {
			jsonData, err := yaml2.YAMLToJSON(patchData)
			if err != nil {
				return nil, err
			}
			patchData = jsonData
		}
		jsonPatch, err := jsonpatch.DecodePatch(patchData)
		if err != nil {
			return nil, err
		}

		// This is going to butcher the YAML ordering/comments/etc.
		jsonData, err := node.MarshalJSON()
		if err != nil {
			return nil, err
		}
		jsonData, err = jsonPatch.Apply(jsonData)
		if err != nil {
			return nil, err
		}
		err = node.UnmarshalJSON(jsonData)
		if err != nil {
			return nil, err
		}
		return node, nil

	default:
		// This patch type is not supported
		return nil, &UnsupportedPatchError{PatchType: f.PatchType}
	}
}

// resetNodeStyle clears out the node style, this is useful to discard JSON formatting.
func (f *PatchFilter) resetNodeStyle(node *yaml.Node) {
	node.Style = 0
	for _, node := range node.Content {
		f.resetNodeStyle(node)
	}
}
