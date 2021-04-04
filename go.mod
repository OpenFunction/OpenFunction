module github.com/openfunction

go 1.15

require (
	cloud.google.com/go v0.74.0 // indirect
	github.com/emicklei/go-restful v2.15.0+incompatible // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.3.0
	github.com/go-logr/zapr v0.3.0 // indirect
	github.com/go-openapi/spec v0.20.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/prometheus/client_golang v1.9.0 // indirect
	github.com/tektoncd/pipeline v0.13.1-0.20200625065359-44f22a067b75
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b // indirect
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	golang.org/x/sys v0.0.0-20201223074533-0d417f636930 // indirect
	golang.org/x/term v0.0.0-20201210144234-2321bbc49cbf // indirect
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	golang.org/x/tools v0.0.0-20201211185031-d93e913c1a58 // indirect
	google.golang.org/genproto v0.0.0-20201211151036-40ec1c210f7a // indirect
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/klog/v2 v2.4.0 // indirect
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920 // indirect
	knative.dev/eventing v0.19.4
	knative.dev/serving v0.19.0
	sigs.k8s.io/controller-runtime v0.6.4
)

replace (
	github.com/tektoncd/pipeline => github.com/tektoncd/pipeline v0.19.0
	k8s.io/api => k8s.io/api v0.18.8
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.8
	k8s.io/client-go => k8s.io/client-go v0.18.8
	knative.dev/eventing => knative.dev/eventing v0.19.4
	knative.dev/serving => knative.dev/serving v0.19.0
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.6.4
)
