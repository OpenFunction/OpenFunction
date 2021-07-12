package http

import (
	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openfunction/controllers/event/connector"
)

// NewHttpAdapter will create a adapter for http request.
func NewHttpAdapter(name string, namespace string, spec *componentsv1alpha1.ComponentSpec) (*connector.Connector, error) {
	rc := &connector.Connector{}
	component := &componentsv1alpha1.Component{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	component.Spec = *spec
	rc.Component = *component
	return rc, nil
}
