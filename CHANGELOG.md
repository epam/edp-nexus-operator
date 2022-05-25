<a name="unreleased"></a>
## [Unreleased]


<a name="v2.11.0"></a>
## [v2.11.0] - 2022-05-10
### Features

- add nexus user to k8s roles [EPMDEDP-8086](https://jiraeu.epam.com/browse/EPMDEDP-8086)
- implement NexusUser custom resource [EPMDEDP-8086](https://jiraeu.epam.com/browse/EPMDEDP-8086)
- Update Makefile changelog target [EPMDEDP-8218](https://jiraeu.epam.com/browse/EPMDEDP-8218)
- Add ingress tls certificate option when using ingress controller [EPMDEDP-8377](https://jiraeu.epam.com/browse/EPMDEDP-8377)
- Generate CRDs and helm docs automatically [EPMDEDP-8385](https://jiraeu.epam.com/browse/EPMDEDP-8385)
- Check nexus user exists before creation [EPMDEDP-8941](https://jiraeu.epam.com/browse/EPMDEDP-8941)

### Bug Fixes

- Fix changelog generation in GH Release Action [EPMDEDP-8468](https://jiraeu.epam.com/browse/EPMDEDP-8468)
- Disable anon access to admin ui [EPMDEDP-8606](https://jiraeu.epam.com/browse/EPMDEDP-8606)

### Code Refactoring

- Remove undefined values from helm [EPMDEDP-6758](https://jiraeu.epam.com/browse/EPMDEDP-6758)

### Testing

- Add tests and mocks [EPMDEDP-7995](https://jiraeu.epam.com/browse/EPMDEDP-7995)
- Add tests and mocks [EPMDEDP-7995](https://jiraeu.epam.com/browse/EPMDEDP-7995)
- GitHub run test fix [EPMDEDP-7995](https://jiraeu.epam.com/browse/EPMDEDP-7995)
- Add tests and mock [EPMDEDP-7995](https://jiraeu.epam.com/browse/EPMDEDP-7995)
- Add tests and mocks [EPMDEDP-7995](https://jiraeu.epam.com/browse/EPMDEDP-7995)
- Add tests [EPMDEDP-7995](https://jiraeu.epam.com/browse/EPMDEDP-7995)

### Routine

- Update Ingress resources to the newest API version [EPMDEDP-7476](https://jiraeu.epam.com/browse/EPMDEDP-7476)
- Update release CI pipelines [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Fix CI for codecov report [EPMDEDP-7995](https://jiraeu.epam.com/browse/EPMDEDP-7995)
- Populate chart with Artifacthub annotations [EPMDEDP-8049](https://jiraeu.epam.com/browse/EPMDEDP-8049)
- Update changelog [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)
- Update Nexus image version [EPMDEDP-8839](https://jiraeu.epam.com/browse/EPMDEDP-8839)
- Update base docker image to alpine 3.15.4 [EPMDEDP-8853](https://jiraeu.epam.com/browse/EPMDEDP-8853)
- Update changelog [EPMDEDP-9185](https://jiraeu.epam.com/browse/EPMDEDP-9185)


<a name="v2.10.0"></a>
## [v2.10.0] - 2021-12-06
### Features

- Provide operator's build information [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)

### Bug Fixes

- Provide Nexus deploy through deployments on OKD cluster [EPMDEDP-7178](https://jiraeu.epam.com/browse/EPMDEDP-7178)
- Changelog links [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)

### Code Refactoring

- Expand nexus operator role [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Add namespace field in roleRef in OKD RB, align CRB name [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Replace cluster-wide role/rolebinding to namespaced [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Address golangci-lint issues [EPMDEDP-7945](https://jiraeu.epam.com/browse/EPMDEDP-7945)

### Routine

- Upgrade Nexus to the LTS 3.36.0 version [EPMDEDP-7778](https://jiraeu.epam.com/browse/EPMDEDP-7778)
- Add changelog generator [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Add codecov report [EPMDEDP-7885](https://jiraeu.epam.com/browse/EPMDEDP-7885)
- Update docker image [EPMDEDP-7895](https://jiraeu.epam.com/browse/EPMDEDP-7895)
- Use custom go build step for operator [EPMDEDP-7932](https://jiraeu.epam.com/browse/EPMDEDP-7932)
- Update go to version 1.17 [EPMDEDP-7932](https://jiraeu.epam.com/browse/EPMDEDP-7932)

### Documentation

- Update the links on GitHub [EPMDEDP-7781](https://jiraeu.epam.com/browse/EPMDEDP-7781)


<a name="v2.9.0"></a>
## [v2.9.0] - 2021-12-03

<a name="v2.8.1"></a>
## [v2.8.1] - 2021-12-03

<a name="v2.8.0"></a>
## [v2.8.0] - 2021-12-03

<a name="v2.7.1"></a>
## [v2.7.1] - 2021-12-03

<a name="v2.7.0"></a>
## [v2.7.0] - 2021-12-03
### Reverts

- test


[Unreleased]: https://github.com/epam/edp-nexus-operator/compare/v2.11.0...HEAD
[v2.11.0]: https://github.com/epam/edp-nexus-operator/compare/v2.10.0...v2.11.0
[v2.10.0]: https://github.com/epam/edp-nexus-operator/compare/v2.9.0...v2.10.0
[v2.9.0]: https://github.com/epam/edp-nexus-operator/compare/v2.8.1...v2.9.0
[v2.8.1]: https://github.com/epam/edp-nexus-operator/compare/v2.8.0...v2.8.1
[v2.8.0]: https://github.com/epam/edp-nexus-operator/compare/v2.7.1...v2.8.0
[v2.7.1]: https://github.com/epam/edp-nexus-operator/compare/v2.7.0...v2.7.1
[v2.7.0]: https://github.com/epam/edp-nexus-operator/compare/v2.3.0-63...v2.7.0
