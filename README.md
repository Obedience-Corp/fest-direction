# fest-direction

`direction` gives Festival work units a **stable direction hash** — a content
address for what an agent system was instructed to do, separable from execution
state, anchorable in git before work begins, and carried in a detached in-toto
attestation beside the bundle's existing snapshot hash.

It answers one question anyone holding the repo can check without trusting the
operator's infrastructure: *which plan produced this commit, and has that plan
been revised since?*

## Status

Scaffold. Verbs land by phase of the build festival (`direction-archive-DA0004`):

| Verb | Phase | Does |
|------|-------|------|
| `hash` | 1 | Normalize a work unit (strip `fest_status`, `fest_updated`, `.fest/`, checkbox state, `status_history`) and hash it with the SPEC §7.1 algorithm; also reports the snapshot `bundle.id` |
| `anchor` | 2 | Refuse a dirty tree, hash, write a local anchor record; hooks for commit trailers and `pre_task_start` |
| `attest` | 3 | Emit an in-toto Statement v1 with a versioned direction-record predicate |
| `verify` | 3 | Recompute both hashes from a bundle and check them against a statement |

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
