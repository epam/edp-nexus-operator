# Nexus Operator

Get acquainted with the Nexus Operator and the installation process as well as the local development, 
and architecture scheme.

## Overview

Nexus Operator is an EDP operator that is responsible for installing and configuring Nexus. 
Operator installation can be applied on two container orchestration platforms: OpenShift and Kubernetes.

_**NOTE:** Operator is platform-independent, that is why there is a unified instruction for deploying._

## Prerequisites
1. Linux machine or Windows Subsystem for Linux instance with [Helm 3](https://helm.sh/docs/intro/install/) installed;
2. Cluster admin access to the cluster;
3. EDP project/namespace is deployed by following one of the instructions: [edp-install-openshift](https://github.com/epam/edp-install/blob/master/documentation/openshift_install_edp.md#edp-project) or [edp-install-kubernetes](https://github.com/epam/edp-install/blob/master/documentation/kubernetes_install_edp.md#edp-namespace).

## Installation
In order to install the Nexus operator, follow the steps below:

1. To add the Helm EPAMEDP Charts for local client, run "helm repo add":
     ```bash
     helm repo add epamedp https://chartmuseum.demo.edp-epam.com/
     ```
2. Choose available Helm chart version:
     ```bash
     helm search repo epamedp/nexus-operator
     NAME                        CHART VERSION   APP VERSION     DESCRIPTION
     epamedp/nexus-operator      v2.4.0                          Helm chart for Golang application/service deplo...
     ```
   
    _**NOTE:** It is highly recommended to use the latest released version._
        
3. Deploy operator:

   Full available chart parameters list:
   ```
    - chart_version                                 # a version of CD Pipeline operator Helm chart;
    - global.edpName                                # a namespace or a project name (in case of OpenShift);
    - global.platform                               # openshift or kubernetes;
    - global.admins                                 # Administrators of your tenant separated by comma (,) (eg --set 'global.admins={test@example.com}');
    - global.dnsWildCard                            # Developers of your tenant separated by comma (,) (eg --set 'global.developers={test@example.com}');
    - image.name                                    # EDP image. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/nexus-operator);
    - image.version                                 # EDP tag. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/nexus-operator/tags);
    - nexus.deploy                                  # Flag to enable/disable Nexus deploy;
    - nexus.name                                    # Nexus name;
    - nexus.image                                   # Image for Nexus. The image can be found on [Dockerhub] (https://hub.docker.com/r/sonatype/nexus3);
    - nexus.version                                 # Nexus version. The released version can be found on [Dockerhub](https://hub.docker.com/r/sonatype/nexus3/tags)'
    - nexus.basePath                                # Base path for Nexus URL;
    - nexus.imagePullSecrets                        # Secrets to pull from private Docker registry;
    - nexus.storage.class                           # Storageclass for Nexus data volume. Default is "gp2";
    - nexus.storage.size                            # Nexus data volume capacity. Default is "10Gi";

    ```

4. Install operator in the <edp_cicd_project> namespace with the helm command; find below the installation command example:
    ```bash
    helm install nexus-operator epamedp/nexus-operator --version <chart_version> --namespace <edp_cicd_project> --set name=nexus-operator --set global.edpName=<edp_cicd_project> --set global.platform=<platform_type> --set global.dnsWildCard=<cluster_DNS_wildcard>
    ```
5. Check the <edp_cicd_project> namespace that should contain operator deployment with your operator in a running status.

## Local Development
In order to develop the operator, first set up a local environment. For details, please refer to the [Local Development](documentation/local-development.md) page.

### Related Articles

- [Architecture Scheme of Nexus Operator](documentation/arch.md)