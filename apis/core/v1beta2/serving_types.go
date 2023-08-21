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

package v1beta2

import (
	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type Engine string

const (
	HookPolicyAppend   = "Append"
	HookPolicyOverride = "Override"

	WorkloadTypeJob                = "Job"
	WorkloadTypeStatefulSet        = "StatefulSet"
	WorkloadTypeDeployment         = "Deployment"
	HttpEngineKnative       Engine = "knative"
	HttpEngineKeda          Engine = "keda"
)

type Triggers struct {
	Http   *HttpTrigger   `json:"http,omitempty"`
	Dapr   []*DaprTrigger `json:"dapr,omitempty"`
	Inputs []*Input       `json:"inputs,omitempty"`
}

type HttpTrigger struct {
	// The port on which the function will be invoked
	Port *int32 `json:"port,omitempty"`
	// Information needed to make HTTPRoute.
	// Will attempt to make HTTPRoute using the default Gateway resource if Route is nil.
	//
	// +optional
	Route *RouteImpl `json:"route,omitempty"`
	// Http function runtime engine, can be set to knative or keda, default to knative if not set
	// +optional
	Engine *Engine `json:"engine,omitempty"`
}

type DaprTrigger struct {
	*DaprComponentRef `json:",inline"`
	// Deprecated: Only for compatibility with v1beta1
	InputName string `json:"inputName,omitempty"`
}

type DaprComponentRef struct {
	// The name of the dapr component, the component can be defined in
	// the `bindings`, `pubsub`, or `states`, or an existing component.
	Name string `json:"name"`
	// Type is the type of the component, if it is not set, controller will get it automatically.
	Type  string `json:"type,omitempty"`
	Topic string `json:"topic,omitempty"`
}

type DaprInput struct {
	*DaprComponentRef `json:",inline"`
}

type Input struct {
	Dapr *DaprInput `json:"dapr,omitempty"`
}

type DaprOutput struct {
	*DaprComponentRef `json:",inline"`
	// Metadata is the metadata for dapr Com.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`
	// Operation field tells the Dapr component which operation it should perform.
	// +optional
	Operation string `json:"operation,omitempty"`
	// Deprecated: Only for compatibility with v1beta1
	OutputName string `json:"outputName,omitempty"`
}

type Output struct {
	Dapr *DaprOutput `json:"dapr,omitempty"`
}

type State struct {
	Spec *componentsv1alpha1.ComponentSpec `json:"spec,omitempty"`
}

type HTTPScaledObject struct {
	// Target metric value
	// +optional
	TargetPendingRequests *int32 `json:"targetPendingRequests,omitempty"`
	// Cooldown period value
	// +optional
	CooldownPeriod *int32 `json:"cooldownPeriod,omitempty"`
}

type KedaScaledObject struct {
	// +optional
	PollingInterval *int32 `json:"pollingInterval,omitempty"`
	// +optional
	CooldownPeriod *int32 `json:"cooldownPeriod,omitempty"`
	// +optional
	Advanced *kedav1alpha1.AdvancedConfig `json:"advanced,omitempty"`
	// +optional
	Fallback *kedav1alpha1.Fallback `json:"fallback,omitempty"`
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
	ScalingStrategy kedav1alpha1.ScalingStrategy `json:"scalingStrategy,omitempty"`
}

type KedaScaleOptions struct {
	// +optional
	HTTPScaledObject *HTTPScaledObject `json:"httpScaledObject,omitempty"`
	// +optional
	ScaledObject *KedaScaledObject `json:"scaledObject,omitempty"`
	// +optional
	ScaledJob *KedaScaledJob `json:"scaledJob,omitempty"`
	// Triggers are used to specify the trigger sources of the function.
	// The Keda (ScaledObject, ScaledJob) configuration in ScaleOptions cannot take effect without Triggers being set.
	// +optional
	Triggers []kedav1alpha1.ScaleTriggers `json:"triggers,omitempty"`
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

type Hooks struct {
	Pre    []string `json:"pre,omitempty"`
	Post   []string `json:"post,omitempty"`
	Policy string   `json:"policy,omitempty"`
}

type TracingConfig struct {
	Enabled  bool              `json:"enabled" yaml:"enabled"`
	Provider *TracingProvider  `json:"provider" yaml:"provider"`
	Tags     map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Baggage  map[string]string `json:"baggage,omitempty" yaml:"baggage,omitempty"`
}

type TracingProvider struct {
	Name      string    `json:"name" yaml:"name"`
	OapServer string    `json:"oapServer,omitempty" yaml:"oapServer,omitempty"`
	Exporter  *Exporter `json:"exporter,omitempty" yaml:"exporter,omitempty"`
}

type Exporter struct {
	Name        string `json:"name" yaml:"name"`
	Endpoint    string `json:"endpoint" yaml:"endpoint"`
	Headers     string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Compression string `json:"compression,omitempty" yaml:"compression,omitempty"`
	Timeout     string `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Protocol    string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
}

type ServingImpl struct {
	// Triggers used to trigger the Function.
	// +optional
	Triggers *Triggers `json:"triggers,omitempty"`
	// The ScaleOptions will help us to set up guidelines for the autoscaling of function workloads.
	// +optional
	ScaleOptions *ScaleOptions `json:"scaleOptions,omitempty"`
	// Function outputs from Dapr components including binding, pubsub
	// +optional
	Outputs []*Output `json:"outputs,omitempty"`
	// Configurations of dapr bindings components.
	// +optional
	Bindings map[string]*componentsv1alpha1.ComponentSpec `json:"bindings,omitempty"`
	// Configurations of dapr pubsub components.
	// +optional
	Pubsub map[string]*componentsv1alpha1.ComponentSpec `json:"pubsub,omitempty"`
	// Configurations of dapr state components.
	// It can refer to an existing state when the `state.spec` is nil.
	// +optional
	States map[string]*State `json:"states,omitempty"`
	// Parameters to pass to the serving.
	// All parameters will be injected into the pod as environment variables.
	// Function code can use these parameters by getting environment variables
	Params map[string]string `json:"params,omitempty"`
	// Parameters of asyncFunc runtime, must not be nil when runtime is OpenFuncAsync.
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations that will be added to the workload.
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
	// Hooks define the hooks that will execute before or after function execution.
	// +optional
	Hooks *Hooks `json:"hooks,omitempty"`
	// Tracing is the config of tracing.
	// +optional
	Tracing *TracingConfig `json:"tracing,omitempty"`
	// How to run the function, known values are Deployment or StatefulSet, default is Deployment.
	WorkloadType string `json:"workloadType,omitempty"`
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
	ServingImpl      `json:",inline"`
}

// ServingStatus defines the observed state of Serving
type ServingStatus struct {
	Phase   string `json:"phase,omitempty"`
	State   string `json:"state,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
	// Associate resources.
	ResourceRef map[string]string `json:"resourceRef,omitempty"`
	// Service holds the service name used to access the serving.
	// +optional
	Service string `json:"url,omitempty"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:storageversion
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
