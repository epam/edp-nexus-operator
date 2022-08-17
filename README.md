[![codecov](https://codecov.io/gh/epam/edp-nexus-operator/branch/master/graph/badge.svg?token=JB9ZT0PCDJ)](https://codecov.io/gh/epam/edp-nexus-operator)

# Nexus Operator

| :heavy_exclamation_mark: Please refer to [EDP documentation](https://epam.github.io/edp-install/) to get the notion of the main concepts and guidelines. |
| --- |

Get acquainted with the Nexus Operator and the installation process as well as the local development, and architecture scheme.

## Overview

Nexus Operator is an EDP operator that is responsible for installing and configuring Nexus. Operator installation can be applied on two container orchestration platforms: OpenShift and Kubernetes.

_**NOTE:** Operator is platform-independent, that is why there is a unified instruction for deploying._

## Prerequisites

1. Linux machine or Windows Subsystem for Linux instance with [Helm 3](https://helm.sh/docs/intro/install/) installed;
2. Cluster admin access to the cluster;
3. EDP project/namespace is deployed by following one the [Install EDP](https://epam.github.io/edp-install/operator-guide/install-edp/) instruction.

## Installation

In order to install the Nexus operator, follow the steps below:

1. To add the Helm EPAMEDP Charts for local client, run "helm repo add":
     ```bash
     helm repo add epamedp https://epam.github.io/edp-helm-charts/stable
     ```
2. Choose available Helm chart version:
     ```bash
     helm search repo epamedp/nexus-operator -l
     NAME                        CHART VERSION   APP VERSION     DESCRIPTION
     epamedp/nexus-operator      2.11.0          2.11.0          A Helm chart for EDP Nexus Operator
     epamedp/nexus-operator      2.10.0          2.10.0          A Helm chart for EDP Nexus Operator
     ```

    _**NOTE:** It is highly recommended to use the latest released version._

3. Full chart parameters available in [deploy-templates/README.md](deploy-templates/README.md).

4. Install operator in the <edp-project> namespace with the helm command; find below the installation command example:
    ```bash
    helm install nexus-operator epamedp/nexus-operator --version <chart_version> --namespace <edp-project> --set name=nexus-operator --set global.edpName=<edp-project> --set global.platform=<platform_type> --set global.dnsWildCard=<cluster_DNS_wildcard>
    ```
5. Check the <edp-project> namespace that should contain operator deployment with your operator in a running status.

## Local Development

In order to develop the operator, first set up a local environment. For details, please refer to the [Local Development](https://epam.github.io/edp-install/developer-guide/local-development/) page.

Development versions are also available, please refer to the [snapshot helm chart repository](https://epam.github.io/edp-helm-charts/snapshot/) page.

To generate deep copy files you can use [deepcopy-gen](https://pkg.go.dev/k8s.io/gengo/examples/deepcopy-gen) tool
Example of usage: `deepcopy-gen -i ./v1alpha1 -o ./v1alpha1`

### Related Articles

* [Architecture Scheme of Nexus Operator](documentation/arch.md)
* [Install EDP](https://epam.github.io/edp-install/operator-guide/install-edp/)