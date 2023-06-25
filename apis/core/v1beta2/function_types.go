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
	k8sgatewayapiv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BundleContainer describes the source code bundle container to pull
type BundleContainer struct {
	// Image reference, i.e. quay.io/org/image:tag
	Image string `json:"image"`
}

type GitRepo struct {
	// Git url to clone
	Url string `json:"url,omitempty"`
	// BundleContainer
	//
	// +optional
	BundleContainer *BundleContainer `json:"bundleContainer,omitempty"`
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
type AddressType string

const (
	BuildPhase     = "Build"
	ServingPhase   = "Serving"
	Created        = "Created"
	Building       = "Building"
	Starting       = "Starting"
	Running        = "Running"
	Succeeded      = "Succeeded"
	Failed         = "Failed"
	Skipped        = "Skipped"
	Timeout        = "Timeout"
	Canceled       = "Canceled"
	UnknownRuntime = "UnknownRuntime"
)

const InternalAddressType AddressType = "Internal"
const ExternalAddressType AddressType = "External"

type GatewayRef struct {
	// Name is the name of the referent.
	// It refers to the name of a Gateway resource.
	Name k8sgatewayapiv1alpha2.ObjectName `json:"name"`
	// Namespace is the namespace of the referent. When unspecified,
	// this refers to the local namespace of the Route.
	Namespace *k8sgatewayapiv1alpha2.Namespace `json:"namespace"`
}

// CommonRouteSpec defines the common attributes that all Routes MUST include
// within their spec.
type CommonRouteSpec struct {
	// GatewayRef references the Gateway resources that a Route wants
	// to be attached to.
	//
	// +optional
	GatewayRef *GatewayRef `json:"gatewayRef,omitempty"`
}

type RouteImpl struct {
	CommonRouteSpec `json:",inline"`
	// Hostnames defines a set of hostname that should match against the HTTP
	// Host header to select a HTTPRoute to process the request.
	//
	// +optional
	// +kubebuilder:validation:MaxItems=16
	Hostnames []k8sgatewayapiv1alpha2.Hostname `json:"hostnames,omitempty"`
	// Rules are a list of HTTP matchers, filters and actions.
	//
	// +optional
	// +kubebuilder:validation:MaxItems=16
	Rules []k8sgatewayapiv1alpha2.HTTPRouteRule `json:"rules,omitempty"`
}

// FunctionSpec defines the desired state of Function
type FunctionSpec struct {
	// WorkloadRuntime for Function. Know values:
	// ```
	// OCIContainer: Nodes will run standard OCI container workloads.
	// WasmEdge: Nodes will run workloads using the crun (with WasmEdge support).
	// ```
	// +optional
	// +kubebuilder:default="OCIContainer"
	WorkloadRuntime string `json:"workloadRuntime,omitempty"`
	// Function version in format like v1.0.0
	Version *string `json:"version,omitempty"`
	// Function image name
	Image string `json:"image"`
	// ImageCredentials references a Secret that contains credentials to access
	// the image repository.
	//
	// +optional
	ImageCredentials *v1.LocalObjectReference `json:"imageCredentials,omitempty"`
	// Information needed to build a function. The build step will be skipped if Build is nil.
	Build *BuildImpl `json:"build,omitempty"`
	// Information needed to run a function. The serving step will be skipped if `Serving` is nil.
	Serving *ServingImpl `json:"serving,omitempty"`
}

type Condition struct {
	State                     string           `json:"state,omitempty"`
	Reason                    string           `json:"reason,omitempty"`
	Message                   string           `json:"message,omitempty"`
	ResourceRef               string           `json:"resourceRef,omitempty"`
	LastSuccessfulResourceRef string           `json:"lastSuccessfulResourceRef,omitempty"`
	ResourceHash              string           `json:"resourceHash,omitempty"`
	Service                   string           `json:"service,omitempty"`
	BuildTime                 *metav1.Duration `json:"buildTime,omitempty"`
}

type FunctionAddress struct {
	// Type of the address.
	//
	Type *AddressType `json:"type"`
	// Value of the address.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Value string `json:"value"`
}

type RouteStatus struct {
	// Hosts list all actual hostnames of HTTPRoute.
	//
	// +optional
	// +kubebuilder:validation:MaxItems=16
	Hosts []k8sgatewayapiv1alpha2.Hostname `json:"hosts,omitempty"`
	// Paths list all actual paths of HTTPRoute.
	//
	// +optional
	// +kubebuilder:validation:MaxItems=16
	Paths []k8sgatewayapiv1alpha2.HTTPPathMatch `json:"paths,omitempty"`
	// Conditions describes the status of the route with respect to the Gateway.
	// Note that the route's availability is also subject to the Gateway's own
	// status conditions and listener status.
	//
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=8
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type Revision struct {
	ImageDigest string `json:"imageDigest,omitempty"`
}

// SourceResult holds the results emitted from the different sources
type SourceResult struct {
	// Name is the name of source
	Name string `json:"name"`

	// Git holds the results emitted from from the
	// step definition of a git source
	//
	// +optional
	Git *GitSourceResult `json:"git,omitempty"`

	// Bundle holds the results emitted from from the
	// step definition of bundle source
	//
	// +optional
	Bundle *BundleSourceResult `json:"bundle,omitempty"`
}

// BundleSourceResult holds the results emitted from the bundle source
type BundleSourceResult struct {
	// Digest hold the image digest result
	Digest string `json:"digest,omitempty"`
}

// GitSourceResult holds the results emitted from the git source
type GitSourceResult struct {
	// CommitSha holds the commit sha of git source
	CommitSha string `json:"commitSha,omitempty"`

	// CommitAuthor holds the commit author of a git source
	CommitAuthor string `json:"commitAuthor,omitempty"`

	// BranchName holds the default branch name of the git source
	// this will be set only when revision is not specified in Build object
	BranchName string `json:"branchName,omitempty"`
}

// FunctionStatus defines the observed state of Function
type FunctionStatus struct {
	Route   *RouteStatus `json:"route,omitempty"`
	Build   *Condition   `json:"build,omitempty"`
	Serving *Condition   `json:"serving,omitempty"`
	// Addresses holds the addresses that used to access the Function.
	// +optional
	Addresses []FunctionAddress `json:"addresses,omitempty"`
	Revision  *Revision         `json:"revision,omitempty"`
	// Sources holds the results emitted from the step definition
	// of different sources
	//
	// +optional
	Sources []SourceResult `json:"sources,omitempty"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:storageversion
//+kubebuilder:resource:shortName=fn
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="BuildState",type=string,JSONPath=`.status.build.state`
//+kubebuilder:printcolumn:name="ServingState",type=string,JSONPath=`.status.serving.state`
//+kubebuilder:printcolumn:name="Builder",type=string,JSONPath=`.status.build.resourceRef`
//+kubebuilder:printcolumn:name="Serving",type=string,JSONPath=`.status.serving.resourceRef`
//+kubebuilder:printcolumn:name="Address",type=string,JSONPath=`.status.addresses[?(@.type=="Internal")].value`
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
