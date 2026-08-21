# fest-direction

`direction` gives Festival work units a **stable direction hash** — a content
address for what an agent system was instructed to do, separable from execution
state, anchorable in git before work begins, and carried in a detached in-toto
attestation beside the bundle's existing snapshot hash.

It answers one question anyone holding the repo can check without trusting the
operator's infrastructure: *which plan produced this commit, and has that plan
been revised since?*

## Status

Phase 1 of the build festival (`direction-archive-DA0004`) is in progress. Verbs:

| Verb | Phase | Does |
|------|-------|------|
| `hash` | **available** | Normalize a work unit (strip `fest_status`, `fest_updated`, `.fest/`, checkbox state, `status_history`) and hash it with the SPEC §7.1 algorithm; also reports the snapshot `bundle.id` |
| `anchor` | 2 | Refuse a dirty tree, hash, write a local anchor record; hooks for commit trailers and `pre_task_start` |
| `attest` | 3 | Emit an in-toto Statement v1 with a versioned direction-record predicate |
| `verify` | 3 | Recompute both hashes from a bundle and check them against a statement |

## Usage

```console
$ direction hash festivals/.dungeon/completed/dashboard-DA0001
direction    sha256:4ee0bd159d49c964e342148f9da8618d1690bcf5bce38c48db87c6ce88041ba9
normalization v1
snapshot     sha256:4faf484b9bfb324cff4657bf0a79df2e6db2e14689b69cfbe4555a1981ba822e
kind         festival
subject      DA0001
```

Run it again after tasks complete and `snapshot` moves while `direction` does
not — that is the whole point. `--json` emits the same fields as JSON; `--out
<file>.festival` also writes the normalized bundle, whose `bundle.id` *is* the
direction hash. The rule set is versioned in `docs/normalization.md`; an
unrecognized `fest_*` field fails the command rather than silently changing
the hash.

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
tamper-evident, not tamper-proof.

## Design

The design of record lives in the Obey-Agent-Economy campaign under
`workflow/design/festival-direction-archive`: normalization spec, anchoring
design, threat model, prior art (in-toto registry audit), glossary. The bundle
format is the Festival Bundle SPEC, implemented in
`github.com/Obedience-Corp/obey-shared/festivalbundle`; this project layers
over it and changes nothing upstream.

## License

Apache-2.0. See `LICENSE` and `NOTICE`.
