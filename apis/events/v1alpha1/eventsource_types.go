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
	"k8s.io/apimachinery/pkg/runtime/schema"

	openfunction "github.com/openfunction/apis/core/v1beta1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

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
	// Mqtt event source, the Key is used to refer to the name of the event
	// +optional
	Mqtt map[string]*MQTTSpec `json:"mqtt,omitempty"`
	// Sink is a callable address, such as Knative Service
	// +optional
	Sink *SinkSpec `json:"sink,omitempty"`
	// Information needed to build a function. The build step will be skipped if Build is nil.
	Build *openfunction.BuildImpl `json:"build,omitempty"`
	// The logging level of the event source handler, e.g. "1", "2", "3".
	// The level increases as the value increases, default is "1".
	// +optional
	LogLevel *string `json:"logLevel,omitempty"`
}

// SinkSpec specifies the receiver of the events an EventSource received, sinks in the Uri format have higher priority than sinks in Reference format.
type SinkSpec struct {
	Ref *Reference `json:"ref,omitempty"`
	Uri *string    `json:"uri,omitempty"`
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

func (ref Reference) GroupVersionKind() schema.GroupVersionKind {
	return schema.FromAPIVersionAndKind(ref.APIVersion, ref.Kind)
}

// EventSourceStatus defines the observed state of EventSource
type EventSourceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Conditions []Condition `json:"conditions,omitempty" description:"List of auditable conditions of EventSource"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName=es
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="EventBus",type=string,JSONPath=`.spec.eventBus`
//+kubebuilder:printcolumn:name="Sink",type=string,JSONPath=`.spec.sink.uri`
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[-1].type"

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
