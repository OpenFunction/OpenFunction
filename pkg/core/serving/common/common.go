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
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	openfunctionv1beta1 "github.com/openfunction/apis/core/v1beta1"
	openfunction "github.com/openfunction/apis/core/v1beta2"
	"github.com/openfunction/pkg/constants"
	"github.com/openfunction/pkg/core"
	"github.com/openfunction/pkg/util"
)

// DaprServiceMode is the inject mode for Dapr sidecar
type DaprServiceMode string

const (
	FunctionContextV1beta1EnvName = "FUNC_CONTEXT"
	FunctionContextV1beta2EnvName = "FUNC_CONTEXT_V1BETA2"

	ServingLabel                   = "openfunction.io/serving"
	ProxyLabel                     = "openfunction.io/proxy"
	OpenfunctionManaged            = "openfunction.io/managed"
	OpenfunctionDaprServiceMode    = "openfunction.io/dapr-service-mode"
	OpenfunctionDaprServiceEnabled = "openfunction.io/enable-dapr"
	DefaultDaprProxyImage          = "openfunction/dapr-proxy:v0.1.0"

	DaprEnabled         = "dapr.io/enabled"
	DaprAppID           = "dapr.io/app-id"
	DaprLogAsJSON       = "dapr.io/log-as-json"
	DaprAppProtocol     = "dapr.io/app-protocol"
	DaprAppPort         = "dapr.io/app-port"
	DaprMetricsPort     = "dapr.io/metrics-port"
	DaprListenAddresses = "dapr.io/sidecar-listen-addresses"

	DaprHostEnvVar      = "DAPR_HOST"
	DaprSidecarIPEnvVar = "DAPR_SIDECAR_IP"
	DaprProtocolEnvVar  = "APP_PROTOCOL"
	DaprProxyName       = "dapr-proxy"

	DaprServiceModeStandalone DaprServiceMode = "standalone"
	DaprServiceModeSidecar    DaprServiceMode = "sidecar"

	daprComponentKey = "dapr.io/component"

	PluginsTracingAnnotation = "plugins.tracing"
	PluginsAnnotation        = "plugins"

	hooksKey   = "hooks"
	tracingKey = "tracing"

	bindingsPrefix = "bindings"
	pubsubPrefix   = "pubsub"
	statePrefix    = "state"
)

func CreateComponents(
	ctx context.Context,
	logger logr.Logger,
	c client.Client,
	scheme *runtime.Scheme,
	s *openfunction.Serving) error {
	log := logger.WithName("CreateDaprComponents").
		WithValues("Serving", fmt.Sprintf("%s/%s", s.Namespace, s.Name))

	components := map[string]*componentsv1alpha1.ComponentSpec{}
	for name, component := range s.Spec.Bindings {
		components[bindingsPrefix+"-"+name] = component.DeepCopy()
	}

	for name, component := range s.Spec.Pubsub {
		components[pubsubPrefix+"-"+name] = component.DeepCopy()
	}

	for name, component := range s.Spec.States {
		if component.Spec != nil {
			components[statePrefix+"-"+name] = component.Spec.DeepCopy()
		}
	}

	value := ""
	for name, daprComponent := range components {
		dc := daprComponent.DeepCopy()
		component := &componentsv1alpha1.Component{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-%s-", s.Name, name),
				Namespace:    s.Namespace,
				Labels: map[string]string{
					OpenfunctionManaged: "true",
					ServingLabel:        s.Name,
				},
			},
			Scopes: []string{fmt.Sprintf("%s-%s", GetFunctionName(s), s.Namespace)},
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
		s.Status.ResourceRef[daprComponentKey] = strings.TrimSuffix(value, ",")
	}

	return nil
}

