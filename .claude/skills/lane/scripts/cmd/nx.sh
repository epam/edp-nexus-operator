# nx: read-only view of the shared Nexus. Names must start with <slug>-.

nx_help() {
  cat <<EOF
lane.sh nx <work> names                  repositories, roles, users and blob stores of this item
lane.sh nx <work> get <kind> [<name>]    GET one kind; kind: repositories | roles | users | blobstores
  <name> must start with <slug>-; a list is filtered to <slug>-*. Credentials come from secret $NX_SECRET
  in namespace $NX_NS. Output is the JSON body. All writes go through the operator's CRs.
EOF
}

cmd_nx() {
  case "${1:-}" in -h|--help) nx_help; return 0;; esac
  work_dir "${1:-}"; shift
  [ $# -gt 0 ] || usage "lane.sh nx <work> names | get <kind> [<name>]"
  if [ "$1" = names ]; then
    [ "$DRY" = 1 ] && { say "+ GET repositories, security/roles, security/users, blobstores (filtered to $SLUG-)"; return 0; }
    nx_item_names "$NX_NS" | grep . || say "no $SLUG- object"
    return 0
  fi
  [ "$1" = get ] && [ $# -ge 2 ] || usage "lane.sh nx <work> names | get <kind> [<name>]"
  local path name=${3:-}
  case "$2" in
    repositories) path=/service/rest/v1/repositories;;
    roles) path=/service/rest/v1/security/roles;;
    users) path=/service/rest/v1/security/users;;
    blobstores) path=/service/rest/v1/blobstores;;
    *) usage "kind must be repositories, roles, users or blobstores";;
  esac
  if [ -n "$name" ]; then
    case "$name" in "$SLUG-"*) ;; *) stop name "$name is not $SLUG-*";; esac
    case "$name" in */*|*..*|*\?*) stop name "$name is not a plain name";; esac
    [ "$2" = users ] && path="$path?userId=$name" || path="$path/$name"
  fi
  if [ "$DRY" = 1 ]; then say "+ GET $path in deploy/$NX_DEPLOY"; return 0; fi
  if [ -z "$name" ]; then
    nx_get "$NX_NS" "$path" | jq --arg p "$SLUG-" '[.[] | select((.name // .id // .userId) | startswith($p))]'
  else
    nx_get "$NX_NS" "$path"
  fi
}
