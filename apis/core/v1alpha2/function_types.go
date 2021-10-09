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

package v1alpha2

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GitRepo struct {
	// Git url to clone
	Url string `json:"url"`
	// Git revision to check out (branch, tag, sha, ref…) (default:"")
	Revision *string `json:"revision,omitempty"`
	// A subpath within the `source` input where the source to build is located.
	SourceSubPath *string `json:"sourceSubPath,omitempty"`
	// Credentials references a Secret that contains credentials to access
	// the repository.
	//
	// +optional
	Credentials *v1.LocalObjectReference `json:"credentials,omitempty"`
}

func (gr *GitRepo) Init() {
	var revision, sourceSubPath string
	gr.Revision = &revision
	gr.SourceSubPath = &sourceSubPath
	gr.Credentials = &v1.LocalObjectReference{}
}

type Language string
type Runtime string

const (
	Go            Language = "go"
	Node          Language = "node"
	BuildPhase             = "Build"
	ServingPhase           = "Serving"
	Created                = "Created"
	Building               = "Building"
	Running                = "Running"
	Succeeded              = "Succeeded"
	Failed                 = "Failed"
	Skipped                = "Skipped"
	Knative       Runtime  = "Knative"
	OpenFuncAsync Runtime  = "OpenFuncAsync"
	Shipwright             = "Shipwright"
)

type Strategy struct {
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string `json:"name"`
	// BuildStrategyKind indicates the kind of the build strategy BuildStrategy or ClusterBuildStrategy, default to BuildStrategy.
	Kind *string `json:"kind,omitempty"`
}

type ShipwrightEngine struct {
	// Strategy references the BuildStrategy to use to build the image.
	// +optional
	Strategy *Strategy `json:"strategy,omitempty"`
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
	Builder *string `json:"builder"`
	// BuilderCredentials references a Secret that contains credentials to access
	// the builder image repository.
	//
	// +optional
	BuilderCredentials *v1.LocalObjectReference `json:"builderCredentials,omitempty"`
	// The configuration for the `Shipwright` build engine.
	Shipwright *ShipwrightEngine `json:"shipwright,omitempty"`
	// Params is a list of key/value that could be used to set strategy parameters.
	// When using _params_, users should avoid:
	// Defining a parameter name that doesn't match one of the `spec.parameters` defined in the `BuildStrategy`.
	// Defining a parameter name that collides with the Shipwright reserved parameters including BUILDER_IMAGE,DOCKERFILE,CONTEXT_DIR and any name starting with shp-.
	Params map[string]string `json:"params,omitempty"`
	// Environment variables to pass to the builder.
	Env map[string]string `json:"env,omitempty"`
	// Function Source code repository
	SrcRepo *GitRepo `json:"srcRepo"`
	// Dockerfile is the path to the Dockerfile used by build strategies that rely on the Dockerfile to build an image.
	//
	// +optional
	Dockerfile *string `json:"dockerfile,omitempty"`
}

type ServingImpl struct {
	// Function runtime such as Knative or OpenFuncAsync.
	Runtime *Runtime `json:"runtime"`
	// Parameters to pass to the serving.
	// All parameters will be injected into the pod as environment variables.
	// Function code can use these parameters by getting environment variables
	Params map[string]string `json:"params,omitempty"`
	// Parameters of asyncFunc runtime, must not be nil when runtime is OpenFuncAsync.
	OpenFuncAsync *OpenFuncAsyncRuntime `json:"openFuncAsync,omitempty"`
	// Template describes the pods that will be created.
	// The container named `function` is the container which is used to run the image built by the builder.
	// If it is not set, the controller will automatically add one.
	// +optional
	Template *v1.PodSpec `json:"template,omitempty"`
}

// FunctionSpec defines the desired state of Function
type FunctionSpec struct {
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
	// Information needed to build a function. The build step will be skipped if Build is nil.
	Build *BuildImpl `json:"build,omitempty"`
	// Information needed to run a function. The serving step will be skipped if `Serving` is nil.
	Serving *ServingImpl `json:"serving,omitempty"`
}

type Condition struct {
	State        string `json:"state,omitempty"`
	ResourceRef  string `json:"resourceRef,omitempty"`
	ResourceHash string `json:"resourceHash,omitempty"`
}

// FunctionStatus defines the observed state of Function
type FunctionStatus struct {
	Build   *Condition `json:"build,omitempty"`
	Serving *Condition `json:"serving,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:storageversion
//+kubebuilder:resource:shortName=fn
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="BuildState",type=string,JSONPath=`.status.build.state`
//+kubebuilder:printcolumn:name="ServingState",type=string,JSONPath=`.status.serving.state`
//+kubebuilder:printcolumn:name="Builder",type=string,JSONPath=`.status.build.resourceRef`
//+kubebuilder:printcolumn:name="Serving",type=string,JSONPath=`.status.serving.resourceRef`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Function is the Schema for the functions API
type Function struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FunctionSpec   `json:"spec,omitempty"`
	Status FunctionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FunctionList contains a list of Function
type FunctionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Function `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Function{}, &FunctionList{})
}
