# fest-direction

`fest direction` names the plan a Festival work unit was given, and keeps that name still while the work moves.

![A sealed plan on a desk, task cards shifting beside it.](docs/images/banner.jpg)

A Festival already has a snapshot hash, `bundle.id`. It changes when a task is checked off or a status field flips. The direction hash is the same content address after that execution state is removed. Completing a task moves the snapshot and leaves the direction hash alone. Editing the instructions moves the direction hash.

Anyone with the repository can ask: which plan was in force for this commit, and has that plan been revised since?

## Install

The program is the `fest-direction` binary. Fest finds that name on `PATH` and runs it as `fest direction`.

```bash
just install
```

There is no separate `direction` executable. `pre_task_start` and the git `commit-msg` shim call `fest-direction` directly, so a commit does not start `fest`.

## The two hashes

`fest direction hash <work-unit>` prints both.

| Hash | Moves when |
|------|------------|
| `direction` | The written plan changes: goals, tasks, dependencies, hooks |
| `snapshot` | Execution state changes: `fest_status`, checkboxes, `.fest/`, `status_history` |

Normalization is version 2. An unknown `fest_*` field fails the command rather than slipping into the hash. The rules are in [`docs/normalization.md`](docs/normalization.md).

```console
$ fest direction hash festivals/dashboard-DA0001
direction    sha256:7db1704665ae7e3125c3b045a4e59941b1be050fc1d9d1669d43e090247c3e6a
normalization v2
snapshot     sha256:4faf484b9bfb324cff4657bf0a79df2e6db2e14689b69cfbe4555a1981ba822e
kind         festival
subject      DA0001
```

## Where the hash is recorded

Three places, all local:

1. **Before the task starts.** `fest` runs `fest-direction anchor .` at `pre_task_start`. The hash of `HEAD` is appended to `.direction/anchors/`. An uncommitted plan change is refused. Commit the record; it is the evidence, and it is not part of the hash.
2. **On each commit.** `fest direction hook install` writes a `commit-msg` shim. The shim hashes the staged tree and adds `Festival-Direction` and `Festival-Normalization`.
3. **In a statement.** `fest direction attest` writes an in-toto Statement v1 beside the work unit. `fest direction verify` recomputes both hashes. A completed task fails the snapshot check and passes the direction check.

Camp background jobs commit with `git commit-tree`, which never runs the hook. They already hold the tree. Pipe the message through `fest direction trailers --tree <sha>` and pass the stdout, unchanged, to `commit-tree`. That stdout is the whole message.

The hook and anchor details are in [`docs/anchoring.md`](docs/anchoring.md). The statement is in [`docs/predicate.md`](docs/predicate.md).

## What a match means

A matching direction hash means the written plan is the one that was anchored and carried on the commit. It does not mean the plan was carried out, that the archive is complete, or that instructions given outside the tree did not exist. What is and is not claimed is in [`docs/claims.md`](docs/claims.md).

## Build

Go 1.25.6 or newer, and [`just`](https://github.com/casey/just).

```bash
just build    # bin/fest-direction
just test
just lint
```

## Design

The design notes live in the Obey-Agent-Economy campaign at `workflow/design/festival-direction-archive`. The bundle format is Festival Bundle, implemented in `github.com/Obedience-Corp/obey-shared/festivalbundle`. This repository layers the direction hash on that format and does not change it.

## License

Apache-2.0. See `LICENSE` and `NOTICE`.
