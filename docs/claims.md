# Claims

What `fest direction verify` proves, and what it does not. Taken from the design's
threat model; where the shipped code proves less than the design hoped, that is
stated rather than smoothed over.

## Adversary

The primary adversary is the **operator of the agent system**, acting after the
fact to make past agent behaviour look better than it was — or to hide what it
was actually instructed to do. Defending against an outsider is easy; defending
against the party who controls the tooling, the repository, and the keys is the
realistic accountability problem.

## What the design prevents

| # | Attack | Prevented by |
|---|--------|--------------|
| 1 | Edit the plan after execution and claim it was always that way | Content addressing: the direction hash changes and no longer matches the anchored record, the commit trailers, or the statement |
| 2 | Substitute a different festival for the one a commit was made under | The commit carries the direction hash as a trailer; substitution produces a mismatch |
| 3 | Silently change *state* to fake progress in an attested record | The snapshot hash covers state; a changed snapshot is reported as such while the direction check still distinguishes plan from state |
| 4 | Claim a plan was written before the work when it was not | The `pre_task_start` anchor records the direction hash at a git HEAD before the task's first transition into work; the record and the HEAD are both in history |

Not built in this festival, and therefore **not claimed**: alteration or deletion
of an archived record (Arweave publishing is deferred), and third-party forgery
resistance (signing is deferred). An unsigned statement proves nothing about
who made it.

## What the design does not prevent

| # | Attack | Why it survives | Mitigation |
|---|--------|-----------------|------------|
| 1 | **Never anchoring at all** | Nothing compels it. An operator with something to hide simply does not run the tool | None technical. Anchoring must be a norm or a counterparty's requirement. In a festival it is enforced by configuration (`fail: closed` hook), which the operator can remove — visibly |
| 2 | **Selective anchoring** | Anchor only the flattering plans | Partially addressable by campaign-level ledgers; not built |
| 3 | **Anchoring a plan that was never executed** | Nothing binds "this plan was anchored" to "this plan actually ran" | The start-time anchor and per-commit trailers bind the plan to the *commits made under it*; they do not prove the commits implement the plan |
| 4 | **Backdating** | `created_at` and `anchored_at` are self-reported | A commit's position in history gives an upper bound; it does not prove content is not older |
| 5 | **Direction that is accurate but materially incomplete** | "Do what I said in Slack" is not in the tree | None. A record of written direction is not a record of all direction |
| 6 | **`git commit --no-verify`, history rewriting** | Hooks are per clone and bypassable; history can be rewritten | Both leave traces — missing trailers, divergent history — that a reader can see; neither is prevented |
| 7 | **Key compromise** | Not applicable yet: statements are unsigned | Deferred with signing |

## The boundary against "semantic intent verification"

The agent-identity literature names *semantic intent verification* — proving
an agent's behaviour reflects genuine rather than hijacked reasoning — as an
unsolved gap. **This tool does not close it and must not be described as if it
did.** A hijacked agent still runs under an honestly anchored plan; an agent
pursuing a misaligned objective while satisfying every written constraint
leaves a clean record. What is supplied is the *declared* half of
declared-versus-observed: a tamper-evident record of the instruction. Comparing
declared against observed remains a reader's judgement.

## The sentence that survives scrutiny

From the design, verbatim:

> This does not make an agent operator honest. It makes a *specific* dishonesty
> — quietly revising the record of what an agent was told to do — detectable by
> anyone, permanently, without trusting the operator's infrastructure.
>
> It does not prove the archive is complete, that archived plans were executed,
> or that unwritten instructions did not exist.

Delta for what is shipped: **"permanently" is not yet earned.** Records,
trailers, and statements live in git; permanence needs the deferred archival
layer. Today the honest form is "detectable by anyone holding the repository".

## Words this project does not use

<!-- avoided -->
| Avoid | Why | Use instead |
|-------|-----|-------------|
| "tamper-proof" | Records can be withheld, never made, or authored post hoc | "tamper-evident" | <!-- avoided -->
| "trustless" | Honest anchoring and, later, signing keys are trusted | "does not require trusting the operator's infrastructure" | <!-- avoided -->
| "provably correct agent behaviour" | Nothing here proves correctness — only what the instruction was | "verifiable direction" | <!-- avoided -->
| "immutable audit trail" | The trail is only as complete as what was anchored | "tamper-evident records; completeness not guaranteed" | <!-- avoided -->
| "prompt archive" | Direction is structured, not a string | "direction record" | <!-- avoided -->
| "reproducible AI" | Implies determinism | "direction reproducibility" (and that is deferred) | <!-- avoided -->

`just checks claims` fails the build if any of these appear in `README.md` or
`docs/` outside lines marked `<!-- avoided -->`.
