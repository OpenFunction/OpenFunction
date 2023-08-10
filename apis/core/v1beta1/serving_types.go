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

package v1beta1

import (
	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	Knative Runtime = "knative"
	Async   Runtime = "async"

	DaprBindings = "bindings"
	DaprPubsub   = "pubsub"

	WorkloadTypeJob         = "Job"
	WorkloadTypeStatefulSet = "StatefulSet"
	WorkloadTypeDeployment  = "Deployment"
)

// Runtime describes the type of the backend runtime.
// +kubebuilder:validation:Enum=knative;async
type Runtime string

// ScaleTargetKind represents the kind of trigger target.
// +kubebuilder:validation:Enum=object;job
type ScaleTargetKind string

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
}

type KedaScaledJob struct {
	// Restart policy for all containers within the pod.
	// One of 'OnFailure', 'Never'.
	// Default to 'Never'.
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
}

type Triggers struct {
	kedav1alpha1.ScaleTriggers `json:",inline"`
	// +optional
	TargetKind *ScaleTargetKind `json:"targetKind,omitempty"`
}

type DaprIO struct {
	// The name of DaprIO.
	Name string `json:"name"`
	// Component indicates the name of components in Dapr
	Component string `json:"component"`
	// Topic name of mq, required when type is pubsub
	// +optional
	Topic string `json:"topic,omitempty"`
	// Parameters for dapr input/output.
	// +optional
	Params map[string]string `json:"params,omitempty"`
	// Operation field tells the Dapr component which operation it should perform.
	// +optional
	Operation string `json:"operation,omitempty"`
}

type ScaleOptions struct {
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`
	MinReplicas *int32 `json:"minReplicas,omitempty"`
	// +optional
	Keda *KedaScaleOptions `json:"keda,omitempty"`
	// Refer to https://knative.dev/docs/serving/autoscaling/ to
	// learn more about the autoscaling options of Knative Serving.
	// +optional
	Knative *map[string]string `json:"knative,omitempty"`
}

type KedaScaleOptions struct {
	// +optional
	ScaledObject *KedaScaledObject `json:"scaledObject,omitempty"`
	// +optional
	ScaledJob *KedaScaledJob `json:"scaledJob,omitempty"`
}

// ServingSpec defines the desired state of Serving
type ServingSpec struct {
	// Function version in format like v1.0.0
	Version *string `json:"version,omitempty"`
	// Function image name
	Image string `json:"image"`
	// ImageCredentials references a Secret that contains credentials to access
	// the image repository.
	// +optional
	ImageCredentials *v1.LocalObjectReference `json:"imageCredentials,omitempty"`
	// The port on which the function will be invoked
	Port *int32 `json:"port,omitempty"`
	// The configuration of the backend runtime for running function.
	Runtime Runtime `json:"runtime"`
	// Function inputs from Dapr components including binding, pubsub
	// Available for Async Runtime only.
	// +optional
	Inputs []*DaprIO `json:"inputs,omitempty"`
	// Function outputs from Dapr components including binding, pubsub
	// +optional
	Outputs []*DaprIO `json:"outputs,omitempty"`
	// The ScaleOptions will help us to set up guidelines for the autoscaling of function workloads.
	// +optional
	ScaleOptions *ScaleOptions `json:"scaleOptions,omitempty"`
	// Configurations of dapr bindings components.
	// +optional
	Bindings map[string]*componentsv1alpha1.ComponentSpec `json:"bindings,omitempty"`
	// Configurations of dapr pubsub components.
	// +optional
	Pubsub map[string]*componentsv1alpha1.ComponentSpec `json:"pubsub,omitempty"`
	// Configurations of dapr state components.
	// +optional
	States map[string]*componentsv1alpha1.ComponentSpec `json:"states,omitempty"`
	// Triggers are used to specify the trigger sources of the function.
	// The Keda (ScaledObject, ScaledJob) configuration in ScaleOptions cannot take effect without Triggers being set.
	// +optional
	Triggers []Triggers `json:"triggers,omitempty"`
	// Parameters to pass to the serving.
	// All parameters will be injected into the pod as environment variables.
	// Function code can use these parameters by getting environment variables
	Params map[string]string `json:"params,omitempty"`
	// Parameters of OpenFuncAsync runtime.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations that will be add to the workload.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// Template describes the pods that will be created.
	// The container named `function` is the container which is used to run the image built by the builder.
	// If it is not set, the controller will automatically add one.
	// +optional
	Template *v1.PodSpec `json:"template,omitempty"`
	// Timeout defines the maximum amount of time the Serving should take to execute before the Serving is running.
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`
}

// ServingStatus defines the observed state of Serving
type ServingStatus struct {
	Phase string `json:"phase,omitempty"`
	State string `json:"state,omitempty"`
	// Associate resources.
	ResourceRef map[string]string `json:"resourceRef,omitempty"`
	// Service holds the service name used to access the serving.
	// +optional
	Service string `json:"url,omitempty"`
}

//+genclient
//+kubebuilder:object:root=true
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

func (s *ServingStatus) IsStarting() bool {
	return s.State == "" || s.State == Starting
}
