// Package normalize produces the input to the direction hash: a copy of a
// work-unit tree with execution state removed. The rules are versioned (see
// Version); the field tables mirror normalization-spec.md in the design and
// docs/normalization.md.
package normalize

import (
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/Obedience-Corp/fest-direction/internal/errs"
)

// Version is the current normalization rule set. Any change to the tables, the
// non-frontmatter rules, or the exclude set is a new version — direction hashes
// are comparable only within one version, and every past version stays
// available through ForVersion so old records can still be verified.
const Version = 2

// ErrUnknownVersion is returned by ForVersion for a version this build lacks.
var ErrUnknownVersion = errors.New("unknown normalization version")

// ErrUnknownField is returned for a fest_* frontmatter key that is in neither
// table. Normalization is fail-closed: an unrecognized field means the schema
// grew, and guessing would silently move every direction hash.
var ErrUnknownField = errors.New("unknown fest_* field: normalization is fail-closed")

// Class is the verdict for one frontmatter key.
type Class int

const (
	// Retain keeps the key: it is direction, or user content.
	Retain Class = iota
	// Strip removes the key: it is execution state or an environment binding.
	Strip
	// Unknown is a fest_* key the policy has never seen; callers must fail.
	Unknown
)

// String renders the class name for messages and tests.
func (c Class) String() string {
	switch c {
	case Retain:
		return "retain"
	case Strip:
		return "strip"
	default:
		return "unknown"
	}
}

// Policy is one versioned rule set.
type Policy struct {
	Version int
	retain  map[string]struct{}
	strip   map[string]struct{}
	exclude map[string]struct{}
}

// V1 returns normalization_version 1 (2026-08-21): the original tables;
// excludes .fest, .bundles, .direction, .git, .env.
func V1() Policy {
	return Policy{
		Version: 1,
		exclude: set(".fest", ".bundles", ".direction", ".git", ".env"),
		retain: set(
			// identity
			"fest_type", "fest_id", "fest_ref", "fest_name", "fest_parent", "fest_order", "fest_created",
			// plan shape
			"fest_dependencies", "fest_parallel_group", "fest_workflow_position",
			// constraints
			"fest_gate_id", "fest_gate_type", "fest_autonomy", "fest_priority", "approval", "hooks",
			// typing
			"fest_festival_type", "fest_phase_type", "fest_sequence_type", "fest_task_type", "fest_work_type",
			// routing intent — instructions about how work should be done
			"fest_agent", "fest_complexity", "fest_estimated_tokens", "fest_requires_human", "fest_requires_context",
			// meta
			"fest_tags", "fest_version", "fest_tracking", "fest_managed",
		),
		// execution state and environment binding
		strip: set("fest_status", "fest_updated", "fest_working_dir"),
	}
}

// V2 returns normalization_version 2 (2026-08-21): V1 plus `results/`
// directories excluded — testing and review outputs written into a sequence
// are evidence of execution, not the plan.
func V2() Policy {
	p := V1()
	p.Version = 2
	p.exclude = set(".fest", ".bundles", ".direction", ".git", ".env", "results")
	return p
}

// Current returns the rule set this build hashes with.
func Current() Policy { return V2() }

// ForVersion returns the rule set for a recorded normalization_version.
func ForVersion(n int) (Policy, error) {
	switch n {
	case 1:
		return V1(), nil
	case 2:
		return V2(), nil
	default:
		return Policy{}, errs.Wrap("normalization version "+strconv.Itoa(n), ErrUnknownVersion)
	}
}

// Excluded reports whether a base name is never copied under p.
func (p Policy) Excluded(name string) bool {
	_, ok := p.exclude[name]
	return ok
}

// Classify reports how key is treated under p. Keys outside the fest_
// namespace are user content and are always retained.
func (p Policy) Classify(key string) Class {
	if _, ok := p.strip[key]; ok {
		return Strip
	}
	if _, ok := p.retain[key]; ok {
		return Retain
	}
	if strings.HasPrefix(key, "fest_") {
		return Unknown
	}
	return Retain
}

// Retained returns the sorted retain table.
func (p Policy) Retained() []string { return sortedKeys(p.retain) }

// Stripped returns the sorted strip table.
func (p Policy) Stripped() []string { return sortedKeys(p.strip) }

// Excludes returns the sorted exclude set.
func (p Policy) Excludes() []string { return sortedKeys(p.exclude) }

func set(keys ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		m[k] = struct{}{}
	}
	return m
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
