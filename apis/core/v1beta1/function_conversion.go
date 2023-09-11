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
// +kubebuilder:docs-gen:collapse=Apache License

package v1beta1

import (
	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/openfunction/apis/core/v1beta2"
)

/*
ConvertTo is expected to modify its argument to contain the converted object.
Most of the conversion is straightforward copying, except for converting our changed field.
*/
// ConvertTo converts this Function to the Hub version (v1beta2).
func (src *Function) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta2.Function)
	dst.ObjectMeta = src.ObjectMeta

	if src.Spec.Serving != nil {
		if err := src.convertServingTo(dst); err != nil {
			return err
		}
	}

	if src.Spec.Build != nil {
		dst.Spec.Build = &v1beta2.BuildImpl{}
		if err := src.convertBuildTo(dst); err != nil {
			return err
		}
	}

	dst.Spec.Version = src.Spec.Version
	dst.Spec.Image = src.Spec.Image
	dst.Spec.ImageCredentials = src.Spec.ImageCredentials
	dst.Spec.WorkloadRuntime = src.Spec.WorkloadRuntime

	// Status
	dst.Status.Build = convertConditionTo(src.Status.Build)
	dst.Status.Serving = convertConditionTo(src.Status.Serving)

	if src.Status.Sources != nil {
		for _, item := range src.Status.Sources {
			res := v1beta2.SourceResult{
				Name: item.Name,
			}

			if item.Git != nil {
				res.Git = &v1beta2.GitSourceResult{
					CommitSha:    item.Git.CommitSha,
					CommitAuthor: item.Git.CommitAuthor,
					BranchName:   item.Git.BranchName,
				}
			}

			if item.Bundle != nil {
				res.Bundle = &v1beta2.BundleSourceResult{
					Digest: item.Bundle.Digest,
				}
			}

			dst.Status.Sources = append(dst.Status.Sources, res)
		}
	}

	if src.Status.Revision != nil {
		dst.Status.Revision = &v1beta2.Revision{
			ImageDigest: src.Status.Revision.ImageDigest,
		}
	}

	for _, item := range src.Status.Addresses {
		t := v1beta2.AddressType(*item.Type)
		dst.Status.Addresses = append(dst.Status.Addresses, v1beta2.FunctionAddress{
			Type:  &t,
			Value: item.Value,
		})
	}

	if src.Status.Route != nil {
		dst.Status.Route = &v1beta2.RouteStatus{
			Hosts:      src.Status.Route.Hosts,
			Paths:      src.Status.Route.Paths,
			Conditions: src.Status.Route.Conditions,
		}
	}

	return nil
}

func convertConditionTo(condition *Condition) *v1beta2.Condition {
	if condition == nil {
		return nil
	}

	return &v1beta2.Condition{
		State:                     condition.State,
		ResourceRef:               condition.ResourceRef,
		ResourceHash:              condition.ResourceHash,
		LastSuccessfulResourceRef: condition.LastSuccessfulResourceRef,
	}
}

