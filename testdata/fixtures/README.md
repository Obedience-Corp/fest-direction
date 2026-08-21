# Fixtures

Byte-stable festival trees with known hashes. Never edit in place — the hashes
below are what the tests assert. `fest pack` writes `.bundles/` into any
tree it packs; that directory is git-ignored here so manual packing cannot
drift the fixtures.

| Fixture | What it is | Snapshot `bundle.id` (SPEC §7.1, whole tree) |
|---------|------------|----------------------------------------------|
| `dashboard-DA0001-baseline/` | Old-format completed festival, copied from the campaign dungeon; the design's PoC subject | `sha256:4faf484b9bfb324cff4657bf0a79df2e6db2e14689b69cfbe4555a1981ba822e` |
| `dashboard-DA0001-mutated/` | Baseline plus one task `pending → completed` (`001_IMPLEMENT/01_data_layer/01_link_project.md`) and one appended `.fest/progress_events.jsonl` line | `sha256:5c7eced663c2efabbae61079f881aa9547c727eac6a29ca65de9ae2a80aaa9cd` |

Expected direction-hash behavior (phase 1):

- **v1 normalization:** baseline and mutated yield the **same** direction hash:
  `sha256:7db1704665ae7e3125c3b045a4e59941b1be050fc1d9d1669d43e090247c3e6a` (recorded 2026-08-21 after v1 was finalized with canonical frontmatter, asserted by `TestGoldenV1`).
- **PoC rules** (design's line-wise procedure: drop `.fest/`, delete
  `^fest_status:` / `^fest_updated:` lines from every `*.md`, reset
  `TODO.md` checkboxes, pack): both yield
  `sha256:50ddaea058b7b8ee46f89422da257d80ccb863ca640d887474544c752cdc854a`
  (reproduced with `fest pack` 2026-08-21; asserted by the PoC reproduction
  test). This is a pipeline sanity
  check, not the v1 spec; v1 also strips `fest.yaml` `status_history` and
  `fest_working_dir`.

| `workitem-note/` | Non-festival kind (`.workitem` type `note`), no execution state | direction hash == snapshot id by construction |

Notes: the DA0001 fixture has no checked `TODO.md` boxes and no
`fest_working_dir`, so those rules are unit-tested on synthetic input only; add
a real fixture when one exists.
