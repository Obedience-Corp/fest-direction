# Direction-record predicate

`fest direction attest` emits an [in-toto Statement v1](https://in-toto.io/Statement/v1)
whose predicate is a **direction record**. The statement is a detached JSON
file beside the work unit or bundle; the bundle itself is never modified, so a
`fest unbundle → fest pack` round trip leaves both the bundle and its hashes
untouched.

## Predicate type

```text
https://github.com/Obedience-Corp/fest-direction/predicate/direction-record/v1
```

**Decision (2026-08-21):** the type URI is namespaced under the repository that
owns the schema — a common in-toto practice that needs no new domain and cannot
collide with anyone else's predicate. Moving it under a brand domain is a `v2`
predicate once any statement has been published; before that it is a rename.
All 15 official in-toto predicates were audited in the design: none models a
work episode's direction, so a custom type is warranted (see the design's
`prior-art.md`).

## Subject

| Field | Value |
|---|---|
| `name` | `<subject.id>.festival` for festivals (e.g. `DA0001.festival`), else `<kind>.festival` |
| `digest.sha256` | the snapshot `bundle.id` as bare hex |

The digest is the Festival Bundle SPEC §7.1 payload hash — a real SHA-256 over
the canonical `path + size + bytes` record stream of `payload/` — **not** a
digest of the `.festival` zip file, whose bytes change with `packed_at`.
Generic tools that compute `sha256(file)` will therefore not match; the
predicate repeats `snapshot_id` in full so the semantics are explicit.

## Predicate schema

| Field | Type | Meaning |
|---|---|---|
| `direction_hash` | `sha256:<64 hex>` | `bundle.id` of the normalized tree — stable across execution |
| `normalization_version` | int | the rule set that produced it (`docs/normalization.md`); verification uses exactly this version |
| `snapshot_id` | `sha256:<64 hex>` | `bundle.id` of the raw tree — what it looked like at attestation |
| `kind` | string | `festival`, `ritual`, `explore`, `design`, `intent`, `note`, `workitem` |
| `subject` | object | `id`, `uuid`, `ref`, `type`, `title`, `created_at` as fest/camp record them (optional) |
| `source` | string | repo-relative work-unit path when `--anchors-from` names the repository; otherwise the absolute source |
| `anchors[]` | list | where the direction hash was anchored: `{type: record, ref: .direction/anchors/…}` and `{type: git-commit, ref: <sha>, at: <time>}` from record events and trailer-bearing commits reachable from HEAD (newest first, capped at 50, deduplicated) |
| `tool` | `{name, version}` | the producer |
| `created_at` | RFC 3339 UTC | when the statement was made |

## Verification

`fest direction verify <path> --statement <file>` runs, in order:

1. `statement` — well-formed Statement v1
2. `predicate-type` — the direction-record type above
3. `normalization-version` — a version this build knows (`normalize.ForVersion`); an unknown version stops here rather than guessing
4. `snapshot` — subject digest equals the recomputed snapshot id
5. `direction` — `direction_hash` equals the recomputed direction hash

Each failure has its own error. The combination *snapshot failed, direction
passed* is reported as **state-only**: execution state changed, the plan is
intact — the distinction the two-hash design exists to make.

## Signing

This version emits an unsigned statement. The statement bytes are exactly what
a DSSE envelope would carry as its payload (`payloadType:
application/vnd.in-toto+json`); signing wraps them and changes nothing inside,
which is the layering the Festival Bundle SPEC §11 anticipates.

## Example

Generated from `testdata/fixtures/dashboard-DA0001-baseline` with a fixed clock
(`internal/attest/testdata/statement_golden.json`):

```json
{
  "_type": "https://in-toto.io/Statement/v1",
  "subject": [
    {
      "name": "DA0001.festival",
      "digest": {
        "sha256": "4faf484b9bfb324cff4657bf0a79df2e6db2e14689b69cfbe4555a1981ba822e"
      }
    }
  ],
  "predicateType": "https://github.com/Obedience-Corp/fest-direction/predicate/direction-record/v1",
  "predicate": {
    "direction_hash": "sha256:7db1704665ae7e3125c3b045a4e59941b1be050fc1d9d1669d43e090247c3e6a",
    "normalization_version": 2,
    "snapshot_id": "sha256:4faf484b9bfb324cff4657bf0a79df2e6db2e14689b69cfbe4555a1981ba822e",
    "kind": "festival",
    "subject": {
      "id": "DA0001",
      "uuid": "b3e22f92-5cf1-47bb-b03b-898c8f5db33a",
      "type": "implementation",
      "title": "dashboard",
      "created_at": "2026-02-18T20:40:59.813189Z"
    },
    "source": "testdata/fixtures/dashboard-DA0001-baseline",
    "tool": {
      "name": "direction",
      "version": "test"
    },
    "created_at": "2026-08-21T12:00:00Z"
  }
}
```
