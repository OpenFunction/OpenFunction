module github.com/openfunction

go 1.16

require (
	github.com/dapr/dapr v1.3.1
	github.com/go-logr/logr v1.2.0
	github.com/json-iterator/go v1.1.11
	github.com/kedacore/keda/v2 v2.4.0
	github.com/mitchellh/hashstructure v1.1.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/shipwright-io/build v0.6.0
	go.uber.org/zap v1.19.0
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	knative.dev/serving v0.26.0
	sigs.k8s.io/controller-runtime v0.9.7
	sigs.k8s.io/gateway-api v0.4.0
)

replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.4.0
	k8s.io/api => k8s.io/api v0.21.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.4
	k8s.io/client-go => k8s.io/client-go v0.21.4
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.9.0
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.9.7
)
