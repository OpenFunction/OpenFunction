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
	k8sgatewayapiv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	DefaultHttpListenersCount    = 1
	DefaultHttpListenerName      = "ofn-http-internal"
	GatewayConfigAnnotation      = "networking.openfunction.io/last-applied-configuration"
	GatewayListenersAnnotation   = "networking.openfunction.io/injected-listeners"
	DefaultGatewayServiceName    = "gateway"
	DefaultK8sGatewayServiceName = "envoy"
)

const (
	GatewayReasonNotFound           k8sgatewayapiv1beta1.GatewayConditionReason = "NotFound"
	GatewayReasonCreationFailure    k8sgatewayapiv1beta1.GatewayConditionReason = "CreationFailure"
	GatewayReasonResourcesAvailable k8sgatewayapiv1beta1.GatewayConditionReason = "ResourcesAvailable"
)

type GatewayRef struct {
	// Name is the name of the referent.
	// It refers to the name of a k8s Gateway resource.
	//
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`
	// Namespace is the namespace of the referent.
	// It refers to a k8s namespace.
	//
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	Namespace string `json:"namespace"`
}

type GatewayDef struct {
	// Name is the name of the referent.
	// It refers to the name of a k8s Gateway resource.
	//
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name,omitempty"`
	// Namespace is the namespace of the referent.
	//
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	Namespace string `json:"namespace"`
	// GatewayClassName used for this Gateway.
	// This is the name of a GatewayClass resource.
	GatewayClassName k8sgatewayapiv1beta1.ObjectName `json:"gatewayClassName"`
}

type K8sGatewaySpec struct {
	// Listeners associated with this Gateway. Listeners define
	// logical endpoints that are bound on this Gateway's addresses.
	// At least one Listener MUST be specified.
	//
	// Each listener in a Gateway must have a unique combination of Hostname,
	// Port, and Protocol.
	//
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MinItems=1
	Listeners []k8sgatewayapiv1beta1.Listener `json:"listeners"`
}

// GatewaySpec defines the desired state of Gateway
type GatewaySpec struct {
	// Used to generate the hostname field of gatewaySpec.listeners.openfunction.hostname
	Domain string `json:"domain"`
	// Used to generate the hostname field of gatewaySpec.listeners.openfunction.hostname
	//
	// +optional
	// +kubebuilder:default="cluster.local"
	ClusterDomain string `json:"clusterDomain,omitempty"`
	// Used to generate the hostname of attaching HTTPRoute
	//
	// +optional
	// +kubebuilder:default="{{.Name}}.{{.Namespace}}.{{.Domain}}"
	HostTemplate string `json:"hostTemplate,omitempty"`
	// Used to generate the path of attaching HTTPRoute
	//
	// +optional
	// +kubebuilder:default="{{.Namespace}}/{{.Name}}"
	PathTemplate string `json:"pathTemplate,omitempty"`
	// Label key to add to the HTTPRoute generated by function
	// The value will be the `gateway.openfunction.openfunction.io` CR's namespaced name
	//
	// +optional
	// +kubebuilder:default="app.kubernetes.io/managed-by"
	HttpRouteLabelKey string `json:"httpRouteLabelKey,omitempty"`
	// Reference to an existing K8s gateway
	//
	// +optional
	GatewayRef *GatewayRef `json:"gatewayRef,omitempty"`
	// Definition to a new K8s gateway
	//
	// +optional
	GatewayDef *GatewayDef `json:"gatewayDef,omitempty"`
	// GatewaySpec defines the desired state of k8s Gateway.
	GatewaySpec K8sGatewaySpec `json:"gatewaySpec"`
}

type Condition struct {
	Type    string                 `json:"type" protobuf:"bytes,1,opt,name=type"`
	Status  metav1.ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status"`
	Reason  string                 `json:"reason" protobuf:"bytes,5,opt,name=reason"`
	Message string                 `json:"message" protobuf:"bytes,6,opt,name=message"`
}

type ListenerStatus struct {
	// Name is the name of the Listener that this status corresponds to.
	Name k8sgatewayapiv1beta1.SectionName `json:"name"`

	// SupportedKinds is the list indicating the Kinds supported by this
	// listener. This MUST represent the kinds an implementation supports for
	// that Listener configuration.
	//
	// If kinds are specified in Spec that are not supported, they MUST NOT
	// appear in this list and an implementation MUST set the "ResolvedRefs"
	// condition to "False" with the "InvalidRouteKinds" reason. If both valid
	// and invalid Route kinds are specified, the implementation MUST
	// reference the valid Route kinds that have been specified.
	//
	// +kubebuilder:validation:MaxItems=8
	SupportedKinds []k8sgatewayapiv1beta1.RouteGroupKind `json:"supportedKinds"`

	// AttachedRoutes represents the total number of Routes that have been
	// successfully attached to this Listener.
	AttachedRoutes int32 `json:"attachedRoutes"`

	// Conditions describe the current condition of this listener.
	//
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=8
	Conditions []Condition `json:"conditions"`
}

// GatewayStatus defines the observed state of Gateway
type GatewayStatus struct {
	// Addresses list the addresses that have actually been bound to the Gateway.
	// This is optional and behavior can depend on the k8s Gateway implementation.
	//
	// +optional
	// +kubebuilder:validation:MaxItems=16

	Addresses []k8sgatewayapiv1beta1.GatewayAddress `json:"addresses,omitempty"`
	// Conditions describe the current conditions of the Gateway.
	//
	// Known condition types are:
	//
	// * "Scheduled"
	// * "Ready"
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=8
	// +kubebuilder:default={{type: "Scheduled", status: "Unknown", reason:"NotReconciled", message:"Waiting for controller"}}
	Conditions []Condition `json:"conditions,omitempty"`

	// Listeners provide status for each unique listener port defined in the Spec.
	//
	// +optional
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=64
	Listeners []ListenerStatus `json:"listeners,omitempty"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:storageversion
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Address",type=string,JSONPath=`.status.addresses[*].value`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Gateway is the Schema for the gateways API
type Gateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GatewaySpec   `json:"spec,omitempty"`
	Status GatewayStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GatewayList contains a list of Gateway
type GatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Gateway `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Gateway{}, &GatewayList{})
}
