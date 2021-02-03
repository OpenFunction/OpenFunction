/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ContainerSource is the Schema for the containersources API
type ContainerSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ContainerSourceSpec   `json:"spec,omitempty"`
	Status ContainerSourceStatus `json:"status,omitempty"`
}

var (
	_ runtime.Object     = (*ContainerSource)(nil)
	_ kmeta.OwnerRefable = (*ContainerSource)(nil)
	_ apis.Validatable   = (*ContainerSource)(nil)
	_ apis.Defaultable   = (*ContainerSource)(nil)
	_ apis.HasSpec       = (*ContainerSource)(nil)
	_ duckv1.KRShaped    = (*ContainerSource)(nil)
)

// ContainerSourceSpec defines the desired state of ContainerSource
type ContainerSourceSpec struct {
	// inherits duck/v1 SourceSpec, which currently provides:
	// * Sink - a reference to an object that will resolve to a domain name or
	//   a URI directly to use as the sink.
	// * CloudEventOverrides - defines overrides to control the output format
	//   and modifications of the event sent to the sink.
	duckv1.SourceSpec `json:",inline"`

	// Template describes the pods that will be created
	Template corev1.PodTemplateSpec `json:"template"`
}

// GetGroupVersionKind returns the GroupVersionKind.
func (*ContainerSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("ContainerSource")
}

// ContainerSourceStatus defines the observed state of ContainerSource
type ContainerSourceStatus struct {
	// inherits duck/v1 SourceStatus, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last
	//   processed by the controller.
	// * Conditions - the latest available observations of a resource's current
	//   state.
	// * SinkURI - the current active sink URI that has been configured for the
	//   Source.
	duckv1.SourceStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ContainerSourceList contains a list of ContainerSource
type ContainerSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ContainerSource `json:"items"`
}

// GetUntypedSpec returns the spec of the ContainerSource.
func (c *ContainerSource) GetUntypedSpec() interface{} {
	return c.Spec
}

// GetStatus retrieves the status of the ContainerSource. Implements the KRShaped interface.
func (c *ContainerSource) GetStatus() *duckv1.Status {
	return &c.Status.Status
}
