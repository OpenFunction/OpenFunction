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

/*
   For imports, we'll need the controller-runtime
   [`conversion`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/conversion?tab=doc)
   package, plus the API version for our hub type (v1), and finally some of the
   standard packages.
*/
import (
	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/openfunction/apis/core/v1beta2"
)

// +kubebuilder:docs-gen:collapse=Imports

/*
Our "spoke" versions need to implement the
[`Convertible`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/conversion?tab=doc#Convertible)
interface.  Namely, they'll need `ConvertTo` and `ConvertFrom` methods to convert to/from
the hub version.
*/

/*
ConvertTo is expected to modify its argument to contain the converted object.
Most of the conversion is straightforward copying, except for converting our changed field.
*/
// ConvertTo converts this Function to the Hub version (v1beta2).
func (src *Serving) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta2.Serving)
	dst.ObjectMeta = src.ObjectMeta

	//if src == nil {
	//	fmt.Println(1)
	//}
	//
	//if dst == nil {
	//	fmt.Println(2)
	//}

	dst.Spec.Pubsub = src.Spec.Pubsub
	dst.Spec.Bindings = src.Spec.Bindings
	dst.Spec.Params = src.Spec.Params
	dst.Spec.Labels = src.Spec.Labels
	dst.Spec.Annotations = src.Spec.Annotations
	dst.Spec.Template = src.Spec.Template
	dst.Spec.Timeout = src.Spec.Timeout

	dst.Spec.Triggers = &v1beta2.Triggers{}
	if src.Spec.Runtime == Knative {
		dst.Spec.Triggers.Http = &v1beta2.HttpTrigger{
			Port: src.Spec.Port,
		}
	} else if src.Spec.Runtime == Async {
		for _, item := range src.Spec.Inputs {
			component := getDaprComponent(src.Spec.Bindings, src.Spec.Pubsub, item.Component)
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
			dst.Spec.Triggers.Dapr = append(dst.Spec.Triggers.Dapr, trigger)
		}
	}

	if src.Spec.ScaleOptions != nil {
		dst.Spec.ScaleOptions = &v1beta2.ScaleOptions{
			MaxReplicas: src.Spec.ScaleOptions.MaxReplicas,
			MinReplicas: src.Spec.ScaleOptions.MinReplicas,
			Knative:     src.Spec.ScaleOptions.Knative,
		}
		if src.Spec.ScaleOptions.Keda != nil &&
			src.Spec.ScaleOptions.Keda.ScaledObject != nil {
			dst.Spec.ScaleOptions.Keda = &v1beta2.KedaScaleOptions{}
			if src.Spec.ScaleOptions.Keda.ScaledObject != nil {
				dst.Spec.ScaleOptions.Keda.ScaledObject = &v1beta2.KedaScaledObject{
					PollingInterval: src.Spec.ScaleOptions.Keda.ScaledObject.PollingInterval,
					CooldownPeriod:  src.Spec.ScaleOptions.Keda.ScaledObject.CooldownPeriod,
					Advanced:        src.Spec.ScaleOptions.Keda.ScaledObject.Advanced,
				}

				if src.Spec.ScaleOptions.Keda.ScaledObject.MaxReplicaCount != nil {
					dst.Spec.ScaleOptions.MaxReplicas = src.Spec.ScaleOptions.Keda.ScaledObject.MaxReplicaCount
				}

				if src.Spec.ScaleOptions.Keda.ScaledObject.MinReplicaCount != nil {
					dst.Spec.ScaleOptions.MinReplicas = src.Spec.ScaleOptions.Keda.ScaledObject.MinReplicaCount
				}

				dst.Spec.WorkloadType = WorkloadTypeDeployment
				if src.Spec.ScaleOptions.Keda.ScaledObject.WorkloadType != "" {
					dst.Spec.WorkloadType = src.Spec.ScaleOptions.Keda.ScaledObject.WorkloadType
				}
			}

			if src.Spec.ScaleOptions.Keda.ScaledJob != nil {
				dst.Spec.ScaleOptions.Keda.ScaledJob = &v1beta2.KedaScaledJob{
					RestartPolicy:              src.Spec.ScaleOptions.Keda.ScaledJob.RestartPolicy,
					PollingInterval:            src.Spec.ScaleOptions.Keda.ScaledObject.PollingInterval,
					SuccessfulJobsHistoryLimit: src.Spec.ScaleOptions.Keda.ScaledJob.SuccessfulJobsHistoryLimit,
					FailedJobsHistoryLimit:     src.Spec.ScaleOptions.Keda.ScaledJob.FailedJobsHistoryLimit,
					ScalingStrategy:            src.Spec.ScaleOptions.Keda.ScaledJob.ScalingStrategy,
				}

				if src.Spec.ScaleOptions.Keda.ScaledJob.MaxReplicaCount != nil {
					dst.Spec.ScaleOptions.MaxReplicas = src.Spec.ScaleOptions.Keda.ScaledJob.MaxReplicaCount
				}
			}

			if src.Spec.Triggers != nil {
				for _, item := range src.Spec.Triggers {
					dst.Spec.ScaleOptions.Keda.Triggers = append(dst.Spec.ScaleOptions.Keda.Triggers, item.ScaleTriggers)
				}
			}
		}
	}

	if src.Spec.Outputs != nil {
		for _, item := range src.Spec.Outputs {
			component := getDaprComponent(src.Spec.Bindings, src.Spec.Pubsub, item.Component)
			if component == nil {
				continue
			}
			dst.Spec.Outputs = append(dst.Spec.Outputs, &v1beta2.Output{
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

	if src.Spec.States != nil {
		dst.Spec.States = make(map[string]*v1beta2.State)
		for k, v := range src.Spec.States {
			dst.Spec.States[k] = &v1beta2.State{
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

			dst.Spec.Tracing = tracingConfig
		}

		if pluginsRaw := src.Annotations["plugins"]; pluginsRaw != "" {
			plugins := &plugins{}
			if err := yaml.Unmarshal([]byte(pluginsRaw), plugins); err != nil {
				return err
			}

			dst.Spec.Hooks = &v1beta2.Hooks{Policy: v1beta2.HookPolicyOverride}
			if plugins.Order != nil {
				var prePlgs []string
				prePlgs = append(prePlgs, plugins.Order...)
				dst.Spec.Hooks.Pre = prePlgs
				dst.Spec.Hooks.Post = reverse(prePlgs)
			}

			if plugins.Pre != nil {
				dst.Spec.Hooks.Pre = plugins.Pre
			}

			if plugins.Post != nil {
				dst.Spec.Hooks.Post = plugins.Post
			}
		}

	}

	dst.Spec.Version = src.Spec.Version
	dst.Spec.Params = src.Spec.Params
	dst.Spec.Image = src.Spec.Image
	dst.Spec.Template = src.Spec.Template
	dst.Spec.ImageCredentials = src.Spec.ImageCredentials

	// Status
	dst.Status.State = src.Status.State
	dst.Status.Phase = src.Status.Phase
	dst.Status.ResourceRef = src.Status.ResourceRef
	return nil
}

/*
ConvertFrom is expected to modify its receiver to contain the converted object.
Most of the conversion is straightforward copying, except for converting our changed field.
*/
// ConvertFrom converts from the Hub version (v1beta2) to this version.
func (dst *Serving) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta2.Serving)
	dst.ObjectMeta = src.ObjectMeta

	dst.Spec.Pubsub = src.Spec.Pubsub
	dst.Spec.Bindings = src.Spec.Bindings
	dst.Spec.Params = src.Spec.Params
	dst.Spec.Labels = src.Spec.Labels
	dst.Spec.Annotations = src.Spec.Annotations
	dst.Spec.Template = src.Spec.Template
	dst.Spec.Timeout = src.Spec.Timeout

	if src.Spec.Triggers != nil {
		if src.Spec.Triggers.Http != nil {
			dst.Spec.Runtime = Knative
			dst.Spec.Port = src.Spec.Triggers.Http.Port
		} else if src.Spec.Triggers.Dapr != nil {
			dst.Spec.Runtime = Async
			for _, item := range src.Spec.Triggers.Dapr {
				input := &DaprIO{
					Name:      item.InputName,
					Component: item.Name,
					Topic:     item.Topic,
				}

				dst.Spec.Inputs = append(dst.Spec.Inputs, input)
			}
		}
	}

	if src.Spec.ScaleOptions != nil {
		dst.Spec.ScaleOptions = &ScaleOptions{
			MaxReplicas: src.Spec.ScaleOptions.MaxReplicas,
			MinReplicas: src.Spec.ScaleOptions.MinReplicas,
			Knative:     src.Spec.ScaleOptions.Knative,
		}
		if src.Spec.ScaleOptions.Keda != nil {
			dst.Spec.ScaleOptions.Keda = &KedaScaleOptions{}
			if src.Spec.ScaleOptions.Keda.ScaledObject != nil {
				dst.Spec.ScaleOptions.Keda.ScaledObject = &KedaScaledObject{
					WorkloadType:    src.Spec.WorkloadType,
					PollingInterval: src.Spec.ScaleOptions.Keda.ScaledObject.PollingInterval,
					CooldownPeriod:  src.Spec.ScaleOptions.Keda.ScaledObject.CooldownPeriod,
					MinReplicaCount: src.Spec.ScaleOptions.MinReplicas,
					MaxReplicaCount: src.Spec.ScaleOptions.MaxReplicas,
					Advanced:        src.Spec.ScaleOptions.Keda.ScaledObject.Advanced,
				}
			}

			if src.Spec.ScaleOptions.Keda.ScaledJob != nil {
				dst.Spec.ScaleOptions.Keda.ScaledJob = &KedaScaledJob{
					RestartPolicy:              src.Spec.ScaleOptions.Keda.ScaledJob.RestartPolicy,
					PollingInterval:            src.Spec.ScaleOptions.Keda.ScaledJob.PollingInterval,
					SuccessfulJobsHistoryLimit: src.Spec.ScaleOptions.Keda.ScaledJob.SuccessfulJobsHistoryLimit,
					FailedJobsHistoryLimit:     src.Spec.ScaleOptions.Keda.ScaledJob.FailedJobsHistoryLimit,
					MaxReplicaCount:            src.Spec.ScaleOptions.MaxReplicas,
					ScalingStrategy:            src.Spec.ScaleOptions.Keda.ScaledJob.ScalingStrategy,
				}
			}

			if src.Spec.ScaleOptions.Keda.Triggers != nil {
				for _, item := range src.Spec.ScaleOptions.Keda.Triggers {
					dst.Spec.Triggers = append(dst.Spec.Triggers, Triggers{
						ScaleTriggers: item,
					})
				}
			}
		}
	}

	if src.Spec.Outputs != nil {
		for _, item := range src.Spec.Outputs {
			dst.Spec.Outputs = append(dst.Spec.Outputs, &DaprIO{
				Name:      item.Dapr.OutputName,
				Component: item.Dapr.Name,
				Params:    item.Dapr.Metadata,
				Operation: item.Dapr.Operation,
				Topic:     item.Dapr.Topic,
			})
		}
	}

	if src.Spec.States != nil {
		dst.Spec.States = make(map[string]*componentsv1alpha1.ComponentSpec)
		for k, v := range src.Spec.States {
			dst.Spec.States[k] = v.Spec
		}
	}

	if src.Spec.Hooks != nil {
		if dst.Annotations == nil {
			dst.Annotations = make(map[string]string)
		}

		plugins := &plugins{
			Post: src.Spec.Hooks.Post,
			Pre:  src.Spec.Hooks.Pre,
		}

		bs, err := yaml.Marshal(plugins)
		if err != nil {
			return err
		}

		dst.Annotations["plugins"] = string(bs)
	}

	if src.Spec.Tracing != nil {
		if dst.Annotations == nil {
			dst.Annotations = make(map[string]string)
		}

		bs, err := yaml.Marshal(src.Spec.Tracing)
		if err != nil {
			return err
		}

		dst.Annotations["plugins.tracing"] = string(bs)
	}

	// Status
	dst.Status.State = src.Status.State
	dst.Status.Phase = src.Status.Phase
	dst.Status.ResourceRef = src.Status.ResourceRef
	return nil
}
