# install: the item namespace, CRDs by scope, the operator release and the Nexus CR.
# upgrade: the release to a fix image. Both wait for the operator's ready log line.

install_help() {
  cat <<EOF
lane.sh install <work> master|fix <tag>     chart from <src> (master) or <tree> (fix)
lane.sh upgrade <work> <tag>                helm upgrade --reset-then-reuse-values to the fix image
  Needs state scope (code|crd) and nexus (shared).
  CRDs: install master adds only missing CRDs, whatever the scope. scope crd: install fix and upgrade apply
  all from <tree>, server-side; scope code: they add only missing CRDs.
  nexus shared: copy $NX_SECRET from $NX_NS, Nexus CR url http://$NX_SVC.$NX_NS:$NX_PORT.
  Appends every object to INVENTORY.txt.
EOF
}

# apply_crds <dir> <item:0|1>. The base install adds only missing CRDs whatever the scope: the installed set may be
# newer than <src>, and a forced apply of the base would prune fields other operators on the cluster use.
# A crd item's fix install and upgrade apply the whole set from <tree>.
apply_crds() {
  local dir=$1 item=${2:-0} f name applied=""
  ls "$dir"/*.yaml >/dev/null 2>&1 || stop worktree "no CRD files in $dir"
  if [ "$SCOPE" = crd ] && [ "$item" = 1 ]; then
    runlog kube apply --server-side --force-conflicts -f "$dir" || fail crds "CRD apply from $dir failed; log: $LOG"
    inv_append "crds: applied all from $dir (scope crd)"
    return 0
  fi
  for f in "$dir"/*.yaml; do
    name=$(grep -m1 -E '^  name: ' "$f" | awk '{print $2}')
    kube get crd "$name" >/dev/null 2>&1 && continue
    runlog kube apply -f "$f" || fail crds "CRD apply $f failed; log: $LOG"
    applied="$applied $name"
  done
  inv_append "crds:${applied:- present, none applied}"
}

copy_admin_secret() {
  if [ "$DRY" = 1 ]; then say "+ copy secret $NX_SECRET from $NX_NS to $NS"; return 0; fi
  kube -n "$NX_NS" get secret "$NX_SECRET" -o json \
    | jq 'del(.metadata.namespace, .metadata.uid, .metadata.resourceVersion, .metadata.creationTimestamp, .metadata.managedFields, .metadata.ownerReferences)' \
    | kube -n "$NS" apply -f - >>"$LOG" 2>&1 || fail secret "secret copy failed; log: $LOG"
  inv_append "secret: $NX_SECRET (copied from $NX_NS)"
}

wait_operator() {
  [ "$DRY" != 1 ] || { say "+ wait for deploy/$DEPLOY_NAME and log line '$READY_LOG'"; return 0; }
  kube -n "$NS" rollout status "deploy/$DEPLOY_NAME" --timeout=180s >>"$LOG" 2>&1 || fail operator "deploy/$DEPLOY_NAME not ready; log: $LOG"
  local i=0
  # grep -q closes the pipe early; under pipefail the SIGPIPE exit of kubectl would fail the check.
  until { kube -n "$NS" logs "deploy/$DEPLOY_NAME" --tail=-1 2>/dev/null || true; } | grep -q "$READY_LOG"; do
    i=$((i + 1)); [ $i -lt 60 ] || fail operator "no '$READY_LOG' in the operator log after 120 s"
    sleep 2
  done
}

apply_nexus_cr() {
  local url=$1
  if [ "$DRY" = 1 ]; then say "+ apply Nexus/$SLUG-nexus url=$url secret=$NX_SECRET"; return 0; fi
  kube -n "$NS" apply -f - >>"$LOG" 2>&1 <<EOF || fail crs "Nexus CR apply failed; log: $LOG"
apiVersion: $CRD_GROUP/v1alpha1
kind: Nexus
metadata:
  name: nexus
spec:
  url: $url
  secret: $NX_SECRET
EOF
  inv_append "cr: Nexus/nexus url=$url"
  kube -n "$NS" wait nexus/nexus --for=jsonpath='{.status.connected}'=true --timeout=120s >>"$LOG" 2>&1 || fail crs "Nexus/nexus not connected; log: $LOG"
}

cmd_install() {
  case "${1:-}" in -h|--help) install_help; return 0;; esac
  work_dir "${1:-}"; shift
  local which=${1:-} tag=${2:-} chart start=$SECONDS
  [ -n "$which" ] && [ -n "$tag" ] || usage "lane.sh install <work> master|fix <tag>"
  case "$which" in master) chart=$SRC/$CHART;; fix) chart=$TREE/$CHART;; *) usage "lane.sh install <work> master|fix <tag>";; esac
  [ "$SCOPE" = code ] || [ "$SCOPE" = crd ] || stop scope "state scope is $SCOPE; set code or crd first"
  [ "$NEXUS" = shared ] || stop nexus "state nexus is $NEXUS; set shared first"
  LOG=$WORK/logs/install-$(ts).log; : >"$LOG"

  kube get ns "$NS" >/dev/null 2>&1 || runlog kube create ns "$NS" || fail namespace "create ns $NS failed; log: $LOG"
  runlog kube label ns "$NS" "$LABEL=$SLUG" --overwrite || fail namespace "label ns $NS failed; log: $LOG"
  inv_append "namespace: $NS ($LABEL=$SLUG)"
  local item=0; [ "$which" != fix ] || item=1
  apply_crds "$chart/crds" "$item"
  local nx_url="http://$NX_SVC.$NX_NS:$NX_PORT"
  copy_admin_secret
  runlog helm install "$RELEASE" "$chart" --kube-context "$CONTEXT" -n "$NS" "${INSTALL_FLAGS[@]}" \
    --set image.repository="$IMAGE_REPO" --set image.tag="$tag" --wait --timeout 5m \
    || fail install "helm install failed; log: $LOG"
  inv_append "release: $RELEASE chart=$chart image=$IMAGE_REPO:$tag"
  wait_operator
  apply_nexus_cr "$nx_url"
  [ "$DRY" = 1 ] || state_set installed "$NS, $RELEASE, $IMAGE_REPO:$tag"
  ledger_row "$(current_step)" "script:install-$which" - $(( (SECONDS - start) * 1000 )) - -
  say "RESULT=OK"
  say "namespace: $NS"
  say "release: $RELEASE"
  say "nexus: $nx_url ($NEXUS)"
  say "log: $LOG"
}

cmd_upgrade() {
  case "${1:-}" in -h|--help) install_help; return 0;; esac
  work_dir "${1:-}"; shift
  local tag=${1:-} chart=$TREE/$CHART start=$SECONDS
  [ -n "$tag" ] || usage "lane.sh upgrade <work> <tag>"
  LOG=$WORK/logs/upgrade-$(ts).log; : >"$LOG"
  [ "$SCOPE" != crd ] || apply_crds "$chart/crds" 1
  runlog helm upgrade "$RELEASE" "$chart" --kube-context "$CONTEXT" -n "$NS" --reset-then-reuse-values --set image.tag="$tag" --wait --timeout 5m \
    || fail upgrade "helm upgrade failed; log: $LOG"
  inv_append "upgrade: image=$IMAGE_REPO:$tag chart=$chart"
  wait_operator
  [ "$DRY" = 1 ] || state_set installed "$NS, $RELEASE, $IMAGE_REPO:$tag"
  ledger_row "$(current_step)" "script:upgrade" - $(( (SECONDS - start) * 1000 )) - -
  say "RESULT=OK"
  say "release: $RELEASE image=$IMAGE_REPO:$tag"
  say "log: $LOG"
}
