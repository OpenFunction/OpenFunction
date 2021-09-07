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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EventSourceCreationStatus describes the creation status
// of the scaler's additional resources such as Services, Ingresses and Deployments
// +kubebuilder:validation:Enum=Created;Error;Pending;Unknown;Terminating;Terminated;Ready
type EventSourceCreationStatus string

// EventSourceConditionReason describes the reason why the condition transitioned
// +kubebuilder:validation:Enum=ErrorConfiguration;ErrorToFindExistEventBus;ErrorGenerateComponent;ErrorGenerateScaledObject;ErrorCreatingEventSourceWorkload;ErrorCreatingEventSource;EventSourceWorkloadCreated;PendingCreation;EventSourceIsReady
type EventSourceConditionReason string

// EventSourceSpec defines the desired state of EventSource
type EventSourceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// EventBus allows you to specify a specific EventBus to be used instead of the "default" one
	// +optional
	EventBus string `json:"eventBus,omitempty"`
	// Redis event source, the Key is used to refer to the name of the event
	// +optional
	Redis map[string]*RedisSpec `json:"redis,omitempty"`
	// Kafka event source, the Key is used to refer to the name of the event
	// +optional
	Kafka map[string]*KafkaSpec `json:"kafka,omitempty"`
	// Cron event source, the Key is used to refer to the name of the event
	// +optional
	Cron map[string]*CronSpec `json:"cron,omitempty"`
	// Sink is a callable address, such as Knative Service
	// +optional
	Sink *SinkSpec `json:"sink,omitempty"`
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
	Conditions []EventSourceCondition `json:"conditions,omitempty" description:"List of auditable conditions of the operator"`
}

type EventSourceCondition struct {
	// Timestamp of the condition
	// +optional
	Timestamp string `json:"timestamp" description:"Timestamp of this condition"`
	// Type of condition
	// +required
	Type EventSourceCreationStatus `json:"type" description:"type of status condition"`
	// Status of the condition, one of True, False, Unknown.
	// +required
	Status metav1.ConditionStatus `json:"status" description:"status of the condition, one of True, False, Unknown"`
	// The reason for the condition's last transition.
	// +optional
	Reason EventSourceConditionReason `json:"reason,omitempty" description:"one-word CamelCase reason for the condition's last transition"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty" description:"human-readable message indicating details about last transition"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName=es
//+kubebuilder:subresource:status

// EventSource is the Schema for the eventsources API
//+kubebuilder:printcolumn:name="EventBus",type=string,JSONPath=`.spec.eventBus`
//+kubebuilder:printcolumn:name="Sink",type=string,JSONPath=`.spec.sink.ref.name`
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[-1].type"
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
