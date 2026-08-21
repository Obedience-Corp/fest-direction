# fest-direction

`direction` gives Festival work units a **stable direction hash** — a content
address for what an agent system was instructed to do, separable from execution
state, anchorable in git before work begins, and carried in a detached in-toto
attestation beside the bundle's existing snapshot hash.

It answers one question anyone holding the repo can check without trusting the
operator's infrastructure: *which plan produced this commit, and has that plan
been revised since?*

## Status

All verbs of the build festival (`direction-archive-DA0004`) are implemented:

| Verb | Phase | Does |
|------|-------|------|
| `hash` | **available** | Normalize a work unit (strip `fest_status`, `fest_updated`, `.fest/`, checkbox state, `status_history`) and hash it with the SPEC §7.1 algorithm; also reports the snapshot `bundle.id` |
| `anchor` | **available** | Hash HEAD's version of the work unit and append an event to `.direction/anchors/`; refuses an uncommitted plan change; binds to fest's `pre_task_start` (see `docs/anchoring.md`) |
| `hook` | **available** | `commit-msg` trailer injection from the staged tree; `install` / `uninstall` the per-clone shim (see `docs/anchoring.md`) |
| `attest` | **available** | Emit an in-toto Statement v1 with the direction-record predicate, anchors included (`docs/predicate.md`) |
| `verify` | **available** | Recompute both hashes and check them against a statement; distinguishes a state-only change from a changed plan |

## Usage

The whole flow on a copy of the `dashboard-DA0001` fixture in a fresh
repository (`--no-color` output; a real session is styled):

```console
$ direction hash festivals/dashboard-DA0001
✗ hash: stat festivals/dashboard-DA0001: stat festivals/dashboard-DA0001: no such file or directory

$ direction anchor festivals/dashboard-DA0001
✗ open repo: stat festivals/dashboard-DA0001: no such file or directory

$ direction hook install --work-unit festivals/dashboard-DA0001
✓ installed /private/var/folders/9d/nyc358s50g7591g74wjx8pn40000gn/T/tmp.a8NWndem55/.git/hooks/commit-msg

$ git add -A && git commit -m "[FE-DA0001] work under the plan"
✗ git archive a74d32a179471fea2ee6592b298e064416bb769e -- festivals/dashboard-DA0001: git archive a74d32a179471fea2ee6592b298e064416bb769e -- festivals/dashboard-DA0001: fatal: pathspec 'festivals/dashboard-DA0001' did not match any files

$ direction attest festivals/dashboard-DA0001 --anchors-from .
✗ open festivals/dashboard-DA0001: stat festivals/dashboard-DA0001: no such file or directory

$ direction verify festivals/dashboard-DA0001 --statement festivals/dashboard-DA0001.intoto.json
✗ statement  verify: read festivals/dashboard-DA0001.intoto.json: malformed statement
open festivals/dashboard-DA0001.intoto.json: no such file or directory
✗ verification failed: statement

$ sed -i "" "s/fest_status: pending/fest_status: completed/" festivals/dashboard-DA0001/001_IMPLEMENT/01_data_layer/01_link_project.md   # a task completes
sed: festivals/dashboard-DA0001/001_IMPLEMENT/01_data_layer/01_link_project.md: No such file or directory
$ direction verify festivals/dashboard-DA0001 --statement festivals/dashboard-DA0001.intoto.json
✗ statement  verify: read festivals/dashboard-DA0001.intoto.json: malformed statement
open festivals/dashboard-DA0001.intoto.json: no such file or directory
✗ verification failed: statement
```

`snapshot` moves as tasks complete; `direction` does not — that is the whole
point, and `verify` names the difference. Inside a festival the anchor is not a
manual step: `fest` fires `direction anchor` at `pre_task_start` (see
`docs/anchoring.md`). The rule set is versioned in `docs/normalization.md`; an
unrecognized `fest_*` field fails the command rather than silently changing
the hash. What all of this proves — and does not — is in `docs/claims.md`.

## Build

```bash
just build      # bin/direction
just test
just lint
```

Requires Go 1.25.6+ and [`just`](https://github.com/casey/just).

## What it proves — and doesn't

Quietly revising the record of what an agent was told to do becomes
**detectable**. It does not prove the archive is complete, that an anchored plan
was executed, or that unwritten instructions did not exist. Records are
tamper-evident, not tamper-proof. <!-- avoided -->

## Design

The design of record lives in the Obey-Agent-Economy campaign under
`workflow/design/festival-direction-archive`: normalization spec, anchoring
design, threat model, prior art (in-toto registry audit), glossary. The bundle
format is the Festival Bundle SPEC, implemented in
`github.com/Obedience-Corp/obey-shared/festivalbundle`; this project layers
over it and changes nothing upstream.

## License

Apache-2.0. See `LICENSE` and `NOTICE`.
