package normalize

import (
	"sort"

	"gopkg.in/yaml.v3"
)

// canonicalize rewrites a YAML node tree into one canonical shape so that
// re-serialization by other tools — fest rewrites a task's frontmatter on
// every status change, turning flow sequences into block sequences and
// dropping or adding blank lines — never moves the direction hash. Rules:
// mapping keys sorted, block style everywhere, no comments. Values and
// nesting are untouched.
func canonicalize(n *yaml.Node) {
	if n == nil {
		return
	}
	n.Style = 0
	n.HeadComment, n.LineComment, n.FootComment = "", "", ""
	if n.Kind == yaml.MappingNode {
		sortMapping(n)
	}
	for _, c := range n.Content {
		canonicalize(c)
	}
}

type pair struct{ key, value *yaml.Node }

func sortMapping(m *yaml.Node) {
	pairs := make([]pair, 0, len(m.Content)/2)
	for i := 0; i+1 < len(m.Content); i += 2 {
		pairs = append(pairs, pair{m.Content[i], m.Content[i+1]})
	}
	sort.SliceStable(pairs, func(i, j int) bool { return pairs[i].key.Value < pairs[j].key.Value })
	m.Content = m.Content[:0]
	for _, p := range pairs {
		m.Content = append(m.Content, p.key, p.value)
	}
}
