# nexus-operator

![Version: 2.18.0-SNAPSHOT](https://img.shields.io/badge/Version-2.18.0--SNAPSHOT-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 2.18.0-SNAPSHOT](https://img.shields.io/badge/AppVersion-2.18.0--SNAPSHOT-informational?style=flat-square)

A Helm chart for EDP Nexus Operator

**Homepage:** <https://epam.github.io/edp-install/>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| epmd-edp | <SupportEPMD-EDP@epam.com> | <https://solutionshub.epam.com/solution/epam-delivery-platform> |
| sergk |  | <https://github.com/SergK> |

## Source Code

* <https://github.com/epam/edp-nexus-operator>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| annotations | object | `{}` |  |
| image.repository | string | `"epamedp/nexus-operator"` | EDP nexus-operator Docker image name. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/nexus-operator) |
| image.tag | string | `nil` | EDP nexus-operator Docker image tag. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/nexus-operator/tags) |
| imagePullPolicy | string | `"IfNotPresent"` |  |
| name | string | `"nexus-operator"` | component name |
| nodeSelector | object | `{}` |  |
| resources.limits.memory | string | `"192Mi"` |  |
| resources.requests.cpu | string | `"50m"` |  |
| resources.requests.memory | string | `"64Mi"` |  |
| tolerations | list | `[]` |  |

