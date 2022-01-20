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

package common

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	functioncontext "github.com/OpenFunction/functions-framework-go/context"
	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	jsoniter "github.com/json-iterator/go"

	openfunction "github.com/openfunction/apis/core/v1beta1"
)

const (
	FUNCCONTEXT         = "FUNC_CONTEXT"
	ServingLabel        = "openfunction.io/serving"
	OpenfunctionManaged = "openfunction.io/managed"

	DaprEnabled     = "dapr.io/enabled"
	DaprAPPID       = "dapr.io/app-id"
	DaprLogAsJSON   = "dapr.io/log-as-json"
	DaprAPPProtocol = "dapr.io/app-protocol"
	DaprAPPPort     = "dapr.io/app-port"
	DaprMetricsPort = "dapr.io/metrics-port"
)

func GenOpenFunctionContext(s *openfunction.Serving, components map[string]*componentsv1alpha1.ComponentSpec, functionName string, componentName string) string {
	var port int32 = 8080
	if s.Spec.Port != nil {
		port = *s.Spec.Port
	}

	version := ""
	if s.Spec.Version != nil {
		version = *s.Spec.Version
	}

	fc := functioncontext.Context{
		Name:    functionName,
		Version: version,
		Runtime: functioncontext.Runtime(s.Spec.Runtime),
		Port:    fmt.Sprintf("%d", port),
	}

	switch s.Spec.Runtime {
	case openfunction.Async:
		if s.Spec.Inputs != nil && len(s.Spec.Inputs) > 0 {
			fc.Inputs = make(map[string]*functioncontext.Input)

			for _, i := range s.Spec.Inputs {
				c, _ := components[i.Component]
				componentType := strings.Split(c.Type, ".")[0]
				uri := i.Topic
				if componentType == string(functioncontext.OpenFuncBinding) {
					uri = i.Component
				}
				input := functioncontext.Input{
					Uri:       uri,
					Component: getComponentName(s, i.Component, componentName),
					Type:      functioncontext.ResourceType(componentType),
					Metadata:  i.Params,
				}
				fc.Inputs[i.Name] = &input
			}
		}

		if s.Spec.Outputs != nil && len(s.Spec.Outputs) > 0 {
			fc.Outputs = make(map[string]*functioncontext.Output)

			for _, o := range s.Spec.Outputs {
				c, _ := components[o.Component]
				componentType := strings.Split(c.Type, ".")[0]
				uri := o.Topic
				if componentType == string(functioncontext.OpenFuncBinding) {
					uri = o.Component
				}
				output := functioncontext.Output{
					Uri:       uri,
					Component: getComponentName(s, o.Component, componentName),
					Type:      functioncontext.ResourceType(componentType),
					Metadata:  o.Params,
					Operation: o.Operation,
				}
				fc.Outputs[o.Name] = &output
			}
		}
	default:
		if s.Spec.Outputs != nil && len(s.Spec.Outputs) > 0 {
			fc.Outputs = make(map[string]*functioncontext.Output)

			for _, o := range s.Spec.Outputs {
				c, _ := components[o.Component]
				componentType := strings.Split(c.Type, ".")[0]
				uri := o.Topic
				if componentType == string(functioncontext.OpenFuncBinding) {
					uri = o.Component
				}
				output := functioncontext.Output{
					Uri:       uri,
					Component: getComponentName(s, o.Component, componentName),
					Type:      functioncontext.ResourceType(componentType),
					Metadata:  o.Params,
					Operation: o.Operation,
				}
				fc.Outputs[o.Name] = &output
			}
		}
	}

	bs, _ := jsoniter.Marshal(fc)
	return string(bs)
}

func getComponentName(s *openfunction.Serving, name string, componentName string) string {

	names := strings.Split(s.Status.ResourceRef[componentName], ",")
	for _, n := range names {
		tmp := strings.TrimPrefix(n, fmt.Sprintf("%s-component-", s.Name))
		if index := strings.LastIndex(tmp, "-"); index != -1 {
			if tmp[:index] == name {
				return n
			}
		}
	}

	return name
}

func GetPendingCreateComponents(s *openfunction.Serving) (map[string]*componentsv1alpha1.ComponentSpec, error) {
	components := map[string]*componentsv1alpha1.ComponentSpec{}
	if s.Spec.Bindings != nil {
		for name, component := range s.Spec.Bindings {
			if _, exist := components[name]; exist {
				return nil, fmt.Errorf("dapr component with this name already exists: %s", name)
			}
			components[name] = component
		}
	}

	if s.Spec.Pubsub != nil {
		for name, component := range s.Spec.Pubsub {
			if _, exist := components[name]; exist {
				return nil, fmt.Errorf("dapr component with this name already exists: %s", name)
			}
			components[name] = component
		}
	}

	return components, nil
}

func CreateComponents(
	ctx context.Context,
	logger logr.Logger,
	c client.Client,
	scheme *runtime.Scheme,
	s *openfunction.Serving,
	components map[string]*componentsv1alpha1.ComponentSpec,
	componentName string,
) error {
	log := logger.WithName("CreateDaprComponents").
		WithValues("Serving", fmt.Sprintf("%s/%s", s.Namespace, s.Name))

	if components == nil {
		return nil
	}

	value := ""
	for name, dc := range components {
		component := &componentsv1alpha1.Component{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-component-%s-", s.Name, name),
				Namespace:    s.Namespace,
				Labels: map[string]string{
					OpenfunctionManaged: "true",
					ServingLabel:        s.Name,
				},
			},
		}

		if dc != nil {
			component.Spec = *dc
		}

		if err := controllerutil.SetControllerReference(s, component, scheme); err != nil {
			log.Error(err, "Failed to SetControllerReference", "Component", name)
			return err
		}

		if err := c.Create(ctx, component); err != nil {
			log.Error(err, "Failed to Create Dapr Component", "Component", name)
			return err
		}

		value = fmt.Sprintf("%s%s,", value, component.Name)
		log.V(1).Info("Component Created", "Component", component.Name)
	}

	if value != "" {
		s.Status.ResourceRef[componentName] = strings.TrimSuffix(value, ",")
	}

	return nil
}

func CheckComponentSpecExist(s *openfunction.Serving, components map[string]*componentsv1alpha1.ComponentSpec) error {
	var cs []string

	switch s.Spec.Runtime {
	case openfunction.Async:
		if s.Spec.Inputs != nil && len(s.Spec.Inputs) > 0 {
			for _, i := range s.Spec.Inputs {
				if _, ok := components[i.Component]; !ok {
					cs = append(cs, i.Component)
				}
			}
		}

		if s.Spec.Outputs != nil && len(s.Spec.Outputs) > 0 {
			for _, o := range s.Spec.Outputs {
				if _, ok := components[o.Component]; !ok {
					cs = append(cs, o.Component)
				}
			}
		}
	default:
		if s.Spec.Outputs != nil && len(s.Spec.Outputs) > 0 {
			for _, o := range s.Spec.Outputs {
				if _, ok := components[o.Component]; !ok {
					cs = append(cs, o.Component)
				}
			}
		}
	}

	if cs != nil && len(cs) > 0 {
		return fmt.Errorf("component %s does not exist", strings.Join(cs, ","))
	}
	return nil
}
