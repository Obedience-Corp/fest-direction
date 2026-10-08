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

## Fest plugin

`fest direction` is this program. `just install` puts two names on `PATH`:

| Name | Who calls it |
|------|----------------|
| `fest-direction` | Fest. Any `fest-*` executable is a plugin, so `fest direction hash <work-unit>` runs this binary. |
| `direction` | The git `commit-msg` shim and a festival's `pre_task_start` hook (`direction anchor .`). Those call the binary directly. They do not go through fest's plugin dispatcher. |

`plugins/manifest.yml` is optional metadata for `fest understand plugins`. Discovery works without it.

## Usage

The whole flow on a copy of the `dashboard-DA0001` fixture in a fresh
repository (`--no-color` output; a real session is styled):

```console
$ direction hash festivals/dashboard-DA0001
direction    sha256:7db1704665ae7e3125c3b045a4e59941b1be050fc1d9d1669d43e090247c3e6a
normalization v2
snapshot     sha256:4faf484b9bfb324cff4657bf0a79df2e6db2e14689b69cfbe4555a1981ba822e
kind         festival
subject      DA0001

$ direction anchor festivals/dashboard-DA0001
direction    sha256:7db1704665ae7e3125c3b045a4e59941b1be050fc1d9d1669d43e090247c3e6a
normalization v2
snapshot     sha256:4faf484b9bfb324cff4657bf0a79df2e6db2e14689b69cfbe4555a1981ba822e
head         57f44d97bfb72929bf143811873031f78f72c321
record       .direction/anchors/sha256-7db1704665ae7e3125c3b045a4e59941b1be050fc1d9d1669d43e090247c3e6a.json
events       1
⚠ commit .direction/anchors/sha256-7db1704665ae7e3125c3b045a4e59941b1be050fc1d9d1669d43e090247c3e6a.json with your next commit — the record is the evidence

$ direction hook install --work-unit festivals/dashboard-DA0001
✓ installed .git/hooks/commit-msg

$ git add -A && git commit -m "[FE-DA0001] work under the plan"
$ git log -1 --format=%B | git interpret-trailers --parse
Festival-Direction: sha256:7db1704665ae7e3125c3b045a4e59941b1be050fc1d9d1669d43e090247c3e6a
Festival-Normalization: 2

$ direction attest festivals/dashboard-DA0001 --anchors-from .
subject      DA0001.festival
digest       sha256:4faf484b9bfb324cff4657bf0a79df2e6db2e14689b69cfbe4555a1981ba822e
direction    sha256:7db1704665ae7e3125c3b045a4e59941b1be050fc1d9d1669d43e090247c3e6a
normalization v2
anchors      3
written      festivals/dashboard-DA0001.intoto.json

$ direction verify festivals/dashboard-DA0001 --statement festivals/dashboard-DA0001.intoto.json
✓ statement  https://in-toto.io/Statement/v1
✓ predicate-type  https://github.com/Obedience-Corp/fest-direction/predicate/direction-record/v1
✓ normalization-version  2
✓ snapshot  sha256:4faf484b9bfb324cff4657bf0a79df2e6db2e14689b69cfbe4555a1981ba822e
✓ direction  sha256:7db1704665ae7e3125c3b045a4e59941b1be050fc1d9d1669d43e090247c3e6a
✓ verified

$ sed -i "" "s/fest_status: pending/fest_status: completed/" festivals/dashboard-DA0001/001_IMPLEMENT/01_data_layer/01_link_project.md   # a task completes
$ direction verify festivals/dashboard-DA0001 --statement festivals/dashboard-DA0001.intoto.json
✓ statement  https://in-toto.io/Statement/v1
✓ predicate-type  https://github.com/Obedience-Corp/fest-direction/predicate/direction-record/v1
✓ normalization-version  2
✗ snapshot  want sha256:4faf484b9bfb324cff4657bf0a79df2e6db2e14689b69cfbe4555a1981ba822e got sha256:6e7037e1edfb9091b86a27d0af06ae507a84c84330556f0381a198aefc88006f
✓ direction  sha256:7db1704665ae7e3125c3b045a4e59941b1be050fc1d9d1669d43e090247c3e6a
⚠ execution state changed; direction intact
✗ verification failed: snapshot
```

`snapshot` moves as tasks complete; `direction` does not — that is the whole
point, and `verify` names the difference. Inside a festival the anchor is not a
manual step: `fest` fires `direction anchor` at `pre_task_start` (see
`docs/anchoring.md`). The rule set is versioned in `docs/normalization.md`; an
unrecognized `fest_*` field fails the command rather than silently changing
the hash. What all of this proves — and does not — is in `docs/claims.md`.

## Build

```bash
just build      # bin/direction and bin/fest-direction
just install    # both names on PATH; `fest direction` then works
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
