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

// EventBusSpec defines the desired state of EventBus and ClusterEventBus
type EventBusSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Topic indicates the name of the message channel of eventbus
	// If not specified, "default" will be used as the name of the message channel
	// +optional
	Topic string `json:"topic,omitempty"`
	// Use Nats streaming as the default backend for event bus
	Nats *componentsv1alpha1.ComponentSpec `json:"nats,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName=eb

// EventBus is the Schema for the eventbus API
type EventBus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec EventBusSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// EventBusList contains a list of EventBus
type EventBusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EventBus `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EventBus{}, &EventBusList{})
}