func CreateDaprProxy(
	ctx context.Context,
	logger logr.Logger,
	c client.Client,
	scheme *runtime.Scheme,
	s *openfunction.Serving,
	cm map[string]string) error {

	labels := map[string]string{
		OpenfunctionManaged: "true",
		ProxyLabel:          s.Name,
	}
	labels = util.AppendLabels(s.Spec.Labels, labels)

	selector := &metav1.LabelSelector{
		MatchLabels: labels,
	}

	var port = int32(constants.DefaultFuncPort)
	if s.Spec.Triggers.Http != nil && s.Spec.Triggers.Http.Port != nil {
		port = *s.Spec.Triggers.Http.Port
	}

	annotations := map[string]string{
		DaprAppID:           fmt.Sprintf("%s-%s", GetFunctionName(s), s.Namespace),
		DaprLogAsJSON:       "true",
		DaprAppProtocol:     "grpc",
		DaprAppPort:         fmt.Sprintf("%d", port),
		DaprListenAddresses: "[::],0.0.0.0",
	}
	annotations = util.AppendLabels(s.Spec.Annotations, annotations)
	annotations[DaprEnabled] = "true"

	defaultConfig := util.GetDefaultConfig(ctx, c, logger)
	image := util.GetConfigOrDefault(defaultConfig,
		"openfunction.dapr-proxy.image",
		DefaultDaprProxyImage,
	)
	spec := &corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:            DaprProxyName,
				Image:           image,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Ports: []corev1.ContainerPort{{
					Name:          core.FunctionPort,
					ContainerPort: port,
					Protocol:      corev1.ProtocolTCP,
				}},
				Env: []corev1.EnvVar{
					{
						Name:  DaprProtocolEnvVar,
						Value: annotations[DaprAppProtocol],
					},
				},
			},
		},
	}

	if env, err := CreateFunctionContextENV(ctx, logger, c, s, cm); err != nil {
		return err
	} else {
		spec.Containers[0].Env = append(spec.Containers[0].Env, env...)
	}

	spec.Containers[0].Env = append(spec.Containers[0].Env, AddPodMetadataEnv(s.Namespace)...)

	template := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: *spec,
	}

	var replicas int32 = 1
	version := constants.DefaultFunctionVersion
	if s.Spec.Version != nil {
		version = *s.Spec.Version
	}
	version = strings.ReplaceAll(version, ".", "")
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-proxy-deployment-%s-", s.Name, version),
			Namespace:    s.Namespace,
			Labels:       labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: selector,
			Template: template,
		},
	}

	if err := controllerutil.SetControllerReference(s, deploy, scheme); err != nil {
		logger.Error(err, "Failed to SetControllerReference for proxy")
		return err
	}

	if err := c.Create(ctx, deploy); err != nil {
		logger.Error(err, "Failed to create proxy")
		return err
	}

	logger.V(1).Info("Proxy created", "Workload", deploy.GetName())

	s.Status.ResourceRef[DaprProxyName] = deploy.GetName()
	return nil
}

func CreateFunctionContextENV(ctx context.Context, logger logr.Logger, c client.Client, s *openfunction.Serving, cm map[string]string) ([]corev1.EnvVar, error) {
	var env []corev1.EnvVar
	if v, err := GenOpenFunctionContextV1beta1(ctx, logger, c, s, cm); err != nil {
		return nil, err
	} else {
		env = append(env, corev1.EnvVar{
			Name:  FunctionContextV1beta1EnvName,
			Value: v,
		})
	}

	if v, err := GenOpenFunctionContextV1beta2(ctx, logger, c, s, cm); err != nil {
		return nil, err
	} else {
		env = append(env, corev1.EnvVar{
			Name:  FunctionContextV1beta2EnvName,
			Value: v,
		})
	}

	return env, nil
}

func CleanDaprProxy(
	ctx context.Context,
	logger logr.Logger,
	c client.Client,
	s *openfunction.Serving) error {
	deploymentList := &appsv1.DeploymentList{}
	if err := c.List(ctx, deploymentList, client.InNamespace(s.Namespace), client.MatchingLabels{ProxyLabel: s.Name}); err != nil {
		return err
	}

	for _, item := range deploymentList.Items {
		if strings.HasPrefix(item.Name, s.Name) {
			if err := c.Delete(context.Background(), &item); util.IgnoreNotFound(err) != nil {
				return err
			}
			logger.V(1).Info("Delete Deployment", "Deployment", item.Name)
		}
	}

	return nil
}

func AddPodMetadataEnv(namespace string) []corev1.EnvVar {
	podNameEnv := corev1.EnvVar{
		Name: "POD_NAME",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath:  "metadata.name",
			},
		},
	}
	podNamespaceEnv := corev1.EnvVar{
		Name:  "POD_NAMESPACE",
		Value: namespace,
	}
	return []corev1.EnvVar{
		podNameEnv,
		podNamespaceEnv,
	}
}

