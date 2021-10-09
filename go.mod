module github.com/openfunction

go 1.15

require (
	github.com/OpenFunction/functions-framework-go v0.0.0-20210922063920-81a7b2951b8a
	github.com/dapr/dapr v1.2.2
	github.com/go-logr/logr v0.4.0
	github.com/json-iterator/go v1.1.11
	github.com/kedacore/keda/v2 v2.2.0
	github.com/mitchellh/hashstructure v1.1.0
	github.com/onsi/ginkgo v1.16.2
	github.com/onsi/gomega v1.12.0
	github.com/shipwright-io/build v0.5.2-0.20210715083206-5d8fb411a1eb
	go.uber.org/zap v1.17.0
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	knative.dev/eventing v0.23.0
	knative.dev/pkg v0.0.0-20210520062216-e749d6a2ad0e // indirect
	knative.dev/serving v0.23.0
	sigs.k8s.io/controller-runtime v0.8.3
)

replace (
	github.com/buger/jsonparser => github.com/buger/jsonparser v1.0.0
	github.com/containerd/containerd => github.com/containerd/containerd v1.4.11
	github.com/dgrijalva/jwt-go => github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gin-gonic/gin => github.com/gin-gonic/gin v1.7.0
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc95
	github.com/tektoncd/pipeline => github.com/tektoncd/pipeline v0.19.0
	github.com/ulikunitz/xz => github.com/ulikunitz/xz v0.5.8
	go.mongodb.org/mongo-driver => go.mongodb.org/mongo-driver v1.5.1
	helm.sh/helm/v3 => helm.sh/helm/v3 v3.6.1
	k8s.io/api => k8s.io/api v0.20.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.4
	//k8s.io/client => github.com/kubernetes-client/go v0.0.0-20200222171647-9dac5e4c5400
	k8s.io/client-go => k8s.io/client-go v0.20.4
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.8.3
)
