/*
Copyright 2022 The OpenFunction Authors.

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

// TriggerSpec defines the desired state of Trigger
type TriggerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// EventBus allows you to specify a specific EventBus to be used instead of the "default" one
	EventBus string `json:"eventBus"`
	// Inputs defines the event sources associated with the Trigger
	Inputs map[string]*Input `json:"inputs"`
	// Subscribers defines the subscribers associated with the Trigger
	Subscribers []*Subscriber `json:"subscribers"`
	// The logging level of the event source handler, e.g. "1", "2", "3".
	// The level increases as the value increases, default is "1".
	// +optional
	LogLevel *string `json:"logLevel,omitempty"`
}

type Input struct {
	// Namespace, namespace of EventSource, default to namespace of Trigger
	Namespace string `json:"namespace,omitempty"`
	// EventSource, name of EventSource
	EventSource string `json:"eventSource"`
	// Event, name of event
	Event string `json:"event"`
}

type Subscriber struct {
	// Condition for judging events
	Condition string `json:"condition"`
	// Sink and DeadLetterSink are used to handle subscribers who use the synchronous call method
	Sink           *SinkSpec `json:"sink,omitempty"`
	DeadLetterSink *SinkSpec `json:"deadLetterSink,omitempty"`
	// Topic and DeadLetterTopic are used to handle subscribers who use the asynchronous call method
	Topic           string `json:"topic,omitempty"`
	DeadLetterTopic string `json:"deadLetterTopic,omitempty"`
}

// TriggerStatus defines the observed state of Trigger
type TriggerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Conditions []Condition `json:"conditions,omitempty" description:"List of auditable conditions of Trigger"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="EventBus",type=string,JSONPath=`.spec.eventBus`
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[-1].type"

// Trigger is the Schema for the triggers API
type Trigger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TriggerSpec   `json:"spec,omitempty"`
	Status TriggerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TriggerList contains a list of Trigger
type TriggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Trigger `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Trigger{}, &TriggerList{})
}