func GetDaprServiceMode(s *openfunction.Serving) DaprServiceMode {
	var mode DaprServiceMode
	if m, ok := s.Spec.Annotations[OpenfunctionDaprServiceMode]; !ok {
		mode = DaprServiceModeStandalone
	} else {
		mode = DaprServiceMode(m)
	}
	return mode
}

func GetDaprServiceEnabled(s *openfunction.Serving) bool {
	if enabled, ok := s.Spec.Annotations[OpenfunctionDaprServiceEnabled]; !ok {
		if len(s.Spec.Triggers.Dapr) != 0 ||
			len(s.Spec.Triggers.Inputs) != 0 ||
			s.Spec.Outputs != nil ||
			s.Spec.States != nil {
			return true
		} else {
			return false
		}
	} else if enabled == "false" {
		return false
	}
	return true
}

func NeedCreateDaprProxy(s *openfunction.Serving) bool {
	enabled := GetDaprServiceEnabled(s)
	if !enabled {
		return false
	}

	mode := GetDaprServiceMode(s)
	if mode != DaprServiceModeStandalone {
		return false
	}

	return true
}

func NeedCreateDaprSidecar(s *openfunction.Serving) bool {
	if GetDaprServiceEnabled(s) && GetDaprServiceMode(s) == DaprServiceModeSidecar {
		return true
	}
	return false
}

func GetFunctionName(s *openfunction.Serving) string {
	return s.Labels[constants.FunctionLabel]
}

func GetProxyName(s *openfunction.Serving) string {
	if s.Status.ResourceRef == nil {
		return ""
	}
	return s.Status.ResourceRef[DaprProxyName]
}

func GenOpenFunctionContextV1beta1(ctx context.Context, logger logr.Logger, c client.Client, s *openfunction.Serving, cm map[string]string) (string, error) {
	var port = int32(constants.DefaultFuncPort)
	if s.Spec.Triggers.Http != nil && s.Spec.Triggers.Http.Port != nil {
		port = *s.Spec.Triggers.Http.Port
	}

	version := ""
	if s.Spec.Version != nil {
		version = *s.Spec.Version
	}

	ofnRuntime := openfunctionv1beta1.Knative
	if s.Spec.Triggers.Dapr != nil {
		ofnRuntime = openfunctionv1beta1.Async
	}

	fc := functionContextV1beta1{
		Name:    GetFunctionName(s),
		Version: version,
		Runtime: cases.Title(language.Und, cases.NoLower).String(string(ofnRuntime)),
		Port:    fmt.Sprintf("%d", port),
	}

	if s.Spec.Triggers.Dapr != nil {
		fc.Inputs = make(map[string]*functionInput)
		for _, item := range s.Spec.Triggers.Dapr {
			input := item.DeepCopy()
			componentType, err := getComponentType(ctx, c, s, input.Name, input.Type)
			if err != nil {
				return "", err
			}
			uri := input.Topic
			if strings.HasPrefix(componentType, bindingsPrefix) {
				uri = input.Name
			}
			fnInput := functionInput{
				Uri:           uri,
				ComponentName: getRealComponentName(s, input.Name, componentType),
				ComponentType: componentType,
			}
			if input.InputName != "" {
				fc.Inputs[input.InputName] = &fnInput
			} else {
				fc.Inputs[input.Name] = &fnInput
			}
		}
	}

	if s.Spec.Outputs != nil && len(s.Spec.Outputs) > 0 {
		fc.Outputs = make(map[string]*functionOutput)
		for _, item := range s.Spec.Outputs {
			if item.Dapr == nil {
				continue
			}
			output := item.DeepCopy()
			componentType, err := getComponentType(ctx, c, s, output.Dapr.Name, output.Dapr.Type)
			if err != nil {
				return "", err
			}
			uri := output.Dapr.Topic
			if strings.HasPrefix(componentType, bindingsPrefix) {
				uri = output.Dapr.Name
			}
			fnOutput := functionOutput{
				Uri:           uri,
				ComponentName: getRealComponentName(s, output.Dapr.Name, componentType),
				ComponentType: componentType,
				Metadata:      output.Dapr.Metadata,
				Operation:     output.Dapr.Operation,
			}
			if output.Dapr.OutputName != "" {
				fc.Outputs[output.Dapr.OutputName] = &fnOutput
			} else {
				fc.Outputs[output.Dapr.Name] = &fnOutput
			}
		}
	}

	if s.Spec.States != nil && len(s.Spec.States) > 0 {
		fc.States = make(map[string]*functionState)
		for name, state := range s.Spec.States {
			stateType, err := getStateType(ctx, c, s, name, state)
			if err != nil {
				return "", err
			}

			fnState := functionState{
				ComponentName: getRealComponentName(s, name, stateType),
				ComponentType: stateType,
			}
			fc.States[name] = &fnState
		}
	}

	// Handle plugins information
	parsePluginsCfg(logger, s, cm, &fc)

	bs, _ := jsoniter.Marshal(fc)
	return string(bs), nil
}

