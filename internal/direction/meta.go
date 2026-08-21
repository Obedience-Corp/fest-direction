package direction

import (
	"os"
	"path/filepath"

	"github.com/Obedience-Corp/obey-shared/festivalbundle"
	"gopkg.in/yaml.v3"
)

// meta is the identity Pack records in info.json — outside the hash, but
// useful to the resolver and the attestation.
type meta struct {
	kind    string
	name    string
	subject *festivalbundle.SubjectMeta
}

type festYAML struct {
	Metadata struct {
		ID           string `yaml:"id"`
		UUID         string `yaml:"uuid"`
		Name         string `yaml:"name"`
		FestivalType string `yaml:"festival_type"`
		CreatedAt    string `yaml:"created_at"`
	} `yaml:"metadata"`
}

type workitemYAML struct {
	ID    string `yaml:"id"`
	Type  string `yaml:"type"`
	Title string `yaml:"title"`
	Ref   string `yaml:"ref"`
}

var workitemKinds = map[string]struct{}{
	festivalbundle.KindExplore:  {},
	festivalbundle.KindDesign:   {},
	festivalbundle.KindIntent:   {},
	festivalbundle.KindNote:     {},
	festivalbundle.KindWorkitem: {},
}

// readMeta infers kind and subject from fest.yaml, then .workitem, then
// falls back to a bare workitem. Unreadable files are treated as absent:
// identity is metadata, and must never block hashing.
func readMeta(src string) meta {
	if m, ok := festivalMeta(src); ok {
		return m
	}
	if m, ok := workitemMeta(src); ok {
		return m
	}
	return meta{kind: festivalbundle.KindWorkitem, name: filepath.Base(src)}
}

func festivalMeta(src string) (meta, bool) {
	var f festYAML
	if !decodeYAML(filepath.Join(src, "fest.yaml"), &f) || f.Metadata.ID == "" {
		return meta{}, false
	}
	kind := festivalbundle.KindFestival
	if f.Metadata.FestivalType == festivalbundle.KindRitual {
		kind = festivalbundle.KindRitual
	}
	return meta{
		kind: kind,
		name: f.Metadata.Name,
		subject: &festivalbundle.SubjectMeta{
			ID:        f.Metadata.ID,
			UUID:      f.Metadata.UUID,
			Type:      f.Metadata.FestivalType,
			Title:     f.Metadata.Name,
			CreatedAt: f.Metadata.CreatedAt,
		},
	}, true
}

func workitemMeta(src string) (meta, bool) {
	var w workitemYAML
	if !decodeYAML(filepath.Join(src, ".workitem"), &w) {
		return meta{}, false
	}
	kind := festivalbundle.KindWorkitem
	if _, ok := workitemKinds[w.Type]; ok {
		kind = w.Type
	}
	return meta{
		kind:    kind,
		name:    w.Title,
		subject: &festivalbundle.SubjectMeta{ID: w.ID, Ref: w.Ref, Type: w.Type, Title: w.Title},
	}, true
}

func decodeYAML(path string, into any) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return yaml.Unmarshal(b, into) == nil
}
