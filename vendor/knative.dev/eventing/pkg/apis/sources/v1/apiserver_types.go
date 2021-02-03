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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:defaulter-gen=true

// ApiServerSource is the Schema for the apiserversources API
type ApiServerSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApiServerSourceSpec   `json:"spec,omitempty"`
	Status ApiServerSourceStatus `json:"status,omitempty"`
}

// Check the interfaces that ApiServerSource should be implementing.
var (
	_ runtime.Object     = (*ApiServerSource)(nil)
	_ kmeta.OwnerRefable = (*ApiServerSource)(nil)
	_ apis.Validatable   = (*ApiServerSource)(nil)
	_ apis.Defaultable   = (*ApiServerSource)(nil)
	_ apis.HasSpec       = (*ApiServerSource)(nil)
	_ duckv1.KRShaped    = (*ApiServerSource)(nil)
)

// ApiServerSourceSpec defines the desired state of ApiServerSource
type ApiServerSourceSpec struct {
	// inherits duck/v1 SourceSpec, which currently provides:
	// * Sink - a reference to an object that will resolve to a domain name or
	//   a URI directly to use as the sink.
	// * CloudEventOverrides - defines overrides to control the output format
	//   and modifications of the event sent to the sink.
	duckv1.SourceSpec `json:",inline"`

	// Resource are the resources this source will track and send related
	// lifecycle events from the Kubernetes ApiServer, with an optional label
	// selector to help filter.
	// +required
	Resources []APIVersionKindSelector `json:"resources,omitempty"`

	// ResourceOwner is an additional filter to only track resources that are
	// owned by a specific resource type. If ResourceOwner matches Resources[n]
	// then Resources[n] is allowed to pass the ResourceOwner filter.
	// +optional
	ResourceOwner *APIVersionKind `json:"owner,omitempty"`

	// EventMode controls the format of the event.
	// `Reference` sends a dataref event type for the resource under watch.
	// `Resource` send the full resource lifecycle event.
	// Defaults to `Reference`
	// +optional
	EventMode string `json:"mode,omitempty"`

	// ServiceAccountName is the name of the ServiceAccount to use to run this
	// source. Defaults to default if not set.
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// ApiServerSourceStatus defines the observed state of ApiServerSource
type ApiServerSourceStatus struct {
	// inherits duck/v1 SourceStatus, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last
	//   processed by the controller.
	// * Conditions - the latest available observations of a resource's current
	//   state.
	// * SinkURI - the current active sink URI that has been configured for the
	//   Source.
	duckv1.SourceStatus `json:",inline"`
}

// APIVersionKind is an APIVersion and Kind tuple.
type APIVersionKind struct {
	// APIVersion - the API version of the resource to watch.
	APIVersion string `json:"apiVersion"`

	// Kind of the resource to watch.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
	Kind string `json:"kind"`
}

// APIVersionKindSelector is an APIVersion Kind tuple with a LabelSelector.
type APIVersionKindSelector struct {
	// APIVersion - the API version of the resource to watch.
	APIVersion string `json:"apiVersion"`

	// Kind of the resource to watch.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
	Kind string `json:"kind"`

	// LabelSelector filters this source to objects to those resources pass the
	// label selector.
	// More info: http://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	// +optional
	LabelSelector *metav1.LabelSelector `json:"selector,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ApiServerSourceList contains a list of ApiServerSource
type ApiServerSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApiServerSource `json:"items"`
}

// GetStatus retrieves the status of the ApiServerSource . Implements the KRShaped interface.
func (a *ApiServerSource) GetStatus() *duckv1.Status {
	return &a.Status.Status
}