func getRealComponentName(s *openfunction.Serving, componentName, componentType string) string {
	resourceRefs := strings.Split(s.Status.ResourceRef[daprComponentKey], ",")
	realName := componentName
	for _, resourceRef := range resourceRefs {
		prefix := fmt.Sprintf("%s-%s-%s-", s.Name, getComponentTypePrefix(componentType), componentName)
		if strings.HasPrefix(resourceRef, prefix) {
			if !strings.Contains(strings.TrimPrefix(resourceRef, prefix), "-") {
				realName = resourceRef
			}
		}
	}

	return realName
}

func getComponentTypePrefix(componentType string) string {
	arrays := strings.Split(componentType, ".")
	if len(arrays) < 2 {
		return ""
	}

	return arrays[0]
}

// parsePluginsCfg parses the plugin configuration information from both ConfigMap and function annotations.
// The plugin configuration information obtained from the function annotations has a higher priority.
// The Tracing plugin is registered at the end of prePlugins and the beginning of postPlugins by default.
func parsePluginsCfg(logger logr.Logger, s *openfunction.Serving, cm map[string]string, fc *functionContextV1beta1) {
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
			logger.Error(err, "failed to unmarshal plugin config")
		} else {
			if plgCfg.Order != nil {
				var prePlgs []string
				prePlgs = append(prePlgs, plgCfg.Order...)
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
			logger.Error(err, "failed to unmarshal tracing config")
		} else {
			if tcCfg.Enabled {
				prePlugins = append(prePlugins, tcCfg.Provider.Name)
				postPlugins = append([]string{tcCfg.Provider.Name}, postPlugins...)
			}

			fc.PluginsTracing = tcCfg
		}
	}

	fc.PrePlugins = prePlugins
	fc.PostPlugins = postPlugins
}

func reverse(originSlice []string) []string {
	var reverseSlice []string
	for i := len(originSlice) - 1; i >= 0; i-- {
		reverseSlice = append(reverseSlice, originSlice[i])
	}
	return reverseSlice
}

func getComponentTypeFromServing(s *openfunction.Serving, name string) string {
	if s.Spec.Bindings != nil {
		if item := s.Spec.Bindings[name]; item != nil {
			return item.Type
		}
	}

	if s.Spec.Pubsub != nil {
		if item := s.Spec.Pubsub[name]; item != nil {
			return item.Type
		}
	}

	if s.Spec.States != nil {
		if item := s.Spec.States[name]; item != nil {
			if item.Spec != nil {
				return item.Spec.Type
			}
		}
	}

	return ""
}

func getExistingComponentType(ctx context.Context, c client.Client, s *openfunction.Serving, name string) (string, error) {
	// Component had created by others.
	dc := &componentsv1alpha1.Component{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: s.Namespace,
			Name:      name,
		},
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(dc), dc); err != nil {
		return "", err
	}

	return dc.Spec.Type, nil
}

func getComponentType(ctx context.Context, c client.Client, s *openfunction.Serving, componentName, componentType string) (string, error) {
	if componentType != "" {
		return componentType, nil
	}

	if t := getComponentTypeFromServing(s, componentName); t != "" {
		return t, nil
	}

	return getExistingComponentType(ctx, c, s, componentName)
}

func getStateType(ctx context.Context, c client.Client, s *openfunction.Serving, name string, state *openfunction.State) (string, error) {
	if state.Spec != nil {
		return state.Spec.Type, nil
	}

	// Component had created by others.
	component := &componentsv1alpha1.Component{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: s.Namespace,
			Name:      name,
		},
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(component), component); err != nil {
		return "", err
	}

	return component.Spec.Type, nil
}

