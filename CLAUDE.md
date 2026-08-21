# fest-direction

`direction` — stable, verifiable direction records for Festival work units. The
trust layer over `.festival` bundles (`obey-shared/festivalbundle`). Design of
record: Obey-Agent-Economy campaign, `workflow/design/festival-direction-archive`
(WI-6baeec). Build festival: `direction-archive-DA0004`.

## Build

```bash
just build    # bin/direction
just test     # unit tests
just lint     # gofmt, vet, golangci-lint (forbidigo bans fmt.Errorf)
just golden   # direction-hash fixture reproduction (phase 1 acceptance)
```

## Structure

- `cmd/direction` — entry point; lifecycle and exit code only
- `internal/cli` — cobra wiring; flags, output, exit codes; no domain logic
- `internal/errs` — THE error framework: sentinels + `Wrap(op, err)`
- `internal/ui` — shared brand palette → Lip Gloss styles; `Printer`
- `internal/version` — ldflags-injected build metadata
- `internal/normalize` — versioned, canonical normalization (`V1`, `V2`, `Current`, `ForVersion`)
- `internal/direction` — `Hash`: direction hash = bundle.id of the normalized tree, via `festivalbundle.Pack`
- `internal/anchor` — git state, HEAD-tree anchoring, records, trailers, commit-msg shim
- `internal/attest` — in-toto Statement v1, direction-record predicate, `Attest`, `Verify`
- Verbs: `hash`, `anchor`, `hook {commit-msg,install,uninstall}`, `attest`, `verify`
- `testdata/fixtures` — byte-stable festival trees with known hashes

## Rules

- `context.Context` first on every I/O path; check `ctx.Err()` before long work
- No `fmt.Errorf` — `errs.Wrap(op, err)` with a sentinel; dynamic detail goes in `op`
- Domain sentinels live in the package that owns them (`normalize.ErrUnknownField`,
  `normalize.ErrDestNotEmpty`, …) and are always wrapped with `errs.Wrap`;
  `internal/errs` holds only cross-cutting sentinels
- Never reimplement SPEC §7.1 — call `festivalbundle.PayloadContentID`
- Normalization is allowlist + fail-closed on unknown `fest_*` fields; any rule
  change bumps `normalization_version`; normalize on-disk bytes, never
  `fest parse --infer` output
- Anchoring: commit body trailers only; never touch the `[FE-…]` subject prefix;
  refuse a dirty tree unless `--force`
- Upstream: no changes to `fest`, `camp`, or `obey-shared` from here; proposals
  go to WI-93268b (obedience-growth-rd)
- Claims: follow `docs/claims.md`; `just checks claims` enforces the avoided-terms list
- Dependencies: stdlib, `obey-shared`, cobra, lipgloss, `yaml.v3`, `x/term`.
  Anything else needs explicit approval
- Files under 500 lines, functions under 50; dependency injection, no globals;
  table-driven tests, error cases first, context-cancellation cases included
