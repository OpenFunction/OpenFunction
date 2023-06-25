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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	shipwrightv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BuilderState defines builder's states that a user can set to overwrite a builder's current state
type BuilderState string

const (
	// BuilderStateCancelled indicates a user's intent to stop the build process if not
	// already canceled or terminated
	BuilderStateCancelled = "Cancelled"
)

type Strategy struct {
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string `json:"name"`
	// BuildStrategyKind indicates the kind of the build strategy BuildStrategy or ClusterBuildStrategy, default to BuildStrategy.
	Kind *string `json:"kind,omitempty"`
}

// SingleValue is the value type contains the properties for a value, this allows for an
// easy extension in the future to support more kinds
type SingleValue struct {

	// The value of the parameter
	// +optional
	Value *string `json:"value"`

	// The ConfigMap value of the parameter
	// +optional
	ConfigMapValue *shipwrightv1alpha1.ObjectKeyRef `json:"configMapValue,omitempty"`

	// The secret value of the parameter
	// +optional
	SecretValue *shipwrightv1alpha1.ObjectKeyRef `json:"secretValue,omitempty"`
}

// ParamValue is a key/value that populates a strategy parameter
// used in the execution of the strategy steps
type ParamValue struct {

	// Inline the properties of a value
	// +optional
	*SingleValue `json:",inline"`

	// Name of the parameter
	// +required
	Name string `json:"name"`

	// Values of an array parameter
	// +optional
	Values []SingleValue `json:"values,omitempty"`
}

type ShipwrightEngine struct {
	// Strategy references the BuildStrategy to use to build the image.
	// +optional
	Strategy *Strategy `json:"strategy,omitempty"`
	// Params is a list of key/value that could be used to set strategy parameters.
	// When using _params_, users should avoid:
	// Defining a parameter name that doesn't match one of the `spec.parameters` defined in the `BuildStrategy`.
	// Defining a parameter name that collides with the Shipwright reserved parameters including BUILDER_IMAGE,DOCKERFILE,CONTEXT_DIR and any name starting with shp-.
	Params []*ParamValue `json:"params,omitempty"`
	// Timeout defines the maximum amount of time the Build should take to execute.
	//
	// +optional
	// +kubebuilder:validation:Format=duration
	Timeout *metav1.Duration `json:"timeout,omitempty"`
}

type BuildImpl struct {
	// Builder refers to the image containing the build tools to build the source code.
	//
	// +optional
	Builder *string `json:"builder,omitempty"`
	// BuilderCredentials references a Secret that contains credentials to access
	// the builder image repository.
	//
	// +optional
	BuilderCredentials *v1.LocalObjectReference `json:"builderCredentials,omitempty"`
	// The configuration for the `Shipwright` build engine.
	Shipwright *ShipwrightEngine `json:"shipwright,omitempty"`

	// Environment variables to pass to the builder.
	Env map[string]string `json:"env,omitempty"`
	// Function Source code repository
	SrcRepo *GitRepo `json:"srcRepo"`
	// Dockerfile is the path to the Dockerfile used by build strategies that rely on the Dockerfile to build an image.
	//
	// +optional
	Dockerfile *string `json:"dockerfile,omitempty"`
	// Timeout defines the maximum amount of time the Build should take to execute.
	//
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// The number of successful builds to retain, default is 0.
	// +optional
	SuccessfulBuildsHistoryLimit *int32 `json:"successfulBuildsHistoryLimit,omitempty"`

	// The number of failed builds to retain, default is 1.
	// +optional
	FailedBuildsHistoryLimit *int32 `json:"failedBuildsHistoryLimit,omitempty"`
	// The duration to retain a completed builder, defaults to 0 (forever).
	// +optional
	BuilderMaxAge *metav1.Duration `json:"builderMaxAge,omitempty"`
}

// BuilderSpec defines the desired state of Builder
type BuilderSpec struct {
	// Function image name
	Image string `json:"image"`
	// ImageCredentials references a Secret that contains credentials to access
	// the image repository.
	//
	// +optional
	ImageCredentials *v1.LocalObjectReference `json:"imageCredentials,omitempty"`
	// State is used for canceling a buildrun (and maybe more later on).
	// +optional
	State     BuilderState `json:"state,omitempty"`
	BuildImpl `json:",inline"`
}

// BuilderOutput holds the results from the output step (build-and-push)
type BuilderOutput struct {
	// Digest holds the digest of output image
	Digest string `json:"digest,omitempty"`

	// Size holds the compressed size of output image
	Size int64 `json:"size,omitempty"`
}

// BuilderStatus defines the observed state of Builder
type BuilderStatus struct {
	Phase         string           `json:"phase,omitempty"`
	State         string           `json:"state,omitempty"`
	Reason        string           `json:"reason,omitempty"`
	Message       string           `json:"message,omitempty"`
	BuildDuration *metav1.Duration `json:"buildDuration,omitempty"`
	// Associate resources.
	ResourceRef map[string]string `json:"resourceRef,omitempty"`
	// Output holds the results emitted from step definition of an output
	//
	// +optional
	Output *BuilderOutput `json:"output,omitempty"`
	// Sources holds the results emitted from the step definition
	// of different sources
	//
	// +optional
	Sources []SourceResult `json:"sources,omitempty"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:storageversion
//+kubebuilder:resource:shortName=fb
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
//+kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.reason`
//+kubebuilder:printcolumn:name="BuildDuration",type=string,JSONPath=`.status.buildDuration`
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Builder is the Schema for the builders API
type Builder struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuilderSpec   `json:"spec,omitempty"`
	Status BuilderStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BuilderList contains a list of Builder
type BuilderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Builder `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Builder{}, &BuilderList{})
}

func (s *BuilderStatus) IsCompleted() bool {
	return s.State != "" && s.State != Building
}

func (s *BuilderStatus) IsSucceeded() bool {
	return s.State == Succeeded
}