func GenOpenFunctionContextV1beta2(ctx context.Context, logger logr.Logger, c client.Client, s *openfunction.Serving, cm map[string]string) (string, error) {
	version := ""
	if s.Spec.Version != nil {
		version = *s.Spec.Version
	}

	var pre, post []string
	globalPreHooks, globalPostHooks := getGlobalHooks(logger, cm)
	pre = globalPreHooks
	post = globalPostHooks

	if s.Spec.Hooks != nil {
		if s.Spec.Hooks.Policy == openfunction.HookPolicyOverride {
			pre = s.Spec.Hooks.Pre
			post = s.Spec.Hooks.Post
		} else {
			pre = append(globalPreHooks, s.Spec.Hooks.Pre...)
			post = s.Spec.Hooks.Post
			post = append(s.Spec.Hooks.Post, globalPostHooks...)
		}
	}

	fc := &functionContextV1beta2{
		Name:      GetFunctionName(s),
		Version:   version,
		Triggers:  s.Spec.Triggers.DeepCopy(),
		PreHooks:  pre,
		PostHooks: post,
		Tracing:   mergerTracingConfig(logger, s, cm),
	}

	if len(fc.Triggers.Dapr) > 0 {
		for index := 0; index < len(fc.Triggers.Dapr); index++ {
			trigger := fc.Triggers.Dapr[index]
			componentType, err := getComponentType(ctx, c, s, trigger.Name, trigger.Type)
			if err != nil {
				return "", err
			}

			fc.Triggers.Dapr[index].Name = getRealComponentName(s, trigger.Name, componentType)
		}
	}

	if len(s.Spec.Triggers.Inputs) > 0 {
		fc.Inputs = make(map[string]*functionComponent)
		for _, item := range s.Spec.Triggers.Inputs {
			if item.Dapr == nil {
				continue
			}

			input := item.DeepCopy()
			componentType, err := getComponentType(ctx, c, s, input.Dapr.Name, input.Dapr.Type)
			if err != nil {
				return "", err
			}

			fnInput := functionComponent{
				ComponentName: getRealComponentName(s, input.Dapr.Name, componentType),
				ComponentType: componentType,
				Topic:         input.Dapr.Topic,
			}
			fc.Inputs[input.Dapr.Name] = &fnInput
		}
	}

	if s.Spec.Outputs != nil && len(s.Spec.Outputs) > 0 {
		fc.Outputs = make(map[string]*functionComponent)
		for _, item := range s.Spec.Outputs {
			if item.Dapr == nil {
				continue
			}
			output := item.DeepCopy()
			componentType, err := getComponentType(ctx, c, s, output.Dapr.Name, output.Dapr.Type)
			if err != nil {
				return "", err
			}
			fnOutput := functionComponent{
				ComponentName: getRealComponentName(s, output.Dapr.Name, componentType),
				ComponentType: componentType,
				Metadata:      output.Dapr.Metadata,
				Operation:     output.Dapr.Operation,
			}
			fc.Outputs[output.Dapr.Name] = &fnOutput
		}
	}

	if s.Spec.States != nil && len(s.Spec.States) > 0 {
		fc.States = make(map[string]*functionComponent)
		for name, state := range s.Spec.States {
			stateType, err := getStateType(ctx, c, s, name, state)
			if err != nil {
				return "", err
			}
			fc.States[name] = &functionComponent{
				ComponentName: getRealComponentName(s, name, stateType),
				ComponentType: stateType,
			}
		}
	}

	bs, _ := jsoniter.Marshal(fc)
	return string(bs), nil
}

func getGlobalHooks(logger logr.Logger, cm map[string]string) ([]string, []string) {
	hooksRaw := ""
	// To compatible with v1beta1
	if raw, ok := cm[PluginsAnnotation]; ok {
		hooksRaw = raw
	}

	if raw, ok := cm[hooksKey]; ok {
		hooksRaw = raw
	}

	if hooksRaw == "" {
		return nil, nil
	}

	hooks := &openfunction.Hooks{}
	if err := yaml.Unmarshal([]byte(hooksRaw), hooks); err != nil {
		logger.Error(err, "failed to unmarshal global hook config")
		return nil, nil
	}

	return hooks.Pre, hooks.Post
}

