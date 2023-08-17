# nexus-operator

![Version: 2.17.0-SNAPSHOT](https://img.shields.io/badge/Version-2.17.0--SNAPSHOT-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 2.17.0-SNAPSHOT](https://img.shields.io/badge/AppVersion-2.17.0--SNAPSHOT-informational?style=flat-square)

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
| global.keycloakUrl | string | `nil` | Keycloak Endpoint which is used for SSO integration. Format https://keycloak.example.com |
| global.openshift.deploymentType | string | `"deployments"` | Which type of kind will be deployed to Openshift (values: deployments/deploymentConfigs) |
| global.platform | string | `"kubernetes"` | platform type that can be "kubernetes" or "openshift" |
| image.repository | string | `"epamedp/nexus-operator"` | EDP nexus-operator Docker image name. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/nexus-operator) |
| image.tag | string | `nil` | EDP nexus-operator Docker image tag. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/nexus-operator/tags) |
| imagePullPolicy | string | `"IfNotPresent"` |  |
| name | string | `"nexus-operator"` | component name |
| nexus.affinity | object | `{}` |  |
| nexus.annotations | object | `{}` |  |
| nexus.deploy | bool | `true` | Flag to enable/disable Nexus deploy |
| nexus.env | list | `[{"name":"INSTALL4J_ADD_VM_PARAMS","value":"-Xms2703M -Xmx2703M\n-XX:MaxDirectMemorySize=2703M\n-XX:+UnlockExperimentalVMOptions\n-XX:+UseCGroupMemoryLimitForHeap\n-Djava.util.prefs.userRoot=/nexus-data/javaprefs"}]` | Custom environment variables to be used by nexus pod |
| nexus.image | string | `"sonatype/nexus3"` | Image for Nexus. The image can be found on [Dockerhub] (https://hub.docker.com/r/sonatype/nexus3) |
| nexus.imagePullPolicy | string | `"IfNotPresent"` |  |
| nexus.imagePullSecrets | string | `nil` | Secrets to pull from private Docker registry |
| nexus.name | string | `"nexus"` | Nexus name |
| nexus.nodeSelector | object | `{}` |  |
| nexus.resources.limits.memory | string | `"6Gi"` |  |
| nexus.resources.requests.cpu | string | `"100m"` |  |
| nexus.resources.requests.memory | string | `"2Gi"` |  |
| nexus.storage.size | string | `"10Gi"` | Nexus data volume capacity |
| nexus.tolerations | list | `[]` |  |
| nexus.version | string | `"3.58.1"` | Nexus version. The released version can be found on [Dockerhub](https://hub.docker.com/r/sonatype/nexus3/tags) |
| nexusCR.create | bool | `true` | Specifies whether Nexus CR should be created |
| nodeSelector | object | `{}` |  |
| oauth2_proxy.affinity | object | `{}` |  |
| oauth2_proxy.annotations | object | `{}` |  |
| oauth2_proxy.enabled | bool | `true` | Install oauth2-proxy as a part of nexus deployment. Default: true |
| oauth2_proxy.existingSecret.secretKey | string | `"cookie-secret"` | Secret key which stores cookie-secret |
| oauth2_proxy.existingSecret.secretName | string | `"nexus-proxy-cookie-secret"` | Secret name which stores cookie-secret |
| oauth2_proxy.extraArgs | object | `{}` |  |
| oauth2_proxy.extraEnv | list | `[]` |  |
| oauth2_proxy.extraVolumeMounts | list | `[]` | Additional volumeMounts to be added to the oauth2-proxy container |
| oauth2_proxy.extraVolumes | list | `[]` | Additional volumes to be added to the oauth2-proxy pod |
| oauth2_proxy.image.repository | string | `"quay.io/oauth2-proxy/oauth2-proxy"` | oauth2-proxy image repository |
| oauth2_proxy.image.tag | string | `"v7.4.0"` | oauth2-proxy image tag |
| oauth2_proxy.ingress.annotations | object | `{}` |  |
| oauth2_proxy.ingress.pathType | string | `"Prefix"` | pathType is only for k8s >= 1.1= |
| oauth2_proxy.ingress.tls | list | `[]` | See https://kubernetes.io/blog/2020/04/02/improvements-to-the-ingress-api-in-kubernetes-1.18/#specifying-the-class-of-an-ingress ingressClassName: nginx |
| oauth2_proxy.nodeSelector | object | `{}` |  |
| oauth2_proxy.resources.limits.cpu | string | `"100m"` |  |
| oauth2_proxy.resources.limits.memory | string | `"300Mi"` |  |
| oauth2_proxy.resources.requests.cpu | string | `"100m"` |  |
| oauth2_proxy.resources.requests.memory | string | `"300Mi"` |  |
| oauth2_proxy.tolerations | list | `[]` |  |
| resources.limits.memory | string | `"192Mi"` |  |
| resources.requests.cpu | string | `"50m"` |  |
| resources.requests.memory | string | `"64Mi"` |  |
| tolerations | list | `[]` |  |

