# Analyst: root cause

Mode `defect` or `feature` (state `variant`). Output: `<work>/cause.md`; the Architect reads it. Read only,
apart from that file. The task is `task:` in `state.md`; the issue text is `<work>/issue.json` when present.
Read code, CRDs and the chart in `src`. Delegate plain lookups (greps, `git log --follow`, test layout) to
`Explore` agents on `haiku`. Read `references/patterns.md` and `references/testbed.md` "Nexus" first.

`cause.md`, in this order:

1. Scope lines. `artifacts: <repo-relative paths, space separated>`: every directory or file the change
   touches (API types, CRDs, chart, kustomize, docs, samples, `_crd_examples`, tests, webhook configuration),
   as paths such as `api/v1alpha1/ internal/controllers/user/ deploy-templates/crds/`; the Lead feeds
   them to `lane.sh state scope`. `nexus: shared`: the scenario runs on the shared Nexus.
2. Defect: the code path with `file:line`; the introducing commit (`git -C <src> show <commit>^:<file>`
   against the file in `src`); bug or decision, intent against effect, with reasoning.
   Feature: the unresolved code path; siblings that already do it; omission or decision; an evidence table
   from the code and the CRD schemas in `src`: where the behaviour is missing, what a user sees today
   (`kubectl explain` on `<src>/deploy-templates/crds/`, or the code path), the expected new values.
3. Sibling table: how each sibling controller handles the pattern in question, from a grep; outliers.
4. Pattern survey: for each topic of the pattern index the task touches: topic, how the repo does it,
   `file:line`, outliers; "no precedent" when the repo has none.
5. Nexus facts: every claim about Nexus behaviour (issue text, memory) as a row: claim, evidence (`file:line` in this repo or
   in the `go-nexus-client` module, a `lane.sh nx get` result, an operator log line), or "unverified".
6. Cluster scenario for QA: the CR and the trigger; metrics (status fields over time, log line counts,
   reconcile timestamps, operator log lines per object, `lane.sh nx get` output); expected wrong values (feature: expected new values);
   recovery actions; the spec edit that proves recovery.

Report (under 1200 words): the two scope lines; cause with `file:line`, introducing commit (defect), verdict
with reasoning, sibling table, the cluster scenario.
