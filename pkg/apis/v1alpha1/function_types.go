/*


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
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GitRepo struct {
	// Git url to clone
	Url string `json:"url"`
	// Git revision to checkout (branch, tag, sha, ref…) (default:"")
	Revision *string `json:"revision,omitempty"`
	// Git refspec to fetch before checking out revision (default:"")
	Refspec *string `json:"refspec,omitempty"`
	// Defines if the resource should initialize and fetch the submodules (default: true)
	Submodules *bool `json:"submodules,omitempty"`
	// Performs a shallow clone where only the most recent commit(s) will be fetched (default: 1)
	Depth *int8 `json:"depth,omitempty"`
	// Defines if http.sslVerify should be set to true or false in the global git config (default: true)
	SslVerify *bool `json:"sslVerify,omitempty"`
	// Subdirectory inside the "output" workspace to clone the git repo into (default: "")
	SubDirectory *string `json:"subDirectory,omitempty"`
	// A subpath within the `source` input where the source to build is located.
	SourceSubPath *string `json:"sourceSubPath,omitempty"`
	// Clean out the contents of the repo's destination directory if it already exists before cloning the repo there (default: true)
	DeleteExisting *string `json:"deleteExisting,omitempty"`
	// Git HTTP proxy server for non-SSL requests (default: "")
	HttpProxy *string `json:"httpProxy,omitempty"`
	// Git HTTPS proxy server for SSL requests (default: "")
	HttpsProxy *string `json:"httpsProxy,omitempty"`
	// Git no proxy - opt out of proxying HTTP/HTTPS requests (default: "")
	NoProxy *string `json:"noProxy,omitempty"`
	// Log the commands that are executed during git-clone's operation (default: true)
	Verbose *bool `json:"verbose,omitempty"`
	// The image used where the git-init binary is (default: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init:v0.17.3")
	GitInitImage *string `json:"gitInitImage,omitempty"`
}

func (gr *GitRepo) Init() {
	var revision, refspec, subDir, sourceSubPath, deletingExisting, httpProxy, httpsProxy, noProxy, gitInitImage string
	var submodules, sslVerify, verbose bool
	var depth int8
	gr.Revision = &revision
	gr.Refspec = &refspec
	gr.SubDirectory = &subDir
	gr.SourceSubPath = &sourceSubPath
	gr.DeleteExisting = &deletingExisting
	gr.HttpProxy = &httpProxy
	gr.HttpsProxy = &httpsProxy
	gr.NoProxy = &noProxy
	gr.GitInitImage = &gitInitImage
	gr.Submodules = &submodules
	gr.SslVerify = &sslVerify
	gr.Verbose = &verbose
	gr.Depth = &depth
}

type Registry struct {
	// Image registry url
	Url *string `json:"url,omitempty"`
	// Image registry account including username and password
	Account *v1.SecretKeySelector `json:"account,omitempty"`
}

func (r *Registry) Init() {
	var url string
	r.Url = &url
	r.Account = &v1.SecretKeySelector{}
}

type Language string
type Runtime string

const (
	Go           Language = "go"
	Python       Language = "python"
	Node         Language = "node"
	BuildPhase            = "Build"
	ServingPhase          = "Serving"
	Created               = "Created"
	Launching             = "Launching"
	Launched              = "Launched"
	Failed                = "Failed"
	Knative      Runtime  = "Knative"
	KEDA         Runtime  = "KEDA"
)

type BuildImpl struct {
	// Cloud Native Buildpacks builders
	Builder string `json:"builder"`
	// Environment params to pass to the builder
	Params map[string]string `json:"params,omitempty"`
	// Function Source code repository
	SrcRepo *GitRepo `json:"srcRepo"`
	// Image registry of the function image
	Registry *Registry `json:"registry"`
}

type ServingImpl struct {
	// Function runtime such as Knative or KEDA
	Runtime *Runtime `json:"runtime,omitempty"`
	// Parameters to pass to the serving.
	// All parameters will be injected into the pod as environment variables.
	// Function code can use these parameters by getting environment variables
	Params map[string]string `json:"params,omitempty"`
}

// FunctionSpec defines the desired state of Function
type FunctionSpec struct {
	// Function version in format like v1.0.0
	Version *string `json:"version,omitempty"`
	// Function image name
	Image string `json:"image"`
	// The port on which the function will be invoked
	Port *int32 `json:"port,omitempty"`
	// Information needed to build a function
	Build *BuildImpl `json:"build"`
	// Information needed to run a function
	Serving *ServingImpl `json:"serving,omitempty"`
}

// FunctionStatus defines the observed state of Function
type FunctionStatus struct {
	Phase string `json:"phase,omitempty"`
	State string `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=fn
// +kubebuilder:subresource:status

// Function is the Schema for the functions API
type Function struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FunctionSpec   `json:"spec,omitempty"`
	Status FunctionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FunctionList contains a list of Function
type FunctionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Function `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Function{}, &FunctionList{})
}
