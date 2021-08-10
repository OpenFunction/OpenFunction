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
	subscriptionsv1alpha1 "github.com/dapr/dapr/pkg/apis/subscriptions/v1alpha1"
	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KedaScaledObject struct {
	// How to run the function, known values are Deployment or StatefulSet, default is Deployment.
	WorkloadType string `json:"workloadType,omitempty"`
	// +optional
	PollingInterval *int32 `json:"pollingInterval,omitempty"`
	// +optional
	CooldownPeriod *int32 `json:"cooldownPeriod,omitempty"`
	// +optional
	MinReplicaCount *int32 `json:"minReplicaCount,omitempty"`
	// +optional
	MaxReplicaCount *int32 `json:"maxReplicaCount,omitempty"`
	// +optional
	Advanced *kedav1alpha1.AdvancedConfig `json:"advanced,omitempty"`

	Triggers []kedav1alpha1.ScaleTriggers `json:"triggers"`
}

type KedaScaledJob struct {
	// Restart policy for all containers within the pod.
	// One of OnFailure, Never.
	// Default to Never.
	// +optional
	RestartPolicy *v1.RestartPolicy `json:"restartPolicy,omitempty"`
	// +optional
	PollingInterval *int32 `json:"pollingInterval,omitempty"`
	// +optional
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty"`
	// +optional
	FailedJobsHistoryLimit *int32 `json:"failedJobsHistoryLimit,omitempty"`
	// +optional
	MaxReplicaCount *int32 `json:"maxReplicaCount,omitempty"`
	// +optional
	ScalingStrategy kedav1alpha1.ScalingStrategy `json:"scalingStrategy,omitempty"`

	Triggers []kedav1alpha1.ScaleTriggers `json:"triggers"`
}

type DaprIO struct {
	// The name of Dapr component, subscriptions, service invocation.
	// Component and subscription must be created in k8s cluster, or defined in the `components` or `subscriptions`.
	Name string `json:"name"`
	// Input type, known values are bindings, pubsub, invoke.
	// bindings: Indicates that the input is the Dapr bindings component.
	// pubsub: Indicates that the input is the Dapr pubsub component or subscription.
	// invoke: Indicates that the input is the Dapr service invocation.
	Type string `json:"type"`
	// Topic name of mq, required when type is pubsub
	Topic string `json:"topic,omitempty"`
	// Method name of dapr service invocation, required when type is invoke.
	MethodName string `json:"methodName,omitempty"`
	// Parameters for dapr input/output.
	Params map[string]string `json:"params,omitempty"`
}

type DaprComponent struct {
	Name                             string `json:"name"`
	componentsv1alpha1.ComponentSpec `json:",inline"`
}

type DaprSubscription struct {
	Name                                   string `json:"name"`
	subscriptionsv1alpha1.SubscriptionSpec `json:",inline"`
	// +optional
	Scopes []string `json:"scopes,omitempty"`
}

type Dapr struct {
	// Annotations for dapr
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// Components of dapr.
	// +optional
	Components []DaprComponent `json:"components,omitempty"`
	// Subscriptions of dapr.
	// +optional
	Subscriptions []DaprSubscription `json:"subscriptions,omitempty"`
	// Function inputs from Dapr components including binding, pubsub, and service invocation
	// +optional
	Inputs []*DaprIO `json:"inputs,omitempty"`
	// Function outputs from Dapr components including binding, pubsub, and service invocation
	// +optional
	Outputs []*DaprIO `json:"outputs,omitempty"`
}

type Keda struct {
	// +optional
	ScaledObject *KedaScaledObject `json:"scaledObject,omitempty"`
	// +optional
	ScaledJob *KedaScaledJob `json:"scaledJob,omitempty"`
}

type OpenFuncAsyncRuntime struct {
	// Configurations of dapr.
	// +optional
	Dapr *Dapr `json:"dapr,omitempty"`
	// Configurations of keda.
	// +optional
	Keda *Keda `json:"keda,omitempty"`
}

// ServingSpec defines the desired state of Serving
type ServingSpec struct {
	// Function version in format like v1.0.0
	Version *string `json:"version,omitempty"`
	// Function image name
	Image string `json:"image"`
	// ImageCredentials references a Secret that contains credentials to access
	// the image repository.
	//
	// +optional
	ImageCredentials *v1.LocalObjectReference `json:"imageCredentials,omitempty"`
	// The port on which the function will be invoked
	Port *int32 `json:"port,omitempty"`
	// The backend runtime to run a function, for example Knative
	Runtime *Runtime `json:"runtime"`
	// Parameters to pass to the serving.
	// All parameters will be injected into the pod as environment variables.
	// Function code can use these parameters by getting environment variables
	Params map[string]string `json:"params,omitempty"`
	// Parameters of OpenFuncAsync runtime.
	// +optional
	OpenFuncAsync *OpenFuncAsyncRuntime `json:"openFuncAsync,omitempty"`
	// Template describes the pods that will be created.
	// The container named `function` is the container which is used to run the image built by the builder.
	// If it is not set, the controller will automatically add one.
	// +optional
	Template *v1.PodSpec `json:"template,omitempty"`
}

// ServingStatus defines the observed state of Serving
type ServingStatus struct {
	Phase string `json:"phase,omitempty"`
	State string `json:"state,omitempty"`
	// Associate resources.
	ResourceRef map[string]string `json:"resourceRef,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName=fs
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Serving is the Schema for the servings API
type Serving struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServingSpec   `json:"spec,omitempty"`
	Status ServingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ServingList contains a list of Serving
type ServingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Serving `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Serving{}, &ServingList{})
}
