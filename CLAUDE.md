# edp-nexus-operator

`github.com/epam/edp-nexus-operator` — Go 1.25, kubebuilder operator that configures an **existing** Nexus Repository Manager instance declaratively via Kubernetes CRDs. It never installs Nexus; it only reconciles configuration into it.

CRD group: `edp.epam.com/v1alpha1`
CRDs: `Nexus`, `NexusRepository`, `NexusBlobStore`, `NexusCleanupPolicy`, `NexusRole`, `NexusUser`, `NexusScript`

## Build & Test

```bash
make build        # → dist/manager-<arch>
make test         # unit + controller tests via envtest (setup-envtest auto-downloaded)
make lint         # golangci-lint (config: .golangci.yaml)
make lint-fix     # golangci-lint with --fix
make fmt          # go fmt ./...
make vet          # go vet ./...
```

Integration tests against a live Nexus instance (optional):

```bash
TEST_NEXUS_URL=http://localhost:8081 TEST_NEXUS_USER=admin TEST_NEXUS_PASSWORD=admin make test
```

Run locally (needs a kubeconfig pointing at a cluster with CRDs installed):

```bash
go run ./cmd
```

## Code Generation

Run these after modifying types in `api/v1alpha1/` or kubebuilder markers:

```bash
make generate    # regenerates DeepCopy methods (zz_generated.deepcopy.go)
make manifests   # regenerates CRDs → deploy-templates/crds/ and config/crd/bases/, plus docs/api.md
make mocks       # regenerates mockery mocks in pkg/client/nexus/mocks/
```

CRDs are output to **`deploy-templates/crds/`** (Helm chart) and `config/crd/bases/` (Kustomize). Both are committed.

## Architecture

### Entry point

`cmd/main.go` — sets up the controller-runtime manager, registers all seven controllers and one admission webhook, wires `ApiClientProvider`.

### Layer overview

```
cmd/main.go
  └── registers all controllers with manager

api/v1alpha1/                   — CRD type definitions; one file per CRD
  repository_formats.go          — per-format structs (18 formats × hosted/proxy/group)
  repository_common.go           — shared embedded structs (storage, cleanup, etc.)

internal/controllers/<resource>/
  <resource>_controller.go       — Reconcile() entry; fetches CR, gets Nexus API client, calls chain
  chain/                         — sequential handler steps (create_*, remove_*)

pkg/client/nexus/
  provider.go                    — ApiClientProvider: looks up Nexus CR → reads Secret → builds client
  contracts.go                   — interfaces: User, Role, Repository, Script, FileBlobStore, S3BlobStore, NexusCleanupPolicyManager
  repository.go                  — hand-written resty HTTP client for /service/rest/v1/repositories
  cleanuppolicy.go               — hand-written resty client for /service/rest/internal/cleanup-policies
  repositorymapper.go            — GetRepoData(): maps NexusRepositorySpec → (format, type, name, data) for API calls
  mocks/                         — mockery-generated mocks (do not edit manually)

pkg/webhook/
  nexusrepository_webhook.go     — validation webhook for NexusRepository (rejects bad format/type combos)

deploy-templates/                — Helm chart with CRDs, values, templates
config/                          — Kustomize manifests (crd, rbac, manager, etc.)
```

### Nexus API client

**Not generated** — hand-written. Two HTTP clients built with `go-resty`:

- `RepoClient` (`repository.go`) — wraps Nexus REST v1 `/service/rest/v1/repositories/{format}/{type}/{id}`
- `NexusCleanupPolicyClient` (`cleanuppolicy.go`) — wraps the internal (unsupported) cleanup-policy API

For all other resources (users, roles, blob stores, scripts) the operator delegates to the `datadrivers/go-nexus-client` library (`nexus3.NexusClient`).

`ApiClientProvider.GetNexusApiClientFromNexus` reads the `Nexus` CR's `spec.secret` (a `Secret` with `user` and `password` keys) to build the client. All non-`Nexus` controllers call `GetNexusApiClientFromNexusRef` which first fetches the referenced `Nexus` CR by name.

### Reconciliation model

Each resource controller follows the same pattern:

1. Fetch the CR; if not found, return.
2. Call `ApiClientProvider` to build the appropriate Nexus API client.
3. On deletion: run `chain.NewRemove*`, remove finalizer.
4. On create/update: add finalizer, run `chain.NewCreate*`, set `status.value = "created"` or `status.error`.
5. Error requeue: 30 s (`controllers.ErrorRequeueTime`). Nexus connectivity recheck: 10 min.

The `Nexus` controller only verifies connectivity (calls `Security.User.List`) and sets `status.connected`. It does not manage any Nexus configuration itself.

### NexusRepository webhook

`ENABLE_WEBHOOKS=false` (env) disables the webhook at startup. The webhook validates that exactly one format and one type is set in the spec — enforced at both create and update.

### NexusRepository format/type mapping

`pkg/client/nexus/repositorymapper.go:GetRepoData` inspects the `NexusRepositorySpec` (18 format fields, each with hosted/proxy/group sub-structs) and returns a `RepoData{Type, Format, Name, Data}` used to build the REST call. When adding a new repository format, update: `api/v1alpha1/repository_formats.go`, `nexusrepository_types.go`, and `repositorymapper.go`.

## Conventions

- Tests use Ginkgo v2 + Gomega; mock interfaces live in `pkg/client/nexus/mocks/` and are regenerated with `make mocks` (mockery v3, config: `.mockery.yaml`).
- All mocks cover only the interfaces in `pkg/client/nexus/contracts.go`.
- `docs/api.md` and `deploy-templates/README.md` are auto-generated — do not edit manually (`make manifests` regenerates them).
- Helm chart README is generated by `make helm-docs`.
