module github.com/epmd-edp/nexus-operator/v2

go 1.12

replace git.apache.org/thrift.git => github.com/apache/thrift v0.12.0

replace github.com/openshift/api => github.com/openshift/api v0.0.0-20180801171038-322a19404e37

require (
	github.com/coreos/prometheus-operator v0.29.0 // indirect
	github.com/dchest/uniuri v0.0.0-20160212164326-8902c56451e9
	github.com/epmd-edp/edp-component-operator v0.0.1-2
	github.com/epmd-edp/jenkins-operator/v2 v2.2.0-92
	github.com/epmd-edp/keycloak-operator v1.0.31-alpha-56
	github.com/go-openapi/spec v0.19.3
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/operator-framework/operator-sdk v0.0.0-20190530173525-d6f9cdf2f52e
	github.com/pkg/errors v0.8.1
	github.com/spf13/pflag v1.0.3
	gopkg.in/resty.v1 v1.12.0
	k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/client-go v0.0.0-20190228174230-b40b2a5939e4
	k8s.io/code-generator v0.0.0-20191003035328-700b1226c0bd
	k8s.io/gengo v0.0.0-20190907103519-ebc107f98eab // indirect
	k8s.io/kube-openapi v0.0.0-20190816220812-743ec37842bf
	sigs.k8s.io/controller-runtime v0.1.12
)
