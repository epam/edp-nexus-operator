# Upgrade

## To the release with `passwordRef`

### Apply the CRD

Helm does not upgrade the `crds/` directory. Apply the `NexusRepository` CRD before the chart upgrade. CRDs are
cluster-wide: run it once per cluster.

```sh
kubectl replace -f https://raw.githubusercontent.com/epam/edp-nexus-operator/<tag>/deploy-templates/crds/edp.epam.com_nexusrepositories.yaml
```

A `replace` that reports `NotFound`: run `kubectl create -f` with the same URL.

With the old CRD, `passwordRef` is pruned and a resource that sets it without `password` is rejected
(`password: Required value`).

### Behaviour changes

- `httpClient.authentication` of every proxy format takes `passwordRef` (`name` and `key` of a Secret in the namespace
  of the resource). `password` is no longer required by the schema. The CRD requires exactly one of `password` and
  `passwordRef`; an empty `password` counts as not set. `username` is unchanged.
- A Secret value that is empty or has leading or trailing whitespace is rejected. Write the Secret without a trailing
  newline, for example `printf '%s' "$PASSWORD"`, or use `stringData`.
- The operator reconciles a `NexusRepository` when the Secret named by its `passwordRef` is created, deleted or
  changes data.
- Maven proxy authentication needs complete credentials: a `username` requires a password source and a password
  requires a `username`; an empty `username` counts as not set. A block with only `type` and `preemptive` stays valid in
  the CRD; Nexus rejects that request.
- `password` is deprecated. The validating webhook warns when it is set; with `ENABLE_WEBHOOKS=false` there is no
  warning. Nothing is rejected for using it.
- Nexus error text that contains the resolved `passwordRef` value shows `<redacted>` in `status.error`.

### Stored objects

Objects stored under the old schema stay valid through CRD validation ratcheting, which is on by default from
Kubernetes 1.30. Without it (earlier versions, or the `CRDValidationRatcheting` feature gate off), every update of a
stored object that the new rules reject fails, including the finalizer updates of the operator. Correct such objects
before the CRD is replaced.

- An edit of an unrelated field is accepted for every stored object, including a Maven proxy with `username` only. The
  API server returns a warning for the invalid authentication block.
- An edit of the authentication block of such a Maven proxy, or of a non-Maven proxy with `password: ""` and no
  `passwordRef`, is rejected until the block is complete.
- The operator reports `one of password or passwordRef must be set` in `status.error` for a stored non-Maven proxy
  with `password: ""` and no `passwordRef`.

List the objects the new rules affect before the upgrade.

Maven proxies with a username and no password, or a password and no username:

```sh
kubectl get nexusrepositories -A -o json | jq -r '.items[] | select(.spec.maven.proxy.httpClient.authentication as $a | $a != null and (($a.username // "") != "") != (($a.password // "") != "") and ($a.passwordRef == null)) | "\(.metadata.namespace)/\(.metadata.name)"'
```

Other proxies with authentication and an empty password:

```sh
kubectl get nexusrepositories -A -o json | jq -r '.items[] | . as $i | .spec | to_entries[] | select(.key != "maven" and (.value | type == "object" and has("proxy"))) | .value.proxy.httpClient.authentication as $a | select($a != null and ($a.password // "") == "" and $a.passwordRef == null) | "\($i.metadata.namespace)/\($i.metadata.name)"'
```

### RBAC

- Kustomize and the OperatorHub bundle: the editor roles of all kinds are removed. Rebind subjects of
  `<kind>-editor-role` to the admin role (platform admins only) or to a role you own.
- Admin roles of `Nexus`, `NexusRepository` and `NexusBlobStore` list explicit verbs (no `deletecollection`).
- Viewer roles are `ClusterRole`s for every kind. The `NexusCleanupPolicy` viewer role now targets
  `nexuscleanuppolicies`.
- Helm: new ClusterRoles `edp-<name>-<namespace>-<kind>-{admin,viewer}` for every kind (`nexus`, `nexusrepository`,
  `nexusblobstore`, `nexuscleanuppolicy`, `nexusrole`, `nexusscript`, `nexususer`). They are not
  bound. There is no value to switch them off.
- The operator ServiceAccount no longer has `create` and `delete` on `Nexus`, `NexusRepository` and `NexusBlobStore`;
  `get`, `list`, `watch`, `update` and `patch` stay. Anything that created or deleted these kinds with the operator's
  identity must use its own.
- Whoever may write these kinds can make the operator or Nexus send Secrets of the namespace to an address in the
  resource. See [security.md](security.md).
