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

	openfunction "github.com/openfunction/apis/core/v1beta1"
	"github.com/openfunction/pkg/constants"
	"github.com/openfunction/pkg/core"
	"github.com/openfunction/pkg/util"
)

// DaprServiceMode is the inject mode for Dapr sidecar
type DaprServiceMode string

const (
	FunctionContextEnvName = "FUNC_CONTEXT"

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

	var port = int32(constants.DefaultFuncPort)
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
		Runtime: cases.Title(language.Und, cases.NoLower).String(string(s.Spec.Runtime)),
		Port:    fmt.Sprintf("%d", port),
	}

	switch s.Spec.Runtime {
	case openfunction.Async:
		if s.Spec.Inputs != nil && len(s.Spec.Inputs) > 0 {
			fc.Inputs = make(map[string]*functionInput)

			for _, input := range s.Spec.Inputs {
				i := input.DeepCopy()
				c, _ := components[i.Component]
				buildingBlockType := strings.Split(c.Type, ".")[0]
				uri := i.Topic
				if buildingBlockType == bindings {
					uri = i.Component
				}
				fnInput := functionInput{
					Uri:           uri,
					ComponentName: getComponentName(s, i.Component, componentName),
					ComponentType: c.Type,
					Metadata:      i.Params,
				}
				fc.Inputs[i.Name] = &fnInput
			}
		}

		if s.Spec.Outputs != nil && len(s.Spec.Outputs) > 0 {
			fc.Outputs = make(map[string]*functionOutput)

			for _, output := range s.Spec.Outputs {
				o := output.DeepCopy()
				c, _ := components[o.Component]
				buildingBlockType := strings.Split(c.Type, ".")[0]
				uri := o.Topic
				if buildingBlockType == bindings {
					uri = o.Component
				}
				fnOutput := functionOutput{
					Uri:           uri,
					ComponentName: getComponentName(s, o.Component, componentName),
					ComponentType: c.Type,
					Metadata:      o.Params,
					Operation:     o.Operation,
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
				buildingBlockType := strings.Split(c.Type, ".")[0]
				uri := o.Topic
				if buildingBlockType == bindings {
					uri = o.Component
				}
				fnOutput := functionOutput{
					Uri:           uri,
					ComponentName: getComponentName(s, o.Component, componentName),
					ComponentType: c.Type,
					Metadata:      o.Params,
					Operation:     o.Operation,
				}
				fc.Outputs[o.Name] = &fnOutput
			}
		}
	}

	if s.Spec.States != nil && len(s.Spec.States) > 0 {
		fc.States = make(map[string]*functionState)
		for name, _ := range s.Spec.States {
			c, _ := components[name]
			fnState := functionState{
				ComponentName: getComponentName(s, name, componentName),
				ComponentType: c.Type,
			}
			fc.States[name] = &fnState
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

	if s.Spec.States != nil {
		for name, component := range s.Spec.States {
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
			var prePlgs []string
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

	if tcCfg != nil && tcCfg.Enabled {
		prePlugins = append(prePlugins, tcCfg.Provider.Name)
		postPlugins = append([]string{tcCfg.Provider.Name}, postPlugins...)
	}

	fc.PrePlugins = prePlugins
	fc.PostPlugins = postPlugins
	fc.PluginsTracing = tcCfg
	return nil
}

func CreateDaprProxy(
	ctx context.Context,
	logger logr.Logger,
	c client.Client,
	scheme *runtime.Scheme,
	s *openfunction.Serving,
	cm map[string]string,
	components map[string]*componentsv1alpha1.ComponentSpec,
	componentName string) error {

	labels := map[string]string{
		OpenfunctionManaged: "true",
		ProxyLabel:          s.Name,
	}
	labels = util.AppendLabels(s.Spec.Labels, labels)

	selector := &metav1.LabelSelector{
		MatchLabels: labels,
	}

	var port = int32(constants.DefaultFuncPort)
	if s.Spec.Port != nil {
		port = *s.Spec.Port
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
						Name:  FunctionContextEnvName,
						Value: GenOpenFunctionContext(ctx, logger, s, cm, components, GetFunctionName(s), componentName),
					},
					{
						Name:  DaprProtocolEnvVar,
						Value: annotations[DaprAppProtocol],
					},
				},
			},
		},
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

func reverse(originSlice []string) []string {
	var reverseSlice []string
	for i := len(originSlice) - 1; i >= 0; i-- {
		reverseSlice = append(reverseSlice, originSlice[i])
	}
	return reverseSlice
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
		if s.Spec.Inputs != nil || s.Spec.Outputs != nil || s.Spec.States != nil {
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
