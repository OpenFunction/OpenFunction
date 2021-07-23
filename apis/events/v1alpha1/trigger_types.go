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

// TriggerSpec defines the desired state of Trigger
type TriggerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// EventBus allows you to specify a specific EventBus to be used instead of the "default" one
	EventBus string `json:"eventBus"`
	// Inputs defines the event sources associated with the Trigger
	Inputs []*Input `json:"inputs"`
	// Subscribers defines the subscribers associated with the Trigger
	Subscribers []*Subscriber `json:"subscribers"`
}

type Input struct {
	// Name, name of input
	Name string `json:"name"`
	// EventSourceNamespace, namespace of EventSource, default to namespace of Trigger
	EventSourceNamespace string `json:"eventSourceNamespace,omitempty"`
	// EventSourceName, name of EventSource
	EventSourceName string `json:"eventSourceName"`
	// EventName, name of event
	EventName string `json:"eventName"`
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
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

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
