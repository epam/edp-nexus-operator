[![codecov](https://codecov.io/gh/epam/edp-nexus-operator/branch/master/graph/badge.svg?token=JB9ZT0PCDJ)](https://codecov.io/gh/epam/edp-nexus-operator)

# Nexus Operator

| :heavy_exclamation_mark: Please refer to [KubeRocketCI documentation](https://docs.kuberocketci.io/) to get the notion of the main concepts and guidelines. |
| --- |

Get acquainted with the Nexus Operator and the installation process as well as the local development, and architecture scheme.

## Overview

Nexus Operator is a KubeRocketCI operator that is responsible for configuring Nexus.

_**NOTE:** Operator is platform-independent, that is why there is a unified instruction for deploying._

## Prerequisites

1. Linux machine or Windows Subsystem for Linux instance with [Helm 3](https://helm.sh/docs/intro/install/) installed;
2. Cluster admin access to the cluster;

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
    epamedp/nexus-operator      3.3.0           3.3.0           A Helm chart for KRCI Nexus Operator
    ```

    _**NOTE:** It is highly recommended to use the latest released version._

3. Full chart parameters available in [deploy-templates/README.md](deploy-templates/README.md).

4. Install operator in the `nexus-operator` namespace with the helm command; find below the installation command example:

    ```bash
    helm install nexus-operator epamedp/nexus-operator --version <chart_version> --namespace nexus
    ```

5. Check the `nexus-operator` namespace that should contain operator deployment with your operator in a running status.

## Quick Start

1. Login into Nexus and create user. Attach permissions to user such as scripts, rules, blobs etc. Insert user credentials into Kubernetes secret.

    ```yaml
    apiVersion: v1
    kind: Secret
    metadata:
      name: nexus-admin-password
    data:
      password: cGFzcw==  # base64-encoded value of "pass"
      user:     dXNlcg==  # base64-encoded value of "user"
    ```

2. Create Custom Resource `kind: Nexus` with Nexus instance URL and secret created on the previous step:

    ```yaml
    apiVersion: edp.epam.com/v1alpha1
    kind: Nexus
    metadata:
      name: nexus
    spec:
      secret: nexus-admin-password
      url: http://nexus.example.com
    ```

    Wait for the `.status` field with  `status.connected: true`

3. Create Role using Custom Resources NexusRole:

    ```yaml
    apiVersion: edp.epam.com/v1alpha1
    kind: NexusRole
    metadata:
      name: edp-admin
    spec:
      description: Read and write access to all repos and scripts
      id: edp-admin
      name: edp-admin
      nexusRef:
        kind: Nexus
        name: nexus
      privileges:
        - nx-apikey-all
        - nx-repository-view-*-*-add
        - nx-repository-view-*-*-browse
        - nx-repository-view-*-*-edit
        - nx-repository-view-*-*-read
        - nx-script-*-add
        - nx-script-*-delete
        - nx-script-*-run
        - nx-search-read
    ```

    Inspect [CR templates folder](./deploy-templates/_crd_examples/) for more examples.

## Local Development

In order to develop the operator, first set up a local environment. For details, please refer to the [Local Development](https://docs.kuberocketci.io/docs/developer-guide/local-development) page.

Development versions are also available, please refer to the [snapshot Helm Chart repository](https://epam.github.io/edp-helm-charts/snapshot/) page.

### Related Articles

* [Install KubeRocketCI](https://docs.kuberocketci.io/docs/operator-guide/install-kuberocketci)