func getGlobalTracingConfig(logger logr.Logger, cm map[string]string) *openfunction.TracingConfig {
	tracingRaw := ""
	// To compatible with v1beta1
	if raw, ok := cm[PluginsTracingAnnotation]; ok {
		tracingRaw = raw
	}

	if raw, ok := cm[tracingKey]; ok {
		tracingRaw = raw
	}

	if tracingRaw == "" {
		return nil
	}

	globalTracingConfig := &openfunction.TracingConfig{}
	if err := yaml.Unmarshal([]byte(tracingRaw), globalTracingConfig); err != nil {
		logger.Error(err, "failed to unmarshal global tracing config")
		return nil
	}

	return globalTracingConfig
}

func mergerTracingConfig(logger logr.Logger, s *openfunction.Serving, cm map[string]string) *openfunction.TracingConfig {
	tracingConfig := s.Spec.Tracing
	if tracingConfig != nil && !tracingConfig.Enabled {
		return nil
	}

	globalTracingConfig := getGlobalTracingConfig(logger, cm)
	if globalTracingConfig == nil {
		return tracingConfig
	}

	if tracingConfig == nil {
		return globalTracingConfig
	}

	if tracingConfig.Provider == nil {
		tracingConfig.Provider = globalTracingConfig.Provider
	} else {
		if globalTracingConfig.Provider != nil {
			if tracingConfig.Provider.Name == "" {
				tracingConfig.Provider.Name = globalTracingConfig.Provider.Name
			}

			if tracingConfig.Provider.OapServer == "" {
				tracingConfig.Provider.OapServer = globalTracingConfig.Provider.OapServer
			}

			if tracingConfig.Provider.Exporter == nil {
				tracingConfig.Provider.Exporter = globalTracingConfig.Provider.Exporter
			} else {
				if globalTracingConfig.Provider.Exporter != nil {
					if tracingConfig.Provider.Exporter.Name == "" {
						tracingConfig.Provider.Exporter.Name = globalTracingConfig.Provider.Exporter.Name
					}

					if tracingConfig.Provider.Exporter.Protocol == "" {
						tracingConfig.Provider.Exporter.Protocol = globalTracingConfig.Provider.Exporter.Protocol
					}

					if tracingConfig.Provider.Exporter.Endpoint == "" {
						tracingConfig.Provider.Exporter.Endpoint = globalTracingConfig.Provider.Exporter.Endpoint
					}

					if tracingConfig.Provider.Exporter.Headers == "" {
						tracingConfig.Provider.Exporter.Headers = globalTracingConfig.Provider.Exporter.Headers
					}

					if tracingConfig.Provider.Exporter.Compression == "" {
						tracingConfig.Provider.Exporter.Compression = globalTracingConfig.Provider.Exporter.Compression
					}

					if tracingConfig.Provider.Exporter.Timeout == "" {
						tracingConfig.Provider.Exporter.Timeout = globalTracingConfig.Provider.Exporter.Timeout
					}
				}
			}
		}
	}

	tracingConfig.Baggage = mergerMap(tracingConfig.Baggage, globalTracingConfig.Baggage)
	tracingConfig.Tags = mergerMap(tracingConfig.Tags, globalTracingConfig.Tags)

	return tracingConfig
}

func mergerMap(m1, m2 map[string]string) map[string]string {
	res := make(map[string]string)

	if m2 != nil {
		for k, v := range m2 {
			res[k] = v
		}
	}

	if m1 != nil {
		for k, v := range m1 {
			res[k] = v
		}
	}

	return res
}

func GetSkywalkingEnv(logger logr.Logger, s *openfunction.Serving, cm map[string]string) []corev1.EnvVar {
	oapServer := ""
	tracing := mergerTracingConfig(logger, s, cm)
	if tracing != nil &&
		tracing.Enabled &&
		tracing.Provider != nil &&
		tracing.Provider.Name == "skywalking" {
		oapServer = tracing.Provider.OapServer
	}

	var env []corev1.EnvVar
	if oapServer != "" {
		env = append(env, corev1.EnvVar{
			Name:  "SW_AGENT_COLLECTOR_BACKEND_SERVICES",
			Value: oapServer,
		})
		env = append(env, corev1.EnvVar{
			Name:  "SW_AGENT_NAME",
			Value: GetFunctionName(s),
		})
		env = append(env, corev1.EnvVar{
			Name:  "SW_AGENT_NAMESPACE",
			Value: s.Namespace,
		})
		env = append(env, corev1.EnvVar{
			Name:  "SW_AGENT_KEEP_TRACING",
			Value: "true",
		})
	}

	return env
}