func (src *Function) convertBuildTo(dst *v1beta2.Function) error {
	dst.Spec.Build.Builder = src.Spec.Build.Builder
	dst.Spec.Build.BuilderCredentials = src.Spec.Build.BuilderCredentials
	dst.Spec.Build.Env = src.Spec.Build.Env
	dst.Spec.Build.Dockerfile = src.Spec.Build.Dockerfile
	dst.Spec.Build.BuilderMaxAge = src.Spec.Build.BuilderMaxAge
	dst.Spec.Build.FailedBuildsHistoryLimit = src.Spec.Build.FailedBuildsHistoryLimit
	dst.Spec.Build.SuccessfulBuildsHistoryLimit = src.Spec.Build.SuccessfulBuildsHistoryLimit

	if src.Spec.Build.SrcRepo != nil {
		dst.Spec.Build.SrcRepo = &v1beta2.GitRepo{}
		dst.Spec.Build.SrcRepo.Url = src.Spec.Build.SrcRepo.Url
		dst.Spec.Build.SrcRepo.SourceSubPath = src.Spec.Build.SrcRepo.SourceSubPath
		dst.Spec.Build.SrcRepo.Revision = src.Spec.Build.SrcRepo.Revision
		dst.Spec.Build.SrcRepo.Credentials = src.Spec.Build.SrcRepo.Credentials

		if src.Spec.Build.SrcRepo.BundleContainer != nil {
			dst.Spec.Build.SrcRepo.BundleContainer = &v1beta2.BundleContainer{
				Image: src.Spec.Build.SrcRepo.Credentials.Name,
			}
		}
	}

	if src.Spec.Build.Shipwright != nil ||
		(src.Spec.Build.Params != nil && len(src.Spec.Build.Params) > 0) {
		dst.Spec.Build.Shipwright = &v1beta2.ShipwrightEngine{}
		if src.Spec.Build.Shipwright.Strategy != nil {
			dst.Spec.Build.Shipwright.Strategy = &v1beta2.Strategy{}
			dst.Spec.Build.Shipwright.Strategy.Name = src.Spec.Build.Shipwright.Strategy.Name
			dst.Spec.Build.Shipwright.Strategy.Kind = src.Spec.Build.Shipwright.Strategy.Kind
		}
		dst.Spec.Build.Shipwright.Timeout = src.Spec.Build.Shipwright.Timeout

		if src.Spec.Build.Params != nil {
			for k, v := range src.Spec.Build.Params {
				value := v
				dst.Spec.Build.Shipwright.Params = append(dst.Spec.Build.Shipwright.Params, &v1beta2.ParamValue{
					SingleValue: &v1beta2.SingleValue{
						Value: &value,
					},
					Name: k,
				})
			}
		}
	}
	return nil
}

func convertRouteTo(route *RouteImpl) *v1beta2.RouteImpl {
	if route == nil {
		return nil
	}

	nr := &v1beta2.RouteImpl{
		Hostnames: route.Hostnames,
		Rules:     route.Rules,
	}
	if route.CommonRouteSpec.GatewayRef != nil {
		nr.CommonRouteSpec = v1beta2.CommonRouteSpec{
			GatewayRef: &v1beta2.GatewayRef{
				Name:      route.GatewayRef.Name,
				Namespace: route.GatewayRef.Namespace,
			},
		}
	}

	return nr
}

type plugins struct {
	Order []string `yaml:"order,omitempty"`
	Pre   []string `yaml:"pre,omitempty"`
	Post  []string `yaml:"post,omitempty"`
}

