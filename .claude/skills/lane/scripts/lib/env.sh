
# Repo constants for the lane skill. The hook .claude/hooks/lane-allow.py parses this file:
# keep every value as `NAME=value` or `NAME=(a b c)` on one line.

CONTEXT=kind-krci
KIND_CLUSTER=krci
LABEL=lane/item
IMAGE_REPO=repo/edp/nexus-image
RELEASE_PREFIX=nexus-operator-
DEPLOY_NAME=nexus-operator
CHART=deploy-templates
CRD_DIR=deploy-templates/crds
CRD_GROUP=edp.epam.com
# A change under one of these paths makes the item `scope: crd`.
CRD_PATHS=(api/ deploy-templates/crds/ config/crd/ pkg/webhook/)
INSTALL_FLAGS=(--skip-crds)
READY_LOG='Starting workers'
TOOLS=(golangci-lint setup-envtest controller-gen mockery kustomize crdoc helmdocs)

# Shared Nexus on the testbed. NX_SECRET holds the keys `user` and `password`.
NX_NS=nexus
NX_SVC=nexus
NX_PORT=8081
NX_SECRET=nexus-admin
NX_DEPLOY=nexus
NX_CONTAINER=nexus-repository-manager
NX_MANIFEST=tests/e2e/helm-success-path/01-install-nexus-server.yaml

# Namespaces an item never deletes in. Deployments that must be ready before intake.
PROTECTED_NS=(nexus cert-manager krci argocd)
TESTBED_DEPLOYS=(nexus/nexus)

# Make targets that touch the cluster, a registry or the network: always a permission prompt.
PROMPT_TARGETS=(install bundle docker-push docker-build)

# Namespaced CR kinds, deleted in this order at teardown (users and roles before repositories, then the Nexus CR).
CR_DELETE_ORDER=(nexususers nexusroles nexusscripts nexusrepositories nexuscleanuppolicies nexusblobstores nexuses)

WORK_DIR=.work
LOCK_FILE=.work/.crd-lock

# Required plugins (installed_plugins.json keys), repo skills and commands.
DEPS_PLUGINS=(krci-general krci-godev krci-triage mattpocock-skills)
DEPS_SKILLS=()
DEPS_CMDS=(git gh kubectl helm docker kind go make jq python3 curl)
