# Normalization

Normalization produces the input to the **direction hash**: a copy of a
work-unit tree with execution state removed, so the hash is stable while work
progresses. Implemented in `internal/normalize`; the direction hash is the
`bundle.id` of the normalized tree packed with `obey-shared/festivalbundle`.

## Version history

| `normalization_version` | Date | Change |
|---|---|---|
| 1 | 2026-08-21 | Initial rule set, derived from the design's `normalization-spec.md` |

Any change to the tables or rules below is a new version. Direction hashes are
comparable only within one version, and the version travels with every hash
(`Result.NormalizationVersion`, the `Festival-Normalization` commit trailer,
the attestation predicate).

## Frontmatter keys (`*.md`)

Only `fest_*` keys are classified. Keys outside that namespace — user content,
template metadata such as `id`, `aliases`, `description` — are always retained.

### Strip — execution state and environment

| Key | Why |
|-----|-----|
| `fest_status` | Mutated by the `fest task` family as work executes |
| `fest_updated` | Written as `time.Now()` by the progress manager on every transition |
| `fest_working_dir` | Environment binding, not intent; changes when a repo moves; leaks layout |

### Retain — direction

| Group | Keys |
|-------|------|
| Identity | `fest_type` `fest_id` `fest_ref` `fest_name` `fest_parent` `fest_order` `fest_created` |
| Plan shape | `fest_dependencies` `fest_parallel_group` `fest_workflow_position` |
| Constraints | `fest_gate_id` `fest_gate_type` `fest_autonomy` `fest_priority` `approval` `hooks` |
| Typing | `fest_festival_type` `fest_phase_type` `fest_sequence_type` `fest_task_type` `fest_work_type` |
| Routing intent | `fest_agent` `fest_complexity` `fest_estimated_tokens` `fest_requires_human` `fest_requires_context` |
| Meta | `fest_tags` `fest_version` `fest_tracking` `fest_managed` |

Routing-intent keys are instructions about *how* work should be done, so they
are direction even though they look operational.

### Unknown — fail closed

A `fest_*` key in neither table returns `normalize.ErrUnknownField` naming the
file and key. Nothing is guessed. A loud break here forces a deliberate version
bump when the `fest` schema grows; a silent one would invalidate every prior
anchor without anyone noticing.

## Non-frontmatter rules

| Path | Rule |
|------|------|
| `.fest/` | Excluded entirely — `progress_events.jsonl`, `status_history.json` are pure execution trace |
| `.bundles/`, `.git/`, `.env` | Excluded — transfer records, VCS internals, secrets |
| `TODO.md` (any depth) | `- [x]` / `- [X]` → `- [ ]`; all text retained |
| `fest.yaml` (root) | `metadata.status_history` removed; everything else retained |
| Symlinks | Rejected (SPEC §4.3) |

Relative paths are preserved exactly: SPEC §7.1 hashes `path + size + bytes`,
so renaming or reordering is a direction change by construction — `fest
renumber` yields a new direction hash, and prior anchors are expected to
mismatch it.

## Determinism

Frontmatter and `fest.yaml` are parsed with the `yaml.v3` Node API and
re-encoded with a two-space indent. Key order and comments survive; quoting may
differ cosmetically from what `fest` wrote. That is fine because every hash
goes through the same normalizer — the only requirement is that equal input
yields equal bytes, which the test suite asserts by normalizing twice.

## What normalization is not

- It never runs over `fest parse --infer` output. Inference sets `Updated` from
  file mtime; normalization reads on-disk bytes.
- It never writes into the source tree. Output goes to an empty destination.
- It is not a merkle tree. The hash is SPEC §7.1's flat sequential SHA-256;
  per-file digests and selective disclosure are a deferred upstream proposal.