func (src *Function) convertServingTo(dst *v1beta2.Function) error {
	dst.Spec.Serving = &v1beta2.ServingImpl{
		Pubsub:      src.Spec.Serving.Pubsub,
		Bindings:    src.Spec.Serving.Bindings,
		Params:      src.Spec.Serving.Params,
		Labels:      src.Spec.Serving.Labels,
		Annotations: src.Spec.Serving.Annotations,
		Template:    src.Spec.Serving.Template,
		Timeout:     src.Spec.Serving.Timeout,
	}
	dst.Spec.Serving.Triggers = &v1beta2.Triggers{}
	if src.Spec.Serving.Runtime == Knative {
		dst.Spec.Serving.Triggers.Http = &v1beta2.HttpTrigger{
			Port:  src.Spec.Port,
			Route: convertRouteTo(src.Spec.Route),
		}
	} else if src.Spec.Serving.Runtime == Async {
		for _, item := range src.Spec.Serving.Inputs {
			component := getDaprComponent(src.Spec.Serving.Bindings, src.Spec.Serving.Pubsub, item.Component)
			trigger := &v1beta2.DaprTrigger{
				DaprComponentRef: &v1beta2.DaprComponentRef{
					Name:  item.Component,
					Topic: item.Topic,
				},
				InputName: item.Name,
			}
			if component != nil {
				trigger.Type = component.Type
			}
			dst.Spec.Serving.Triggers.Dapr = append(dst.Spec.Serving.Triggers.Dapr, trigger)
		}
	}

	if src.Spec.Serving.ScaleOptions != nil {
		dst.Spec.Serving.ScaleOptions = &v1beta2.ScaleOptions{
			MaxReplicas: src.Spec.Serving.ScaleOptions.MaxReplicas,
			MinReplicas: src.Spec.Serving.ScaleOptions.MinReplicas,
			Knative:     src.Spec.Serving.ScaleOptions.Knative,
		}

		if src.Spec.Serving.ScaleOptions.Keda != nil {
			dst.Spec.Serving.ScaleOptions.Keda = &v1beta2.KedaScaleOptions{}
			if src.Spec.Serving.ScaleOptions.Keda.ScaledObject != nil {
				dst.Spec.Serving.ScaleOptions.Keda.ScaledObject = &v1beta2.KedaScaledObject{
					PollingInterval: src.Spec.Serving.ScaleOptions.Keda.ScaledObject.PollingInterval,
					CooldownPeriod:  src.Spec.Serving.ScaleOptions.Keda.ScaledObject.CooldownPeriod,
					Advanced:        src.Spec.Serving.ScaleOptions.Keda.ScaledObject.Advanced,
				}

				if src.Spec.Serving.ScaleOptions.Keda.ScaledObject.MaxReplicaCount != nil {
					dst.Spec.Serving.ScaleOptions.MaxReplicas = src.Spec.Serving.ScaleOptions.Keda.ScaledObject.MaxReplicaCount
				}

				if src.Spec.Serving.ScaleOptions.Keda.ScaledObject.MinReplicaCount != nil {
					dst.Spec.Serving.ScaleOptions.MinReplicas = src.Spec.Serving.ScaleOptions.Keda.ScaledObject.MinReplicaCount
				}

				dst.Spec.Serving.WorkloadType = WorkloadTypeDeployment
				if src.Spec.Serving.ScaleOptions.Keda.ScaledObject.WorkloadType != "" {
					dst.Spec.Serving.WorkloadType = src.Spec.Serving.ScaleOptions.Keda.ScaledObject.WorkloadType
				}
			}

			if src.Spec.Serving.ScaleOptions.Keda.ScaledJob != nil {
				dst.Spec.Serving.ScaleOptions.Keda.ScaledJob = &v1beta2.KedaScaledJob{
					RestartPolicy:              src.Spec.Serving.ScaleOptions.Keda.ScaledJob.RestartPolicy,
					PollingInterval:            src.Spec.Serving.ScaleOptions.Keda.ScaledObject.PollingInterval,
					SuccessfulJobsHistoryLimit: src.Spec.Serving.ScaleOptions.Keda.ScaledJob.SuccessfulJobsHistoryLimit,
					FailedJobsHistoryLimit:     src.Spec.Serving.ScaleOptions.Keda.ScaledJob.FailedJobsHistoryLimit,
					ScalingStrategy:            src.Spec.Serving.ScaleOptions.Keda.ScaledJob.ScalingStrategy,
				}

				if src.Spec.Serving.ScaleOptions.Keda.ScaledJob.MaxReplicaCount != nil {
					dst.Spec.Serving.ScaleOptions.MaxReplicas = src.Spec.Serving.ScaleOptions.Keda.ScaledJob.MaxReplicaCount
				}

				dst.Spec.Serving.WorkloadType = WorkloadTypeJob
			}

			if src.Spec.Serving.Triggers != nil {
				for _, item := range src.Spec.Serving.Triggers {
					dst.Spec.Serving.ScaleOptions.Keda.Triggers = append(dst.Spec.Serving.ScaleOptions.Keda.Triggers, item.ScaleTriggers)
				}
			}
		}
	}

	if src.Spec.Serving.Outputs != nil {
		for _, item := range src.Spec.Serving.Outputs {
			component := getDaprComponent(src.Spec.Serving.Bindings, src.Spec.Serving.Pubsub, item.Component)
			if component == nil {
				continue
			}
			dst.Spec.Serving.Outputs = append(dst.Spec.Serving.Outputs, &v1beta2.Output{
				Dapr: &v1beta2.DaprOutput{
					DaprComponentRef: &v1beta2.DaprComponentRef{
						Type:  component.Type,
						Name:  item.Component,
						Topic: item.Topic,
					},
					Metadata:   item.Params,
					Operation:  item.Operation,
					OutputName: item.Name,
				},
			})
		}
	}

	if src.Spec.Serving.States != nil {
		dst.Spec.Serving.States = make(map[string]*v1beta2.State)
		for k, v := range src.Spec.Serving.States {
			dst.Spec.Serving.States[k] = &v1beta2.State{
				Spec: v,
			}
		}
	}

	if src.Annotations != nil {
		if tracingRaw := src.Annotations["plugins.tracing"]; tracingRaw != "" {
			tracingConfig := &v1beta2.TracingConfig{}
			if err := yaml.Unmarshal([]byte(tracingRaw), tracingConfig); err != nil {
				return err
			}

			dst.Spec.Serving.Tracing = tracingConfig
		}

		if pluginsRaw := src.Annotations["plugins"]; pluginsRaw != "" {
			plugins := &plugins{}
			if err := yaml.Unmarshal([]byte(pluginsRaw), plugins); err != nil {
				return err
			}

			dst.Spec.Serving.Hooks = &v1beta2.Hooks{Policy: v1beta2.HookPolicyOverride}
			if plugins.Order != nil {
				var prePlgs []string
				prePlgs = append(prePlgs, plugins.Order...)
				dst.Spec.Serving.Hooks.Pre = prePlgs
				dst.Spec.Serving.Hooks.Post = reverse(prePlgs)
			}

			if plugins.Pre != nil {
				dst.Spec.Serving.Hooks.Pre = plugins.Pre
			}

			if plugins.Post != nil {
				dst.Spec.Serving.Hooks.Post = plugins.Post
			}
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

func getDaprComponent(bindings, pubsub map[string]*componentsv1alpha1.ComponentSpec, name string) *componentsv1alpha1.ComponentSpec {
	if bindings != nil {
		if c := bindings[name]; c != nil {
			return c
		}
	}

	if pubsub != nil {
		if c := pubsub[name]; c != nil {
			return c
		}
	}

	return nil
}

/*
ConvertFrom is expected to modify its receiver to contain the converted object.
Most of the conversion is straightforward copying, except for converting our changed field.
*/
// ConvertFrom converts from the Hub version (v1beta2) to this version.
func (dst *Function) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta2.Function)
	dst.ObjectMeta = src.ObjectMeta
	if src.Spec.Serving != nil {
		dst.Spec.Serving = &ServingImpl{}
		if err := dst.convertServingFrom(src); err != nil {
			return err
		}
	}

	if src.Spec.Build != nil {
		dst.Spec.Build = &BuildImpl{}
		if err := dst.convertBuildFrom(src); err != nil {
			return err
		}
	}

	dst.Spec.Version = src.Spec.Version
	dst.Spec.Image = src.Spec.Image
	dst.Spec.ImageCredentials = src.Spec.ImageCredentials

	dst.Spec.Version = src.Spec.Version
	dst.Spec.Image = src.Spec.Image
	dst.Spec.ImageCredentials = src.Spec.ImageCredentials
	dst.Spec.WorkloadRuntime = src.Spec.WorkloadRuntime

	// Status
	dst.Status.Build = convertConditionFrom(src.Status.Build)
	dst.Status.Serving = convertConditionFrom(src.Status.Serving)

	if src.Status.Sources != nil {
		for _, item := range src.Status.Sources {
			res := SourceResult{
				Name: item.Name,
			}

			if item.Git != nil {
				res.Git = &GitSourceResult{
					CommitSha:    item.Git.CommitSha,
					CommitAuthor: item.Git.CommitAuthor,
					BranchName:   item.Git.BranchName,
				}
			}

			if item.Bundle != nil {
				res.Bundle = &BundleSourceResult{
					Digest: item.Bundle.Digest,
				}
			}

			dst.Status.Sources = append(dst.Status.Sources, res)
		}

		if src.Status.Revision != nil {
			dst.Status.Revision = &Revision{
				ImageDigest: src.Status.Revision.ImageDigest,
			}
		}

		for _, item := range src.Status.Addresses {
			t := AddressType(*item.Type)
			dst.Status.Addresses = append(dst.Status.Addresses, FunctionAddress{
				Type:  &t,
				Value: item.Value,
			})
		}

		if src.Status.Route != nil {
			dst.Status.Route = &RouteStatus{
				Hosts:      src.Status.Route.Hosts,
				Paths:      src.Status.Route.Paths,
				Conditions: src.Status.Route.Conditions,
			}
		}
	}

	return nil
}

func convertConditionFrom(condition *v1beta2.Condition) *Condition {
	if condition == nil {
		return nil
	}

	return &Condition{
		State:                     condition.State,
		ResourceRef:               condition.ResourceRef,
		ResourceHash:              condition.ResourceHash,
		LastSuccessfulResourceRef: condition.LastSuccessfulResourceRef,
	}
}

func (dst *Function) convertBuildFrom(src *v1beta2.Function) error {
	dst.Spec.Build.Builder = src.Spec.Build.Builder
	dst.Spec.Build.BuilderCredentials = src.Spec.Build.BuilderCredentials
	dst.Spec.Build.Env = src.Spec.Build.Env
	dst.Spec.Build.Dockerfile = src.Spec.Build.Dockerfile
	dst.Spec.Build.BuilderMaxAge = src.Spec.Build.BuilderMaxAge
	dst.Spec.Build.FailedBuildsHistoryLimit = src.Spec.Build.FailedBuildsHistoryLimit
	dst.Spec.Build.SuccessfulBuildsHistoryLimit = src.Spec.Build.SuccessfulBuildsHistoryLimit

	if src.Spec.Build.SrcRepo != nil {
		dst.Spec.Build.SrcRepo = &GitRepo{}
		dst.Spec.Build.SrcRepo.Url = src.Spec.Build.SrcRepo.Url
		dst.Spec.Build.SrcRepo.SourceSubPath = src.Spec.Build.SrcRepo.SourceSubPath
		dst.Spec.Build.SrcRepo.Revision = src.Spec.Build.SrcRepo.Revision
		dst.Spec.Build.SrcRepo.Credentials = src.Spec.Build.SrcRepo.Credentials

		if src.Spec.Build.SrcRepo.BundleContainer != nil {
			dst.Spec.Build.SrcRepo.BundleContainer = &BundleContainer{
				Image: src.Spec.Build.SrcRepo.Credentials.Name,
			}
		}
	}

	if src.Spec.Build.Shipwright != nil {
		dst.Spec.Build.Shipwright = &ShipwrightEngine{}
		if src.Spec.Build.Shipwright.Strategy != nil {
			dst.Spec.Build.Shipwright.Strategy = &Strategy{}
			dst.Spec.Build.Shipwright.Strategy.Name = src.Spec.Build.Shipwright.Strategy.Name
			dst.Spec.Build.Shipwright.Strategy.Kind = src.Spec.Build.Shipwright.Strategy.Kind
		}
		dst.Spec.Build.Shipwright.Timeout = src.Spec.Build.Shipwright.Timeout

		if src.Spec.Build.Shipwright.Params != nil {
			dst.Spec.Build.Params = make(map[string]string)
			for _, item := range src.Spec.Build.Shipwright.Params {
				if item.SingleValue == nil || item.SingleValue.Value == nil {
					continue
				}

				dst.Spec.Build.Params[item.Name] = *item.Value
			}
		}
	}

	return nil
}

func (dst *Function) convertServingFrom(src *v1beta2.Function) error {
	dst.Spec.Serving = &ServingImpl{
		Pubsub:      src.Spec.Serving.Pubsub,
		Bindings:    src.Spec.Serving.Bindings,
		Params:      src.Spec.Serving.Params,
		Labels:      src.Spec.Serving.Labels,
		Annotations: src.Spec.Serving.Annotations,
		Template:    src.Spec.Serving.Template,
		Timeout:     src.Spec.Serving.Timeout,
	}

	if src.Spec.Serving.Triggers != nil {
		if src.Spec.Serving.Triggers.Http != nil {
			dst.Spec.Serving.Runtime = Knative
			dst.Spec.Port = src.Spec.Serving.Triggers.Http.Port
			dst.Spec.Route = convertRouteFrom(src.Spec.Serving.Triggers.Http.Route)
		} else if src.Spec.Serving.Triggers.Dapr != nil {
			dst.Spec.Serving.Runtime = Async
			for _, item := range src.Spec.Serving.Triggers.Dapr {
				input := &DaprIO{
					Name:      item.InputName,
					Component: item.Name,
					Topic:     item.Topic,
				}

				dst.Spec.Serving.Inputs = append(dst.Spec.Serving.Inputs, input)
			}
		}
	}

	if src.Spec.Serving.ScaleOptions != nil {
		dst.Spec.Serving.ScaleOptions = &ScaleOptions{
			MaxReplicas: src.Spec.Serving.ScaleOptions.MaxReplicas,
			MinReplicas: src.Spec.Serving.ScaleOptions.MinReplicas,
			Knative:     src.Spec.Serving.ScaleOptions.Knative,
		}
		if src.Spec.Serving.ScaleOptions.Keda != nil {
			dst.Spec.Serving.ScaleOptions.Keda = &KedaScaleOptions{}
			if src.Spec.Serving.ScaleOptions.Keda.ScaledObject != nil {
				dst.Spec.Serving.ScaleOptions.Keda.ScaledObject = &KedaScaledObject{
					WorkloadType:    src.Spec.Serving.WorkloadType,
					PollingInterval: src.Spec.Serving.ScaleOptions.Keda.ScaledObject.PollingInterval,
					CooldownPeriod:  src.Spec.Serving.ScaleOptions.Keda.ScaledObject.CooldownPeriod,
					MinReplicaCount: src.Spec.Serving.ScaleOptions.MinReplicas,
					MaxReplicaCount: src.Spec.Serving.ScaleOptions.MaxReplicas,
					Advanced:        src.Spec.Serving.ScaleOptions.Keda.ScaledObject.Advanced,
				}
			}

			if src.Spec.Serving.ScaleOptions.Keda.ScaledJob != nil {
				dst.Spec.Serving.ScaleOptions.Keda.ScaledJob = &KedaScaledJob{
					RestartPolicy:              src.Spec.Serving.ScaleOptions.Keda.ScaledJob.RestartPolicy,
					PollingInterval:            src.Spec.Serving.ScaleOptions.Keda.ScaledJob.PollingInterval,
					SuccessfulJobsHistoryLimit: src.Spec.Serving.ScaleOptions.Keda.ScaledJob.SuccessfulJobsHistoryLimit,
					FailedJobsHistoryLimit:     src.Spec.Serving.ScaleOptions.Keda.ScaledJob.FailedJobsHistoryLimit,
					MaxReplicaCount:            src.Spec.Serving.ScaleOptions.MaxReplicas,
					ScalingStrategy:            src.Spec.Serving.ScaleOptions.Keda.ScaledJob.ScalingStrategy,
				}
			}

			if src.Spec.Serving.ScaleOptions.Keda.Triggers != nil {
				for _, item := range src.Spec.Serving.ScaleOptions.Keda.Triggers {
					dst.Spec.Serving.Triggers = append(dst.Spec.Serving.Triggers, Triggers{
						ScaleTriggers: item,
					})
				}
			}
		}
	}

	if src.Spec.Serving.Outputs != nil {
		for _, item := range src.Spec.Serving.Outputs {
			dst.Spec.Serving.Outputs = append(dst.Spec.Serving.Outputs, &DaprIO{
				Name:      item.Dapr.OutputName,
				Component: item.Dapr.Name,
				Params:    item.Dapr.Metadata,
				Operation: item.Dapr.Operation,
				Topic:     item.Dapr.Topic,
			})
		}
	}

	if src.Spec.Serving.States != nil {
		dst.Spec.Serving.States = make(map[string]*componentsv1alpha1.ComponentSpec)
		for k, v := range src.Spec.Serving.States {
			dst.Spec.Serving.States[k] = v.Spec
		}
	}

	if src.Spec.Serving.Hooks != nil {
		if dst.Annotations == nil {
			dst.Annotations = make(map[string]string)
		}

		plugins := &plugins{
			Post: src.Spec.Serving.Hooks.Post,
			Pre:  src.Spec.Serving.Hooks.Pre,
		}

		bs, err := yaml.Marshal(plugins)
		if err != nil {
			return err
		}

		dst.Annotations["plugins"] = string(bs)
	}

	if src.Spec.Serving.Tracing != nil {
		if dst.Annotations == nil {
			dst.Annotations = make(map[string]string)
		}

		bs, err := yaml.Marshal(src.Spec.Serving.Tracing)
		if err != nil {
			return err
		}

		dst.Annotations["plugins.tracing"] = string(bs)
	}

	return nil
}

func convertRouteFrom(route *v1beta2.RouteImpl) *RouteImpl {
	if route == nil {
		return nil
	}

	nr := &RouteImpl{
		Hostnames: route.Hostnames,
		Rules:     route.Rules,
	}
	if route.CommonRouteSpec.GatewayRef != nil {
		nr.CommonRouteSpec = CommonRouteSpec{
			GatewayRef: &GatewayRef{
				Name:      route.GatewayRef.Name,
				Namespace: route.GatewayRef.Namespace,
			},
		}
	}

	return nr
}
