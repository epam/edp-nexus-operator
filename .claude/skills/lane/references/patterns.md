# Precedents and design rules

## Pattern index

The topics the Analyst surveys and the Design reviewer searches. Index only: open the file; the code wins.
A topic not listed here still needs a grep of the sibling controllers before any design choice.

| Topic | Look first |
|---|---|
| Source references (ConfigMap or Secret key) | `api/common/common.go` (`SourceRef`); `pkg/helper/helper.go` (`GetValueFromSourceRef`); use: `internal/controllers/blobstore/chain/crete_s3_blobstore.go` |
| Secret references as `$name:key` | `internal/controllers/user/chain/create_user.go` (`ParseSecretRef`, `getSecretFromRef`); Secret watch: `SetupWithManager` in `internal/controllers/user/nexususer_controller.go` |
| Nexus connection, credentials | `pkg/client/nexus/provider.go` (`GetNexusApiClientFromNexus`, `GetNexusApiClientFromNexusRef`); `internal/controllers/nexus/chain/check_connection.go` |
| Failure, retry, requeue | `internal/controllers/common.go` (`ErrorRequeueTime`, 30 s); `defaultRequeueTime`, `successRequeueTime` in `internal/controllers/nexus/nexus_controller.go`; no failure counter in this repo |
| Status | `api/common/common.go` (`StatusCreated`, `StatusError`); `update*Status` in `internal/controllers/user/nexususer_controller.go` and the sibling controllers; `status.connected` of `Nexus` |
| Idempotent writes, diff before write | `userChanged` in `internal/controllers/user/chain/create_user.go`; grep the sibling `chain/` packages |
| Finalizers, validation, events | `controllers.NexusOperatorFinalizer` in `internal/controllers/common.go`; `internal/controllers/<kind>/<kind>_controller.go`; no metrics endpoint in the chart |
| Optional fields, defaults | `+optional`, `+kubebuilder:default` markers in `api/common/common.go` (`NexusRef`) and `api/v1alpha1/*_types.go`; grep the sibling types |
| Immutability, enums, CRD validation | CEL markers `+kubebuilder:validation:XValidation:rule="self == oldSelf"` in `api/v1alpha1/nexususer_types.go`, `nexusrole_types.go`, `nexusblobstore_types.go`; `Enum` and `MinItems` in `nexususer_types.go` |
| Controller structure | `internal/controllers/<kind>/`: `<kind>_controller.go`, `chain/` (`create_*`, `remove_*`), `suite_test.go`, `*_integration_test.go`; `CLAUDE.md` "Reconciliation model" |
| Chain handlers | `internal/controllers/<kind>/chain/`; multi-step: `internal/controllers/script/chain/` (`chain.go`, `factory.go`); `internal/controllers/blobstore/chain/` |
| Nexus client | `pkg/client/nexus/` (`contracts.go`, `repository.go`, `cleanuppolicy.go`); the other kinds go through `go-nexus-client`; `mocks/` is generated, never edited |
| Repository format and type mapping | `pkg/client/nexus/repositorymapper.go` (`GetRepoData`); `api/v1alpha1/repository_formats.go`, `repository_common.go` |
| Admission webhook | `pkg/webhook/` (`nexusrepository_webhook.go`, `validation.go`, `cert.go`, `webhook.go`); `deploy-templates/templates/validation_webhook.yaml` |
| Helm, kustomize, CRD examples | `deploy-templates/`, `config/`, `deploy-templates/_crd_examples/` |
| Tests | `references/testing-proof.md` |

### Secret references

The pattern exists in this repo. A change that adds a Secret reference or a remote address to a spec cites
these rows; a part with status "not adopted" is a **new pattern** (step 4).

| Topic | Look first | Status |
|---|---|---|
| Secret reference reader | `pkg/secretref/sourceref.go`; `CLAUDE.md` "Sensitive values" | in this repo |
| Request body without the reference, error redaction | `pkg/client/nexus/repositorybody.go` (`WithAuthentication`); `internal/controllers/repository/chain/proxy_auth.go` | in this repo |
| Rotation: field index and Secret watch | `internal/controllers/repository/nexusrepository_controller.go` | in this repo |
| Deprecated field and admission warning | `CLAUDE.md` "Deprecating a field"; `pkg/webhook/nexusrepository_webhook.go` | in this repo |
| Scaffolded RBAC helper roles | `CLAUDE.md` "Scaffolded RBAC helper roles"; `config/rbac/`; `deploy-templates/templates/helper_roles.yaml`; `tests/rbac` | in this repo |
| Destination allowlist for remote addresses | none | not adopted: RBAC and `docs/security.md` are the controls |

## Precedent first

- Every design decision rests on a pattern the repo already uses. The Analyst surveys the siblings; the
  Architect cites `file:line` per decision; the Design reviewer opens each citation and hunts for missed
  precedents; the Developer codes only what the plan lists.
- Before adding any mechanism (status cleanup, new reasons, validation, chain reorder, a new way to handle
  secrets, retries, backoff, requeue, events or metrics), grep the repo for a precedent. None: it is a
  **new pattern**. It needs the Approver's explicit yes or no in step 4; the Architect lists it beside a
  precedent-based alternative, never as the default. Present the plan as the smallest change that follows
  existing patterns; list extras as options.
- Before "mirror controller X", survey every controller with a grep table of the pattern; a sibling can be
  broken itself.
- A plan cites a helper by what it does, not by its name: open its body. A helper named for one mechanism
  may use another.

## Bug or decision

- `git log --follow` the file to the introducing commit and compare with the siblings added in the same
  commit. Code whose intent and effect contradict is a bug. A feature runs the same way: the unresolved code
  path plus "omission or decision"; a deliberate removal is a decision.
- Restate the issue reporter's concern against each rule of `CLAUDE.md` before implementing; a rule can
  leave the reported case unfixed. Say so in the plan.
- Ask "who wrote the bad value": data the operator stored itself (legacy shapes) is repaired by the operator;
  values a user wrote follow the repo's resolution pattern (`GetValueFromSourceRef`, `$name:key`; set refs
  are read as written, values are not validated).

## Repo rules the plan must honour

- List every artifact a change touches: CRD, Helm template, kustomize overlays, docs, samples,
  `_crd_examples`, tests. Validate each before the first commit: `helm template <tree>/deploy-templates`,
  `<root>/bin/kustomize build <tree>/config/default`.
- `bundle/` is regenerated by the user at release. Leave it untouched (no `make bundle`, no
  `operator-sdk generate bundle`, no hand-synced bundle CRDs or CSV); list the bundle impact in the PR body.
- A sample that references a Secret breaks OperatorHub "Create instance": samples stay ref-free; ref
  examples go in `deploy-templates/_crd_examples/`.
- A doc sentence about webhook behaviour must also hold with `ENABLE_WEBHOOKS=false`.
- Test the final state the PR ships: CRDs, chart and image from `<tree>`. Upgrade paths (objects stored
  under the old schema, list or validation ratcheting, a GitOps re-apply across a CRD replacement) are
  documented from the API-server source and a scratch prototype in `<work>`, and go into the upgrade note;
  no repo test and no QA step installs the old CRD, unless the Approver asks.
- Re-review prompts list owner decisions as "do not re-report" and ask for critical defects only;
  otherwise rejected design points return each round.
