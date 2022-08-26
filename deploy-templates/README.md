# nexus-operator

![Version: 2.12.0](https://img.shields.io/badge/Version-2.12.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 2.12.0](https://img.shields.io/badge/AppVersion-2.12.0-informational?style=flat-square)

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
| global.admins | list | `["stub_user_one@example.com"]` | Administrators of your tenant |
| global.dnsWildCard | string | `nil` | a cluster DNS wildcard name |
| global.edpName | string | `""` | namespace or a project name (in case of OpenShift) |
| global.openshift.deploymentType | string | `"deployments"` | Wich type of kind will be deployed to Openshift (values: deployments/deploymentConfigs) |
| global.platform | string | `"openshift"` | platform type that can be "kubernetes" or "openshift" |
| image.repository | string | `"epamedp/nexus-operator"` | EDP nexus-operator Docker image name. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/nexus-operator) |
| image.tag | string | `nil` | EDP nexus-operator Docker image tag. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/nexus-operator/tags) |
| imagePullPolicy | string | `"IfNotPresent"` |  |
| name | string | `"nexus-operator"` | component name |
| nexus.affinity | object | `{}` |  |
| nexus.annotations | object | `{}` |  |
| nexus.basePath | string | `""` | Base path for Nexus URL |
| nexus.deploy | bool | `true` | Flag to enable/disable Nexus deploy |
| nexus.image | string | `"sonatype/nexus3"` | Image for Nexus. The image can be found on [Dockerhub] (https://hub.docker.com/r/sonatype/nexus3) |
| nexus.imagePullPolicy | string | `"IfNotPresent"` |  |
| nexus.imagePullSecrets | string | `nil` | Secrets to pull from private Docker registry |
| nexus.ingress.annotations | object | `{}` |  |
| nexus.ingress.pathType | string | `"Prefix"` | pathType is only for k8s >= 1.1= |
| nexus.ingress.tls | list | `[]` | See https://kubernetes.io/blog/2020/04/02/improvements-to-the-ingress-api-in-kubernetes-1.18/#specifying-the-class-of-an-ingress ingressClassName: nginx |
| nexus.name | string | `"nexus"` | Nexus name |
| nexus.nodeSelector | object | `{}` |  |
| nexus.proxyImage | string | `"quay.io/keycloak/keycloak-gatekeeper:10.0.0"` |  |
| nexus.resources.limits.memory | string | `"3Gi"` |  |
| nexus.resources.requests.cpu | string | `"100m"` |  |
| nexus.resources.requests.memory | string | `"1.5Gi"` |  |
| nexus.storage.class | string | `"gp2"` | Storageclass for Nexus data volume |
| nexus.storage.size | string | `"10Gi"` | Nexus data volume capacity |
| nexus.tolerations | list | `[]` |  |
| nexus.version | string | `"3.41.0"` | Nexus version. The released version can be found on [Dockerhub](https://hub.docker.com/r/sonatype/nexus3/tags) |
| nodeSelector | object | `{}` |  |
| resources.limits.memory | string | `"192Mi"` |  |
| resources.requests.cpu | string | `"50m"` |  |
| resources.requests.memory | string | `"64Mi"` |  |
| tolerations | list | `[]` |  |

