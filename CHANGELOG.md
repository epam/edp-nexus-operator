<a name="unreleased"></a>
## [Unreleased]


<a name="v3.5.0"></a>
## [v3.5.0] - 2025-04-02
### Routine

- Add ImagePullSecrets field support ([#59](https://github.com/epam/edp-nexus-operator/issues/59))
- Update current development version ([#61](https://github.com/epam/edp-nexus-operator/issues/61))


<a name="v3.4.0"></a>
## [v3.4.0] - 2025-03-26
### Features

- Implement password update for NexusUser ([#51](https://github.com/epam/edp-nexus-operator/issues/51))

### Routine

- Update k8s version for tests ([#53](https://github.com/epam/edp-nexus-operator/issues/53))
- Publish release 3.3.0 on the OperatorHub ([#47](https://github.com/epam/edp-nexus-operator/issues/47))
- Update current development version ([#45](https://github.com/epam/edp-nexus-operator/issues/45))


<a name="v3.3.0"></a>
## [v3.3.0] - 2024-12-12
### Features

- Add support for S3 Nexus blobstore configuration ([#43](https://github.com/epam/edp-nexus-operator/issues/43))

### Routine

- Update Pull Request Template ([#10](https://github.com/epam/edp-nexus-operator/issues/10))
- Update KubeRocketCI names and documentation links ([#37](https://github.com/epam/edp-nexus-operator/issues/37))
- Bump to Go 1.22 ([#35](https://github.com/epam/edp-nexus-operator/issues/35))
- Add codeowners file to the repo ([#31](https://github.com/epam/edp-nexus-operator/issues/31))
- Migrate from gerrit to github pipelines ([#29](https://github.com/epam/edp-nexus-operator/issues/29))
- Setup setup-envtest version instead latest ([#16](https://github.com/epam/edp-nexus-operator/issues/16))
- Prepare bundle version 3.2.0 for OperatorHub ([#26](https://github.com/epam/edp-nexus-operator/issues/26))
- Update current development version ([#26](https://github.com/epam/edp-nexus-operator/issues/26))

### Documentation

- Fix broken format in README md ([#39](https://github.com/epam/edp-nexus-operator/issues/39))
- Fix README.md ([#28](https://github.com/epam/edp-nexus-operator/issues/28))
- Add the Quick Start section to the README ([#28](https://github.com/epam/edp-nexus-operator/issues/28))


<a name="v3.2.0"></a>
## [v3.2.0] - 2024-02-21
### Features

- Add NexusCleanupPolicy Custom Resource ([#25](https://github.com/epam/edp-nexus-operator/issues/25))
- Add resources for integration tests ([#22](https://github.com/epam/edp-nexus-operator/issues/22))
- Add NexusScript execute property ([#21](https://github.com/epam/edp-nexus-operator/issues/21))
- Add NexusBlobStore custom resource ([#20](https://github.com/epam/edp-nexus-operator/issues/20))

### Bug Fixes

- Creation of the nexus repository with default values failed ([#24](https://github.com/epam/edp-nexus-operator/issues/24))
- The repository type pypi hosted is not created ([#23](https://github.com/epam/edp-nexus-operator/issues/23))

### Routine

- Generate bundle for v3.1.0 OperatorHub ([#19](https://github.com/epam/edp-nexus-operator/issues/19))
- Update current development version ([#19](https://github.com/epam/edp-nexus-operator/issues/19))

### Documentation

- Update README md file ([#132](https://github.com/epam/edp-nexus-operator/issues/132))


<a name="v3.1.0"></a>
## [v3.1.0] - 2024-01-11
### Features

- Add NexusRepository validation ([#18](https://github.com/epam/edp-nexus-operator/issues/18))
- Add NexusScript custom resource ([#17](https://github.com/epam/edp-nexus-operator/issues/17))

### Routine

- Update sonar properties for the project ([#17](https://github.com/epam/edp-nexus-operator/issues/17))
- Generate bundle for OperatorHub ([#16](https://github.com/epam/edp-nexus-operator/issues/16))
- Update current development version ([#16](https://github.com/epam/edp-nexus-operator/issues/16))


<a name="v3.0.0"></a>
## [v3.0.0] - 2023-12-11
### Features

- Add NexusRepository custom resource ([#14](https://github.com/epam/edp-nexus-operator/issues/14))
- Automate rekor uuid in release tag ([#13](https://github.com/epam/edp-nexus-operator/issues/13))
- Add configuration for operator publishing ([#12](https://github.com/epam/edp-nexus-operator/issues/12))
- Add NexusUser custom resource ([#9](https://github.com/epam/edp-nexus-operator/issues/9))
- Add NexusRole custom resource ([#8](https://github.com/epam/edp-nexus-operator/issues/8))
- Refactor Nexus custom resource ([#3](https://github.com/epam/edp-nexus-operator/issues/3))

### Bug Fixes

- Race condition in go-resty package v2.10.0 ([#15](https://github.com/epam/edp-nexus-operator/issues/15))

### Routine

- Upgrade golang.org/x/net to version 0.17.0 ([#11](https://github.com/epam/edp-nexus-operator/issues/11))
- Upgrade pull request template ([#10](https://github.com/epam/edp-nexus-operator/issues/10))
- Update current development version ([#6](https://github.com/epam/edp-nexus-operator/issues/6))


<a name="v2.17.0"></a>
## [v2.17.0] - 2023-09-20
### Code Refactoring

- Remove deprecated edpName parameter ([#5](https://github.com/epam/edp-nexus-operator/issues/5))

### Routine

- Update groovy api script to receive get-nuget-token in version 3.59.0 ([#4](https://github.com/epam/edp-nexus-operator/issues/4))
- Update current development version ([#3](https://github.com/epam/edp-nexus-operator/issues/3))


<a name="v2.16.0"></a>
## [v2.16.0] - 2023-08-17

[Unreleased]: https://github.com/epam/edp-nexus-operator/compare/v3.5.0...HEAD
[v3.5.0]: https://github.com/epam/edp-nexus-operator/compare/v3.4.0...v3.5.0
[v3.4.0]: https://github.com/epam/edp-nexus-operator/compare/v3.3.0...v3.4.0
[v3.3.0]: https://github.com/epam/edp-nexus-operator/compare/v3.2.0...v3.3.0
[v3.2.0]: https://github.com/epam/edp-nexus-operator/compare/v3.1.0...v3.2.0
[v3.1.0]: https://github.com/epam/edp-nexus-operator/compare/v3.0.0...v3.1.0
[v3.0.0]: https://github.com/epam/edp-nexus-operator/compare/v2.17.0...v3.0.0
[v2.17.0]: https://github.com/epam/edp-nexus-operator/compare/v2.16.0...v2.17.0
[v2.16.0]: https://github.com/epam/edp-nexus-operator/compare/v2.15.0...v2.16.0
