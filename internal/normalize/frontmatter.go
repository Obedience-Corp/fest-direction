package normalize

import (
	"bytes"
	"errors"

	"gopkg.in/yaml.v3"

	"github.com/Obedience-Corp/fest-direction/internal/errs"
)

// ErrInvalidFrontmatter marks a frontmatter block that is not a YAML mapping.
var ErrInvalidFrontmatter = errors.New("invalid frontmatter")

var (
	fenceOpen  = []byte("---\n")
	fenceClose = []byte("\n---")
)

// splitFrontmatter returns the YAML between the fences and the body after the
// closing fence. The body keeps every byte after "---", including its newline,
// so reassembly is byte-exact. ok is false when doc has no frontmatter.
func splitFrontmatter(doc []byte) (fm, body []byte, ok bool) {
	if !bytes.HasPrefix(doc, fenceOpen) {
		return nil, nil, false
	}
	rest := doc[len(fenceOpen):]
	i := bytes.Index(rest, fenceClose)
	if i < 0 {
		return nil, nil, false
	}
	after := rest[i+len(fenceClose):]
	if len(after) > 0 && after[0] != '\n' {
		return nil, nil, false
	}
	return rest[:i+1], after, true
}

// filterFrontmatter removes stripped fest_* keys from doc's frontmatter and
// returns the rewritten document, the number of keys stripped, and an error
// for unknown fest_* keys or a malformed block. Documents without frontmatter
// are returned unchanged.
func filterFrontmatter(doc []byte, p Policy) ([]byte, int, error) {
	fm, body, ok := splitFrontmatter(doc)
	if !ok {
		return doc, 0, nil
	}
	root, mapping, err := decodeMapping(fm, ErrInvalidFrontmatter, "frontmatter")
	if err != nil {
		return nil, 0, err
	}
	stripped, err := filterMapping(mapping, p)
	if err != nil {
		return nil, 0, err
	}
	encoded, err := encodeNode(root, "frontmatter")
	if err != nil {
		return nil, 0, err
	}
	out := make([]byte, 0, len(fenceOpen)+len(encoded)+3+len(body))
	out = append(out, fenceOpen...)
	out = append(out, encoded...)
	out = append(out, "---"...)
	out = append(out, body...)
	return out, stripped, nil
}

func filterMapping(m *yaml.Node, p Policy) (int, error) {
	kept := make([]*yaml.Node, 0, len(m.Content))
	stripped := 0
	for i := 0; i+1 < len(m.Content); i += 2 {
		key, value := m.Content[i], m.Content[i+1]
		switch p.Classify(key.Value) {
		case Strip:
			stripped++
		case Unknown:
			return 0, errs.Wrap("frontmatter key "+key.Value, ErrUnknownField)
		default:
			kept = append(kept, key, value)
		}
	}
	m.Content = kept
	return stripped, nil
}
