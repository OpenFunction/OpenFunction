/*
 * Copyright 2020 The Knative Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1beta1

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

// EventType represents a type of event that can be consumed from a Broker.
type EventType struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the EventType.
	Spec EventTypeSpec `json:"spec,omitempty"`

	// Status represents the current state of the EventType.
	// This data may be out of date.
	// +optional
	// TODO might be removed https://github.com/knative/eventing/issues/2750
	Status EventTypeStatus `json:"status,omitempty"`
}

var (
	// Check that EventType can be validated, can be defaulted, and has immutable fields.
	_ apis.Validatable = (*EventType)(nil)
	_ apis.Defaultable = (*EventType)(nil)

	// Check that EventType can return its spec untyped.
	_ apis.HasSpec = (*EventType)(nil)

	_ runtime.Object = (*EventType)(nil)

	// Check that we can create OwnerReferences to an EventType.
	_ kmeta.OwnerRefable = (*EventType)(nil)

	// Check that the type conforms to the duck Knative Resource shape.
	_ duckv1.KRShaped = (*EventType)(nil)
)

type EventTypeSpec struct {
	// Type represents the CloudEvents type. It is authoritative.
	Type string `json:"type"`
	// Source is a URI, it represents the CloudEvents source.
	// +optional
	Source *apis.URL `json:"source,omitempty"`
	// Schema is a URI, it represents the CloudEvents schemaurl extension attribute.
	// It may be a JSON schema, a protobuf schema, etc. It is optional.
	// +optional
	Schema *apis.URL `json:"schema,omitempty"`
	// SchemaData allows the CloudEvents schema to be stored directly in the
	// EventType. Content is dependent on the encoding. Optional attribute.
	// The contents are not validated or manipulated by the system.
	// +optional
	SchemaData string `json:"schemaData,omitempty"`
	// TODO remove https://github.com/knative/eventing/issues/2750
	// Broker refers to the Broker that can provide the EventType.
	// +optional
	Broker string `json:"broker,omitempty"`
	// Description is an optional field used to describe the EventType, in any meaningful way.
	// +optional
	Description string `json:"description,omitempty"`
}

// EventTypeStatus represents the current state of a EventType.
type EventTypeStatus struct {
	// inherits duck/v1 Status, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last processed by the controller.
	// * Conditions - the latest available observations of a resource's current state.
	duckv1.Status `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EventTypeList is a collection of EventTypes.
type EventTypeList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EventType `json:"items"`
}

// GetGroupVersionKind returns GroupVersionKind for EventType
func (p *EventType) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("EventType")
}

// GetUntypedSpec returns the spec of the EventType.
func (e *EventType) GetUntypedSpec() interface{} {
	return e.Spec
}

// GetStatus retrieves the status of the EventType. Implements the KRShaped interface.
func (t *EventType) GetStatus() *duckv1.Status {
	return &t.Status.Status
}
