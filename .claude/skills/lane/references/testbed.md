# Testbed: the decisions a cluster run needs

The commands live in `scripts/lane.sh` (`build`, `install`, `upgrade`, `nx`, `tests`, `teardown`, `rebase`).
This file holds what the scripts cannot decide. Constants (context, namespaces, image repository, flags) are
in `scripts/lib/env.sh`; read it once instead of guessing a value. The cluster is the `try-kuberocketci`
testbed (its `CLAUDE.md` and Makefile stand it up; the kube context is pinned by krci-kubelock); cert-manager,
the operator CRDs and the shared Nexus stay installed between runs. "The user fixes the testbed" means that repo.

## Shared cluster

- The kind cluster hosts other components and other items at the same time. An item works only in its
  namespace `<ns>` (= `<slug>`, labelled by `lane.sh install`). Several items run from one checkout; each
  owns `<work>`, `<tree>`, `<branch>`, `<ns>`, `<release>`, its image tags and its `<slug>-*` Nexus objects,
  and touches nothing of another item. Items share `<root>` (read-only), `<root>/bin`, the cluster, its CRDs
  and the shared Nexus.
- The operator reconciles one namespace: `WATCH_NAMESPACE` is the item namespace `<ns>`, set by the chart
  (no cluster-wide flag exists). Never change it. The chart has one ValidatingWebhookConfiguration
  (NexusRepository only), `edp-nexus-operator-validating-webhook-configuration-<ns>`, selected to `<ns>`,
  `failurePolicy: Fail`. The operator patches its CA bundle itself; cert-manager is not used.
- Cluster-scoped objects an item may create or change, all named with `<ns>` or `<slug>`: the chart's webhook
  configuration, its ClusterRole and ClusterRoleBinding `edp-nexus-operator-<ns>`, the helper ClusterRoles
  `edp-nexus-operator-<ns>-{nexus,nexusrepository,nexusblobstore}-{admin,viewer}`, and the namespace itself. Nothing else. Never touch the protected namespaces listed in `env.sh`, apart from reading the
  Nexus admin secret.
- Deleting the namespace is the user's command: `! kubectl --context <ctx> delete ns <ns>`. The hook denies
  it to the skill so no item can remove another's.

## CRDs are shared by every item

- `scope: code` (no change under the CRD paths of `env.sh`): the scripts apply a CRD only when it is missing
  and never overwrite one; teardown leaves them.
- `scope: crd`: `state scope` registers the item in the crd-lock (`lane.sh lock show`), information only; no
  item waits on it. The master run still adds only missing CRDs (the installed set may be newer than `<src>`,
  and a forced base apply would prune fields other operators use); the fix run applies the CRDs from `<tree>`
  server-side, and teardown restores them from `<src>` only when no other live item namespace exists. While
  another crd item is live, the installed CRDs may be theirs: QA names them in the evidence.
- Helm never upgrades CRDs; the chart is installed with `--skip-crds` for that reason.

## Images and releases

- Images come only from `lane.sh build`: the native architecture through the Makefile; never QEMU or a raw `go build`. A detached `<src>` has no reachable tag: `git describe`
  prints `No names found`; the build still succeeds with an empty version and the full commit hash. One fresh
  tag per build: kind caches by tag.
- After `lane.sh upgrade`, compare `helm get manifest <release> -n <ns>` with `helm template` of the same
  chart and values before measuring; a difference means the release did not take the chart you think.

## Nexus

- One mode: `nexus: shared`. The shared Nexus is `sonatype/nexus3` 3.82.0 in namespace `nexus`, deploy and
  service `nexus`, port 8081, defined by the `NX_MANIFEST` of `env.sh`. Admin credentials: secret `NX_SECRET` of `env.sh`, keys `user` and
  `password`. `lane.sh install` copies the secret into `<ns>` and applies a `Nexus` CR `nexus` that points at
  `http://nexus.nexus:8081`. There is no dedicated Nexus. Never
  `kubectl delete -f` the manifest, never restart or delete the shared deployment. A scenario that needs
  another Nexus version or a Nexus restart goes to the Approver.
- The `NX_MANIFEST` holds the Deployment, Service, PVC and ConfigMap only: no Namespace and no `NX_SECRET`.
  A recreated shared Nexus needs the namespace, the secret (keys `user` and `password`) and, once, the EULA
  accepted by the user. Until then `/repository/*` returns 403: a scenario that makes Nexus call a proxy
  remote cannot run. The user fixes this; the skill does not recreate the shared Nexus.
- The shared Nexus is also the fixture of `lane.sh tests` for every item (`TEST_NEXUS_URL`,
  `TEST_NEXUS_USER`, `TEST_NEXUS_PASSWORD`; the script port-forwards and reads the secret). Tests share
  one Nexus: names are global there. Never start a local Nexus container for the tests.
- Every repository, role, user, blob store, cleanup policy and script an item creates is named
  `<slug>-<purpose>` (`<slug>-main`, `<slug>-viewer`). Never list, read or change another name.
  `lane.sh nx <work> names` lists the item's repositories, roles, users and blob stores;
  `lane.sh nx <work> get repositories|roles|users|blobstores [<name>]` is a read-only GET and enforces the
  prefix. All writes go through the operator's CRs. Cleanup policies and scripts have no `nx` kind: read
  their CR `.status`.
- Verify Nexus behaviour on the cluster, never from memory. Evidence: operator logs
  (`kubectl -n <ns> logs deploy/nexus-operator -c nexus-operator --since-time=<t>`), the CR `.status`
  (`value`, `error`; `Nexus`: `connected`) and `lane.sh nx get`.
