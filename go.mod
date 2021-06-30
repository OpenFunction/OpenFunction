module github.com/openfunction

go 1.15

require (
	github.com/OpenFunction/functions-framework-go v0.0.0-20210628081257-4137e46a99a6
	github.com/dapr/dapr v1.2.2
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.4.0
	github.com/json-iterator/go v1.1.10
	github.com/kedacore/keda/v2 v2.2.0
	github.com/mitchellh/hashstructure v1.1.0
	github.com/onsi/ginkgo v1.15.2
	github.com/onsi/gomega v1.11.0
	github.com/tektoncd/pipeline v0.13.1-0.20200625065359-44f22a067b75
	go.uber.org/zap v1.16.0
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	knative.dev/eventing v0.23.0
	knative.dev/pkg v0.0.0-20210520062216-e749d6a2ad0e
	knative.dev/serving v0.23.0
	sigs.k8s.io/controller-runtime v0.7.0
)

replace (
	github.com/tektoncd/pipeline => github.com/tektoncd/pipeline v0.19.0
	k8s.io/api => k8s.io/api v0.20.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.4
	k8s.io/client => github.com/kubernetes-client/go v0.0.0-20200222171647-9dac5e4c5400
	k8s.io/client-go => k8s.io/client-go v0.20.4
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.6.4
)
