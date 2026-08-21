# Anchoring

How a direction hash becomes evidence: recorded against a git HEAD before work
begins, and carried on every commit. Implemented in `internal/anchor`.

## Enforced, or discipline-dependent?

| Moment | Mechanism | Enforced? | Hash |
|---|---|---|---|
| A task starts (festival and ritual kinds) | `pre_task_start` hook → `direction anchor .` | **Yes** — by `fest.yaml` configuration with `fail: closed`: a failed anchor blocks the task start | direction, of HEAD's tree |
| Every commit | commit-msg hook → `Festival-Direction` / `Festival-Normalization` trailers | **Yes**, per clone, once `direction hook install` has run (sequence `02_commit_trailers`) | direction, of the staged tree |
| Work begins on a non-festival kind (explore, design, intent, note) | manual `direction anchor <path>`, or the first trailer-bearing commit | No — discipline | direction |
| Completion | archive publish (deferred, not in this festival) | — | snapshot |

Say which row applies when you describe an anchor. A manually invoked anchor is
not an enforced one.

## What `anchor` does

1. Resolves the repository and the work unit's repo-relative path.
2. Hashes **HEAD's version** of the work unit (`git archive HEAD -- <path>`),
   because the record must point at a tree anyone can retrieve later.
3. If the working tree differs from HEAD: an equal direction hash means the
   changes are execution state only (status flips, progress events, fest's
   own frontmatter rewrite on completion) and HEAD's result is anchored; a
   different direction hash is refused, naming the paths. `--force` anchors
   the working tree instead and marks the event `forced`.
4. Appends `{head, source, snapshot_id, anchored_at, tool}` to
   `.direction/anchors/sha256-<direction hash>.json` at the repository root.
   The first event is the pre-execution anchor. Records are evidence about the
   plan, not the plan: they are excluded from normalization and **must be
   committed**.

## Binding in a festival

`fest.yaml`:

```yaml
hooks:
  enabled: true
  definitions:
    direction_anchor:
      command: direction anchor .   # cwd is the festival root; split on whitespace, no shell
      fail: closed
      timeout: 60s                  # a Go duration string — a bare number is rejected
      enabled: true
```

Each task document:

```yaml
hooks:
  start:
    pre: [direction_anchor]
```

The `start:` stage is honored on task documents only. Tasks created later
need the binding too — put it in the festival's task template or add it on
creation. `task_start` fires on the first transition into work, including a
direct completion; `fest task reset` re-arms it.

Prerequisite: `just install` (the binary must be on PATH as `direction`).
With `fail: closed`, a missing binary, an uncommitted plan change, or a
timeout blocks the task start and `fest next` shows the hook error. That is
the intended loud failure.

## Verification — 2026-08-21, festival `direction-archive-DA0004`

`fest hooks list`:

```text
direction_anchor   [festival]   enabled   fail=closed  timeout=1m0s
  command: direction anchor .
```

Completing the first three tasks of `002_IMPLEMENT_ANCHORING/01_anchor_command`
fired the hook at `pre` / `task_start` (festival ledger; the three `fail`
entries are gate tasks whose own `results/` files were still uncommitted — the
case that produced normalization v2, which excludes `results/`):

- `01_git_state.md` → pass (149 ms)
- `02_anchor_record.md` → pass (179 ms)
- `03_anchor_verb.md` → pass (152 ms)
- `04_task_start_binding.md` → pass (154 ms)
- `05_testing.md` → fail (159 ms)
- `06_review.md` → fail (164 ms)
- `07_iterate.md` → fail (162 ms)
- `08_fest_commit.md` → pass (115 ms)

-  → pass in 149 ms
-  → pass in 179 ms
-  → pass in 152 ms

Record `.direction/anchors/sha256-cb5b635bd0243edd8ccbd9bb7cd7fb3d47a1a2d66056fbb5c75437ef9f42b7f2.json` — direction hash
`sha256:cb5b635bd0243edd8ccbd9bb7cd7fb3d47a1a2d66056fbb5c75437ef9f42b7f2`, every event at HEAD `e16a2cae2ada9942df84b7b65187e37de110a78b`, none forced.
`direction hash festivals/active/direction-archive-DA0004` on the working tree
reports the same hash while statuses keep flipping.

Bring-up finding: the very first hook run exposed that fest re-serializes a
task's whole frontmatter on completion (flow sequences become block
sequences, blank lines appear, the trailing newline changes). v1 normalization
was finalized with canonical frontmatter and separators before any record was
committed; the single pre-finalization record was discarded rather than carried.

## What this proves

Anchoring makes "this plan predates this work" demonstrable from history alone:
the record and the commit trailers are checkable by anyone holding the
repository. It does not prove the plan was followed, that it was complete, or
that unwritten instructions did not exist. Phase 3's `docs/claims.md` carries
the full statement; until then see the design's `threat-model.md`.