- GET returns no user password: a password write shows only in the operator log and the CR `.status`. Count
  writes from the operator log lines of the chain steps; read the Nexus state back with `lane.sh nx get`
  and compare it with the spec. The lane has no Nexus audit stream.
- The operator calls the Nexus REST API under `/service/rest/`; the cleanup-policy kind uses the internal
  `/service/rest/internal/cleanup-policies` endpoint, which the public API documentation does not cover.

## Measurements

- **Fast feedback:** keep measurement windows at 2 to 3 minutes. A tunable timing may not drive the
  controller under test: check that controller's own requeue first. `ErrorRequeueTime` (30 s) and
  `successRequeueTime` (10 min, `Nexus` controller) are constants: a changed value needs a test-only build. Write
  `timing: <name> original=<value>` to `INVENTORY.txt`, restore afterwards and verify (`helm get values`,
  deployment env). Nudge reconciles with an annotation instead of waiting for a requeue.
- **Reconcile rate:** create the CR, wait for `status.value`, then over a fixed window `grep -c` of
  `Reconciling <Kind>` and of `Reconciler error` in the operator log; recreate the CR after switching images.
- **Webhook A/B:** patch `clientConfig.service.path` of the webhook to an unserved path; restore it and
  compare with a saved copy.
- **Metrics:** the chart starts the manager with the default `--metrics-bind-address=0`: no metrics endpoint,
  no scraper. Measure with logs and `.status`.
- Capture operator logs before any `rollout restart`; the pod is gone afterwards. After an upgrade wait for
  the ready log line (`lane.sh upgrade` does) before measuring.
- `kubectl apply` of a manifest whose last-applied holds `[]` restores `[]` over a value the operator rewrote.
- Port-forwards: a free local port picked at run time, never a fixed one; stop every background job you
  started (port-forwards, watchers, log tails) with `TaskStop` before you report. A job alive at hand-back
  blocks your resume. Keep `kubectl delete` in a separate call from installs.

## Inventory and teardown

- `<work>/evidence/master/INVENTORY.txt` is the teardown list. The scripts append what they install; QA
  appends every CR, secret, port and timing it creates or changes by hand, as it goes, never at the end.
- `lane.sh teardown <work>` removes everything in the documented order and prints the leftovers and the
  user's namespace command. A Nexus object that outlives its CR is printed as `nexus:<name>`; the lane has
  no write access: the user deletes it through the REST API. Timings are restored by hand from the `timing:` lines it prints.

## Permissions and sandbox

- The `krci-kubelock` mod of the `krci-triage` plugin pins `KUBECONFIG` to the context in `env.sh`: every
  `kubectl`, `helm`, `make` target and script sees only that context, and a command that names another
  context, `--kubeconfig`, a `KUBECONFIG=` override or a `~/.kube` path is denied before it runs. Its
  settings are the user's (`/config`); intake stops with `STOP:kubelock` while the pin is off. The scripts
  still pass `--context` and `--kube-context` explicitly, so a run without the lock cannot hit another cluster.
- The hook `.claude/hooks/lane-allow.py` allows the lane script, Make through `LOCALBIN`, `go test` and
  read-only git in the lane worktrees, and denies namespace deletes, deletes in protected namespaces, CRD
  deletes, a push from a lane tree and `gh pr create`; its docstring is the list. `.claude/settings.json`
  allows every `kubectl` on the pinned context without a prompt: the hook denies are the only brake, and
  isolation between items rests on the rules in this file. Everything else prompts by design
  (`git add|commit|rebase|fetch|worktree remove|branch -D`, `helm uninstall`, `docker rmi`, `gh`, the Make
  targets in `PROMPT_TARGETS`, `lane.sh teardown <work> --all --discard`). A command with a shell separator (`;`, `|`, `&&`) never matches an
  allow rule: one command per call.
- `make build`, `vet`, `lint-fix`, `generate`, `manifests` and `validate-docs` run inside the sandbox.
  `validate-docs` compares regenerated `docs/api.md` and `deploy-templates/README.md` with the index: run it
  after `git add` or after the commit, not with unstaged changes to those files.
- Envtest starts a local API server only and never touches the cluster. It needs the sandbox disabled because
  it removes its port files under `~/Library/Caches/kubebuilder-envtest`; the user can instead allow that
  path for writes in the sandbox settings.
- `$TMPDIR` differs between sandboxed and unsandboxed shells: write and read a file in the same mode.
- Process substitution `<(...)` and `/dev/fd` fail; an empty `diff` is then false evidence. Write temp files.
- `kill`, `ps`, `pgrep` fail with "operation not permitted"; stop background work with `TaskStop`.
- Shell writes into `.claude/` and into files outside the checkout (a tracker file in another repository)
  fail; use the Write and Edit tools there.
- Tools in `<root>/bin` are built with the host `go` and not rebuilt when it changes. After a Go upgrade,
  `make lint` fails with typecheck errors such as `could not import math/rand/v2`: check with
  `go version -m <root>/bin/<tool>`, delete `<root>/bin/<tool>` and `<root>/bin/<tool>-<version>`, and the
  Lead reruns `lane.sh intake <slug> <branch>` for the item (sandbox disabled); the worktrees share `<root>/bin`.
- `git worktree remove` refuses a dirty worktree: inspect `git -C <tree> status` and ask before `--force`.
- The sandbox protects `.vscode/settings.json` and `.vscode/launch.json` in every directory: a worktree
  removal inside the sandbox drops the git entry and leaves the directory.
  `lane.sh teardown` therefore runs with the sandbox disabled and removes such a leftover directory itself.
  A worktree directory removed by hand leaves a stale entry: `git -C <root> worktree prune`.
