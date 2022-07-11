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
	"bytes"
	"fmt"
	"text/template"

	"k8s.io/apimachinery/pkg/util/json"

	"github.com/openfunction/pkg/constants"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	k8sgatewayapiv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// log is for logging in this package.
var gatewaylog = logf.Log.WithName("gateway-resource")

func (r *Gateway) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-networking-openfunction-io-v1alpha1-gateway,mutating=true,failurePolicy=fail,sideEffects=None,groups=networking.openfunction.io,resources=gateways,verbs=create;update,versions=v1alpha1,name=mgateway.of.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Gateway{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Gateway) Default() {
	gatewaylog.Info("default", "name", r.Name)

	if r.Spec.GatewayDef != nil && r.Spec.GatewayDef.Name == "" {
		r.Spec.GatewayDef.Name = r.GetName()
	}

	needInjectDefaultListeners := true
	for index, listener := range r.Spec.GatewaySpec.Listeners {
		if listener.Name == DefaultHttpListenerName {
			needInjectDefaultListeners = false
			internalHostname := k8sgatewayapiv1alpha2.Hostname(fmt.Sprintf("*.%s", r.Spec.ClusterDomain))
			namespaceFromAll := k8sgatewayapiv1alpha2.NamespacesFromAll
			listener.Hostname = &internalHostname
			listener.Port = constants.DefaultGatewayListenerPort
			listener.Protocol = constants.DefaultGatewayListenerProtocol
			listener.AllowedRoutes = &k8sgatewayapiv1alpha2.AllowedRoutes{
				Namespaces: &k8sgatewayapiv1alpha2.RouteNamespaces{
					From: &namespaceFromAll,
				},
			}
		} else {
			hostname := k8sgatewayapiv1alpha2.Hostname(fmt.Sprintf("*.%s", r.Spec.Domain))
			listener.Hostname = &hostname
		}
		r.Spec.GatewaySpec.Listeners[index] = listener
	}

	if needInjectDefaultListeners {
		internalHostname := k8sgatewayapiv1alpha2.Hostname(fmt.Sprintf("*.%s", r.Spec.ClusterDomain))
		namespaceFromAll := k8sgatewayapiv1alpha2.NamespacesFromAll
		internalHttpListener := k8sgatewayapiv1alpha2.Listener{
			Name:     DefaultHttpListenerName,
			Hostname: &internalHostname,
			Port:     constants.DefaultGatewayListenerPort,
			Protocol: constants.DefaultGatewayListenerProtocol,
			AllowedRoutes: &k8sgatewayapiv1alpha2.AllowedRoutes{
				Namespaces: &k8sgatewayapiv1alpha2.RouteNamespaces{
					From: &namespaceFromAll,
				},
			},
		}
		r.Spec.GatewaySpec.Listeners = append(r.Spec.GatewaySpec.Listeners, internalHttpListener)
	}

	if r.Annotations == nil {
		annotations := make(map[string]string)
		gatewaySpecAnnotation, _ := json.Marshal(r.Spec)
		annotations[GatewayConfigAnnotation] = string(gatewaySpecAnnotation)
		r.Annotations = annotations
	} else if _, ok := r.Annotations[GatewayConfigAnnotation]; !ok {
		gatewaySpecAnnotation, _ := json.Marshal(r.Spec)
		r.Annotations[GatewayConfigAnnotation] = string(gatewaySpecAnnotation)
	}
}

//+kubebuilder:webhook:path=/validate-networking-openfunction-io-v1alpha1-gateway,mutating=false,failurePolicy=fail,sideEffects=None,groups=networking.openfunction.io,resources=gateways,verbs=create;update,versions=v1alpha1,name=vgateway.of.io,admissionReviewVersions=v1

var _ webhook.Validator = &Gateway{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Gateway) ValidateCreate() error {
	gatewaylog.Info("validate create", "name", r.Name)
	return r.Validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Gateway) ValidateUpdate(old runtime.Object) error {
	gatewaylog.Info("validate update", "name", r.Name)
	return r.Validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Gateway) ValidateDelete() error {
	gatewaylog.Info("validate delete", "name", r.Name)

	return nil
}

func (r *Gateway) Validate() error {
	var hostnameBuffer bytes.Buffer
	var pathBuffer bytes.Buffer
	if r.Spec.Domain == "" {
		return field.Required(field.NewPath("spec", "domain"),
			"must specify domain")
	}

	if hostTemplate, err := template.New("host").Parse(r.Spec.HostTemplate); err != nil {
		return field.Invalid(field.NewPath("spec", "hostTemplate"),
			r.Spec.HostTemplate, "invalid template")
	} else {
		hostInfoObj := struct {
			Name      string
			Namespace string
			Domain    string
		}{Name: r.Name, Namespace: r.Namespace, Domain: r.Spec.Domain}
		if err := hostTemplate.Execute(&hostnameBuffer, hostInfoObj); err != nil {
			return field.Invalid(field.NewPath("spec", "hostTemplate"),
				r.Spec.HostTemplate, "invalid host template, please make sure that only the {{Name}}, {{Namespace}}, {{Domain}} are included in the template")
		}
	}

	if pathTemplate, err := template.New("path").Parse(r.Spec.PathTemplate); err != nil {
		return field.Invalid(field.NewPath("spec", "pathTemplate"),
			r.Spec.HostTemplate, "invalid path template")
	} else {
		pathInfoObj := struct {
			Name      string
			Namespace string
		}{Name: r.Name, Namespace: r.Namespace}
		if err := pathTemplate.Execute(&pathBuffer, pathInfoObj); err != nil {
			return field.Invalid(field.NewPath("spec", "pathTemplate"),
				r.Spec.HostTemplate, "invalid path template, please make sure that only the {{Name}} and {{Namespace}} are included in the template")
		}
	}

	if r.Spec.GatewayRef == nil && r.Spec.GatewayDef == nil {
		return field.Required(field.NewPath("spec", "gatewayRef"),
			"must specify at least one of gatewayRef and gatewayDef")
	}

	if r.Spec.GatewayRef != nil && r.Spec.GatewayDef != nil {
		return field.Invalid(field.NewPath("spec", "gatewayRef"),
			r.Spec.GatewayRef, "specify at most one of gatewayRef and gatewayDef")
	}

	if len(r.Spec.GatewaySpec.Listeners) == DefaultHttpListenersCount {
		return field.Required(field.NewPath("spec", "gatewaySpec", "listeners"),
			"must specify at least one listener")
	}
	return nil
}
