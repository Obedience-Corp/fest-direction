# Anchoring

How a direction hash becomes evidence: recorded against a git HEAD before work
begins, and carried on every commit. Implemented in `internal/anchor`.

## Enforced, or discipline-dependent?

| Moment | Mechanism | Enforced? | Hash |
|---|---|---|---|
| A task starts (festival and ritual kinds) | `pre_task_start` hook → `fest-direction anchor .` | **Yes** — by `fest.yaml` configuration with `fail: closed`: a failed anchor blocks the task start | direction, of HEAD's tree |
| Every commit | commit-msg hook → `Festival-Direction` / `Festival-Normalization` trailers | **Yes**, per clone, once `fest direction hook install` has run (see *Commit trailers*) | direction, of the staged tree |
| Work begins on a non-festival kind (explore, design, intent, note) | manual `fest direction anchor <path>`, or the first trailer-bearing commit | No — discipline | direction |
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
      command: fest-direction anchor .   # cwd is the festival root; split on whitespace, no shell
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

Prerequisite: `just install` (the binary must be on PATH as `fest-direction`).
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

## Commit trailers

Every commit in the repository that holds the plan carries two trailers:

```text
Festival-Direction: sha256:…
Festival-Normalization: 2
```

A `commit-msg` hook writes them from the **staged** version of the work unit
(`git write-tree` → `git archive`), so each commit states exactly the plan it
was made under — including commits that do not touch the plan. That is the
point: the plan in force is recorded on the work done under it. Trailers join
an existing trailer block (`Signed-off-by` and friends are kept), sit before
git's comment tail, and are replaced rather than duplicated on amend.

### Install

```console
$ fest direction hook install --repo <repo> --work-unit <path to the work unit>
```

writes `.git/hooks/commit-msg` (honouring `core.hooksPath`) and
`.direction/config.yaml` with `default_work_unit`. Commit the config; the shim
is per clone. A foreign `commit-msg` hook is refused; `--force` keeps it as
`commit-msg.before-direction` and chains it ahead of the shim.
`fest direction hook uninstall` reverses both. The shim execs `fest-direction`, not `fest`, so a commit does not start the fest process. Inside the hook the work unit comes
from `--work-unit`, then `$DIRECTION_WORK_UNIT`, then the config; with none
configured the hook is a no-op, so unrelated repositories are never blocked.

The hook fails closed: a missing binary, an unknown `fest_*` field, or a git
failure aborts the commit with a message. `git commit --no-verify` bypasses
it — and leaves a commit without trailers, which is itself visible.

### `trailers` for `git commit-tree`

`fest-direction trailers --tree <git-tree-sha>` hashes that git tree, not the
index and not the working tree. Camp background jobs commit with
`git commit-tree`, which never runs the `commit-msg` hook, but they already
hold the tree they are about to commit. The job pipes its message to the
command and passes the successful stdout, unchanged, to `git commit-tree`.
That stdout is the whole message. Appending it to the original text
duplicates the subject and body, including when no work unit is configured
and stdin is copied through. The work unit resolves the same way as the
hook. A hash or git failure exits non-zero and writes nothing to stdout.

### Campaign wiring decision

`campaign.yaml` declares `hooks.commit_message: ob commit` — the message
*writer* that `fest commit --auto-write` runs. Two ways to put trailers on the
campaign-root commits `fest commit` creates for festival files:

- **A — a git `commit-msg` hook in the campaign root (chosen).** Enforced for
  every commit in that repository, whatever tool makes it; no change to `ob`,
  `fest`, or `camp`.
- **B — chain `direction` into `ob commit`.** Requires changing `ob`; out of
  scope. Recorded as an upstream proposal, together with a `--trailer` flag
  for `fest commit`.

Project repositories (here `projects/fest-direction`) do not carry trailers:
the work unit must live in the same repository as the commit. Their commits
are bound through the campaign-root commit that moves the submodule pointer —
that commit carries the trailer.

### Verification — campaign root, 2026-08-21

```console
$ direction hook install --repo . --work-unit festivals/active/direction-archive-DA0004
✓ installed .git/hooks/commit-msg
$ fest commit -m "anchor: commit trailers — …"      # creates the campaign-root commit fc3e7978f7dd
$ git log -1 --format=%B | git interpret-trailers --parse
Festival-Direction: sha256:0433491da2db54647743f2e8835544cb78d4a8f46e593ec74948d52a8dc84417
Festival-Normalization: 2
$ direction hash festivals/active/direction-archive-DA0004
direction    sha256:0433491da2db54647743f2e8835544cb78d4a8f46e593ec74948d52a8dc84417
```

The trailer on the campaign-root commit equals the direction hash of the
festival tree in that commit, and equals the `pre_task_start` record
`.direction/anchors/sha256-0433491da2db54647743f2e8835544cb78d4a8f46e593ec74948d52a8dc84417.json`.

## What this proves

Anchoring makes "this plan predates this work" demonstrable from history alone:
the record and the commit trailers are checkable by anyone holding the
repository. It does not prove the plan was followed, that it was complete, or
that unwritten instructions did not exist. Phase 3's `docs/claims.md` carries
the full statement; until then see the design's `threat-model.md`.
