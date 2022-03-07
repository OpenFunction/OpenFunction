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
	"bytes"
	"context"
	"fmt"
	"strings"

	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"github.com/go-logr/logr"
	jsoniter "github.com/json-iterator/go"
	yaml "gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	openfunction "github.com/openfunction/apis/core/v1beta1"
)

const (
	FunctionContextEnvName = "FUNC_CONTEXT"

	ServingLabel        = "openfunction.io/serving"
	OpenfunctionManaged = "openfunction.io/managed"

	DaprEnabled     = "dapr.io/enabled"
	DaprAppID       = "dapr.io/app-id"
	DaprLogAsJSON   = "dapr.io/log-as-json"
	DaprAppProtocol = "dapr.io/app-protocol"
	DaprAppPort     = "dapr.io/app-port"
	DaprMetricsPort = "dapr.io/metrics-port"

	PluginsTracingAnnotation = "plugins.tracing"
	PluginsAnnotation        = "plugins"
)

func GenOpenFunctionContext(
	ctx context.Context,
	logger logr.Logger,
	s *openfunction.Serving,
	cm map[string]string,
	components map[string]*componentsv1alpha1.ComponentSpec,
	functionName string,
	componentName string,
) string {
	log := logger.WithName("GenOpenFunctionContext").
		WithValues("Serving", fmt.Sprintf("%s/%s", s.Namespace, s.Name))

	var port int32 = 8080
	if s.Spec.Port != nil {
		port = *s.Spec.Port
	}

	version := ""
	if s.Spec.Version != nil {
		version = *s.Spec.Version
	}

	fc := functionContext{
		Name:    functionName,
		Version: version,
		Runtime: strings.Title(strings.ToLower(string(s.Spec.Runtime))),
		Port:    fmt.Sprintf("%d", port),
	}

	switch s.Spec.Runtime {
	case openfunction.Async:
		if s.Spec.Inputs != nil && len(s.Spec.Inputs) > 0 {
			fc.Inputs = make(map[string]*functionInput)

			for _, input := range s.Spec.Inputs {
				i := input.DeepCopy()
				c, _ := components[i.Component]
				componentType := strings.Split(c.Type, ".")[0]
				uri := i.Topic
				if componentType == bindings {
					uri = i.Component
				}
				fnInput := functionInput{
					Uri:       uri,
					Component: getComponentName(s, i.Component, componentName),
					Type:      componentType,
					Metadata:  i.Params,
				}
				fc.Inputs[i.Name] = &fnInput
			}
		}

		if s.Spec.Outputs != nil && len(s.Spec.Outputs) > 0 {
			fc.Outputs = make(map[string]*functionOutput)

			for _, output := range s.Spec.Outputs {
				o := output.DeepCopy()
				c, _ := components[o.Component]
				componentType := strings.Split(c.Type, ".")[0]
				uri := o.Topic
				if componentType == bindings {
					uri = o.Component
				}
				fnOutput := functionOutput{
					Uri:       uri,
					Component: getComponentName(s, o.Component, componentName),
					Type:      componentType,
					Metadata:  o.Params,
					Operation: o.Operation,
				}
				fc.Outputs[o.Name] = &fnOutput
			}
		}
	default:
		if s.Spec.Outputs != nil && len(s.Spec.Outputs) > 0 {
			fc.Outputs = make(map[string]*functionOutput)

			for _, output := range s.Spec.Outputs {
				o := output.DeepCopy()
				c, _ := components[o.Component]
				componentType := strings.Split(c.Type, ".")[0]
				uri := o.Topic
				if componentType == bindings {
					uri = o.Component
				}
				fnOutput := functionOutput{
					Uri:       uri,
					Component: getComponentName(s, o.Component, componentName),
					Type:      componentType,
					Metadata:  o.Params,
					Operation: o.Operation,
				}
				fc.Outputs[o.Name] = &fnOutput
			}
		}
	}

	// Handle plugins information
	if err := parsePluginsCfg(s, cm, &fc); err != nil {
		// Just log the error
		log.Error(err, "failed to parse plugins configuration.")
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
			c := component.DeepCopy()
			if _, exist := components[name]; exist {
				return nil, fmt.Errorf("dapr component with this name already exists: %s", name)
			}
			components[name] = c
		}
	}

	if s.Spec.Pubsub != nil {
		for name, component := range s.Spec.Pubsub {
			c := component.DeepCopy()
			if _, exist := components[name]; exist {
				return nil, fmt.Errorf("dapr component with this name already exists: %s", name)
			}
			components[name] = c
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
	for name, daprComponent := range components {
		dc := daprComponent.DeepCopy()
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
			for _, input := range s.Spec.Inputs {
				i := input.DeepCopy()
				if _, ok := components[i.Component]; !ok {
					cs = append(cs, i.Component)
				}
			}
		}

		if s.Spec.Outputs != nil && len(s.Spec.Outputs) > 0 {
			for _, output := range s.Spec.Outputs {
				o := output.DeepCopy()
				if _, ok := components[o.Component]; !ok {
					cs = append(cs, o.Component)
				}
			}
		}
	default:
		if s.Spec.Outputs != nil && len(s.Spec.Outputs) > 0 {
			for _, output := range s.Spec.Outputs {
				o := output.DeepCopy()
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

// parsePluginsCfg parses the plugin configuration information from both ConfigMap and function annotations.
// The plugin configuration information obtained from the function annotations has a higher priority.
// The Tracing plugin is registered at the end of prePlugins and the beginning of postPlugins by default.
func parsePluginsCfg(s *openfunction.Serving, cm map[string]string, fc *functionContext) error {
	var plgCfg = &plugins{}
	var tcCfg = &functionPluginsTracing{}
	var prePlugins []string
	var postPlugins []string

	pluginsRaw := ""
	pluginsTracingRaw := ""

	if raw, ok := cm[PluginsAnnotation]; ok {
		pluginsRaw = raw
	}
	if raw, ok := s.Annotations[PluginsAnnotation]; ok {
		pluginsRaw = raw
	}
	if pluginsRaw != "" {
		cfg := bytes.NewBufferString(pluginsRaw)
		if err := yaml.Unmarshal(cfg.Bytes(), plgCfg); err != nil {
			return err
		}
	}

	if raw, ok := cm[PluginsTracingAnnotation]; ok {
		pluginsTracingRaw = raw
	}
	if raw, ok := s.Annotations[PluginsTracingAnnotation]; ok {
		pluginsTracingRaw = raw
	}
	if pluginsTracingRaw != "" {
		cfg := bytes.NewBufferString(pluginsTracingRaw)
		if err := yaml.Unmarshal(cfg.Bytes(), tcCfg); err != nil {
			return err
		}
	}

	if plgCfg != nil {
		if plgCfg.Order != nil {
			prePlgs := []string{}
			for _, plg := range plgCfg.Order {
				prePlgs = append(prePlgs, plg)
			}
			prePlugins = prePlgs
			postPlugins = reverse(prePlgs)
		}

		if plgCfg.Pre != nil {
			prePlugins = plgCfg.Pre
		}

		if plgCfg.Post != nil {
			postPlugins = plgCfg.Post
		}
	}

	if tcCfg != nil && tcCfg.Enable {
		prePlugins = append(prePlugins, tcCfg.Provider.Name)
		postPlugins = append([]string{tcCfg.Provider.Name}, postPlugins...)
	}

	fc.PrePlugins = prePlugins
	fc.PostPlugins = postPlugins
	fc.PluginsTracing = tcCfg
	return nil
}

func reverse(originSlice []string) []string {
	reverseSlice := []string{}
	for i := len(originSlice) - 1; i >= 0; i-- {
		reverseSlice = append(reverseSlice, originSlice[i])
	}
	return reverseSlice
}

func AddPodMetadataEnv(namespace string) []v1.EnvVar {
	podNameEnv := v1.EnvVar{
		Name: "POD_NAME",
		ValueFrom: &v1.EnvVarSource{
			FieldRef: &v1.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath:  "metadata.name",
			},
		},
	}
	podNamespaceEnv := v1.EnvVar{
		Name:  "POD_NAMESPACE",
		Value: namespace,
	}
	return []v1.EnvVar{
		podNameEnv,
		podNamespaceEnv,
	}
}
