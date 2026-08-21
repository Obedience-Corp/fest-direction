package normalize

import (
	"errors"
	"regexp"
)

// ErrInvalidFestYAML marks a fest.yaml whose root is not a mapping.
var ErrInvalidFestYAML = errors.New("invalid fest.yaml")

// checkboxRE matches a checked Markdown task box at the start of a list item.
var checkboxRE = regexp.MustCompile(`(?m)^(\s*[-*] )\[[xX]\]`)

// resetCheckboxes turns "- [x]" / "- [X]" into "- [ ]" and reports how many
// boxes changed. Text after the box is untouched.
func resetCheckboxes(b []byte) ([]byte, int) {
	n := len(checkboxRE.FindAllIndex(b, -1))
	if n == 0 {
		return b, 0
	}
	return checkboxRE.ReplaceAll(b, []byte("${1}[ ]")), n
}

// stripStatusHistory removes metadata.status_history from a fest.yaml and
// re-encodes it deterministically. removed reports whether the key existed.
func stripStatusHistory(b []byte) ([]byte, bool, error) {
	root, mapping, err := decodeMapping(b, ErrInvalidFestYAML, "fest.yaml")
	if err != nil {
		return nil, false, err
	}
	removed := false
	if meta := mappingValue(mapping, "metadata"); meta != nil && meta.Kind == yamlMapping {
		removed = removeKey(meta, "status_history")
	}
	out, err := encodeNode(root, "fest.yaml")
	if err != nil {
		return nil, false, err
	}
	return out, removed, nil
}
