package client

import (
	tekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	kneventing "knative.dev/eventing/pkg/client/clientset/versioned"
	knserving "knative.dev/serving/pkg/client/clientset/versioned"
)

type Client struct {
	Kclient    *kubernetes.Clientset
	Knserving  *knserving.Clientset
	Kneventing *kneventing.Clientset
	Tekton     *tekton.Clientset
}
