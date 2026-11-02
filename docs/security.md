# Security considerations

## Trust boundary: Secret references

The operator reads Kubernetes Secrets with its own ServiceAccount. The chart grants it `get`, `list` and `watch` on
Secrets in its namespace, including the webhook certificate Secret `edp-nexus-operator-webhook-certs`. A custom
resource that names a Secret makes the operator, or Nexus, use that Secret on the author's behalf.

The operator does not restrict the addresses a resource names. No destination allowlist exists. RBAC on the kinds
below is the only control.

Anyone who can create or update `Nexus`, `NexusRepository` or `NexusBlobStore` in a namespace can make the operator
or Nexus send any Secret key of that namespace that the operator can read to an address the author chooses.

## Impacted resources

| Kind | Address field | What is sent there | Sent by |
|---|---|---|---|
| `Nexus` | `spec.url` | admin user and password from `spec.secret` | operator |
| `NexusRepository` (every proxy format) | `proxy.remoteUrl`; Docker `dockerProxy.indexUrl` | proxy `username` and the password (`password` or `passwordRef`) | Nexus |
| `NexusBlobStore` (S3) | `s3.advancedBucketConnection.endpoint` | access key id and session token (the secret key signs requests) | Nexus |
| `NexusUser` (indirect) | none; `nexusRef` points to `Nexus.spec.url` | user password from its Secret | operator |
| `NexusScript` (indirect) | none | arbitrary Groovy that runs on Nexus and can call any URL; no operator Secret access | Nexus |

`NexusRole` and `NexusCleanupPolicy` read no Secret.

## What a resource author can read

Any Secret key in the namespace of the resource that the operator ServiceAccount can read. Nothing limits the
address that receives it.

## Who must be aware

- Cluster admins and anyone who grants `create` or `update` on the kinds above, or `get` on Secrets.
- Bind the admin helper role of `Nexus`, `NexusRepository` and `NexusBlobStore` only to trusted platform admins. Do not
  give namespace tenants `create` on these kinds.
- When a GitOps tool applies these resources, every committer to its source repository is an author.

## Helper roles

- Kustomize ships `nexus-admin-role`, `nexusrepository-admin-role`, `nexusblobstore-admin-role` and the matching
  viewer roles. There are no editor roles.
- The Helm chart always ships the same roles as ClusterRoles named `edp-<name>-<namespace>-<kind>-{admin,viewer}`.
  There is no value to switch them off.
- Only the admin role may create, update, patch or delete `Nexus`, `NexusRepository` and `NexusBlobStore`. It lists
  explicit verbs; it does not grant `deletecollection`. Bind it to platform admins only.
- The viewer role grants `get`, `list` and `watch`. A viewer can read the resource and see the name and key of a
  referenced Secret. A viewer cannot read the Secret value and cannot change the resource.
- No helper role is bound or aggregated into the built-in `admin`, `edit` and `view` roles.
- The operator ServiceAccount has no `create` or `delete` on these three kinds. It keeps `get`, `list`, `watch`,
  `update` and `patch`, so it can still change an existing resource.

## Controls outside the operator

- NetworkPolicy egress for the Nexus pod.
- An admission policy (Kyverno, Gatekeeper or ValidatingAdmissionPolicy) that restricts the hosts in `remoteUrl`,
  `indexUrl`, `endpoint` and `Nexus.spec.url`.
- A dedicated namespace for these resources that holds no other Secrets.

## Redirects

The HTTP client from the operator to Nexus uses the Go default redirect policy. It has no redirect hardening.

## Secret value handling

- `passwordRef` is read on every reconcile. The value is sent to Nexus in the request body only. It is not written to
  the resource, its status, events, logs or error messages. The operator replaces the value with `<redacted>` in an
  error text returned by Nexus.
- A `passwordRef` value that is empty or has leading or trailing whitespace is rejected; the error names the Secret
  and the key.
- `httpClient.authentication.password` stores the value in clear text in the resource. It is deprecated.
