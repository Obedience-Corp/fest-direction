package normalize

import (
	"bytes"
	"errors"

	"gopkg.in/yaml.v3"

	"github.com/Obedience-Corp/fest-direction/internal/errs"
)

// decodeMapping parses YAML whose root must be a mapping and returns the
// document node (for re-encoding) and the mapping node (for editing).
func decodeMapping(b []byte, sentinel error, what string) (*yaml.Node, *yaml.Node, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(b, &root); err != nil {
		return nil, nil, errs.Wrap(what+": parse", errors.Join(sentinel, err))
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) != 1 || root.Content[0].Kind != yaml.MappingNode {
		return nil, nil, errs.Wrap(what+": root is not a mapping", sentinel)
	}
	return &root, root.Content[0], nil
}

// encodeNode renders a document node deterministically (two-space indent).
func encodeNode(n *yaml.Node, what string) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(n); err != nil {
		return nil, errs.Wrap(what+": encode", err)
	}
	if err := enc.Close(); err != nil {
		return nil, errs.Wrap(what+": encode", err)
	}
	return buf.Bytes(), nil
}

// mappingValue returns the value node for key, or nil.
func mappingValue(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// removeKey drops key from the mapping and reports whether it was present.
func removeKey(m *yaml.Node, key string) bool {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content = append(m.Content[:i], m.Content[i+2:]...)
			return true
		}
	}
	return false
}

// yamlMapping is the mapping kind, re-exported so rule files stay free of a
// direct yaml import.
const yamlMapping = yaml.MappingNode
