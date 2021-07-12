/*
Copyright 2021.

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
	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EventSourceSpec defines the desired state of EventSource
type EventSourceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// EventBusName allows you to specify a specific EventBus to be used instead of the "default" one
	// +optional
	EventBusNames []string `json:"eventBusNames,omitempty"`
	// Redis event source, the Key is used to refer to the name of the event
	// +optional
	Redis map[string]componentsv1alpha1.ComponentSpec `json:"redis,omitempty"`
	// Kafka event source, the Key is used to refer to the name of the event
	// +optional
	Kafka map[string]componentsv1alpha1.ComponentSpec `json:"kafka,omitempty"`
	// Topic Specify the topic name of the EventBus
	Topic string `json:"topic,omitempty"`
	// Sink is a callable address, such as Knative Service
	// +optional
	Sink *SinkSpec `json:"sink,omitempty"`
}

// SourceSpec describes specs for component of event source.
type SourceSpec struct {
	// Metadata holds the user defined metadata which will passed along the event payload.
	Metadata []componentsv1alpha1.MetadataItem `json:"metadata"`
	// +optional
	IgnoreErrors bool `json:"ignoreErrors"`
	// +optional
	InitTimeout string `json:"initTimeout"`
}

// SinkSpec describes an event source for the Kafka.
type SinkSpec struct {
	Ref *Reference `json:"ref,omitempty"`
}

type Reference struct {
	// Kind of the referent.
	Kind string `json:"kind"`
	// Namespace of the referent.
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// Name of the referent.
	Name string `json:"name"`
	// API version of the referent.
	APIVersion string `json:"apiVersion"`
}

// EventSourceStatus defines the observed state of EventSource
type EventSourceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName=es
//+kubebuilder:subresource:status

// EventSource is the Schema for the eventsources API
type EventSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventSourceSpec   `json:"spec,omitempty"`
	Status EventSourceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EventSourceList contains a list of EventSource
type EventSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EventSource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EventSource{}, &EventSourceList{})
}
