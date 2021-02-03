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

package v1alpha1

import (
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

// ConfigMapPropagation is used to propagate configMaps from original namespace to current namespace
type ConfigMapPropagation struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the ConfigMapPropagation
	Spec ConfigMapPropagationSpec `json:"spec,omitempty"`

	// Status represents the current state of the EventType.
	// This data may be out of date.
	// +optional
	Status ConfigMapPropagationStatus `json:"status,omitempty"`
}

var (
	// Check that ConfigMapPropagation can be validated, can be defaulted, and has immutable fields.
	_ apis.Validatable = (*ConfigMapPropagation)(nil)
	_ apis.Defaultable = (*ConfigMapPropagation)(nil)

	// Check that ConfigMapPropagation can return its spec untyped.
	_ apis.HasSpec = (*ConfigMapPropagation)(nil)

	_ runtime.Object = (*ConfigMapPropagation)(nil)

	// Check that we can create OwnerReferences to a ConfigMapPropagation.
	_ kmeta.OwnerRefable = (*ConfigMapPropagation)(nil)

	// Check that the type conforms to the duck Knative Resource shape.
	_ duckv1.KRShaped = (*ConfigMapPropagation)(nil)
)

type ConfigMapPropagationSpec struct {
	// OriginalNamespace is the namespace where the original configMaps are in
	OriginalNamespace string `json:"originalNamespace,omitempty"`
	// Selector only selects original configMaps with corresponding labels
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

// ConfigMapPropagationStatus represents the current state of a ConfigMapPropagation.
type ConfigMapPropagationStatus struct {
	// inherits duck/v1 Status, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last processed by the controller.
	// * Conditions - the latest available observations of a resource's current state.
	duckv1.Status `json:",inline"`

	//CopyConfigMaps is the status for each copied configmap.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	CopyConfigMaps []ConfigMapPropagationStatusCopyConfigMap `json:"copyConfigmaps" patchStrategy:"merge" patchMergeKey:"name"`
}

// ConfigMapPropagationStatusCopyConfigMap represents the status of a copied configmap
type ConfigMapPropagationStatusCopyConfigMap struct {
	// Name is copy configmap's name
	// +required
	Name string `json:"name,omitempty"`

	// Source is "originalNamespace/originalConfigMapName"
	Source string `json:"source,omitempty"`

	// Operation represents the operation CMP takes for this configmap. The operations are copy|delete|stop
	Operation string `json:"operation,omitempty"`

	// Ready represents the operation is ready or not
	Ready string `json:"ready,omitempty"`

	// Reason indicates reasons if the operation is not ready
	Reason string `json:"reason,omitempty"`

	// ResourceVersion is the resourceVersion of original configmap
	ResourceVersion string `json:"resourceVersionFromSource,omitempty" protobuf:"bytes,6,opt,name=resourceVersion"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConfigMapPropagationList is a collection of ConfigMapPropagation.
type ConfigMapPropagationList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConfigMapPropagation `json:"items"`
}

// GetGroupVersionKind returns GroupVersionKind for ConfigMapPropagation
func (cmp *ConfigMapPropagation) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("ConfigMapPropagation")
}

// GetUntypedSpec returns the spec of the ConfigMapPropagation.
func (cmp *ConfigMapPropagation) GetUntypedSpec() interface{} {
	return cmp.Spec
}

// GetStatus retrieves the status of the ConfigMapPropagation. Implements the KRShaped interface.
func (cmp *ConfigMapPropagation) GetStatus() *duckv1.Status {
	return &cmp.Status.Status
}
