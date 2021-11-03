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

package v1alpha2

/*
For imports, we'll need the controller-runtime
[`conversion`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/conversion?tab=doc)
package, plus the API version for our hub type (v1), and finally some of the
standard packages.
*/
import (
	"fmt"

	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/openfunction/apis/core/v1alpha1"
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
// ConvertTo converts this CronJob to the Hub version (v1alpha1).
func (src *Function) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha1.Function)
	dst.ObjectMeta = src.ObjectMeta

	if src.Spec.Serving != nil {
		dst.Spec.Serving = &v1alpha1.ServingImpl{}
		if err := src.convertServingTo(dst); err != nil {
			return err
		}
	}

	if src.Spec.Build != nil {
		dst.Spec.Build = &v1alpha1.BuildImpl{}
		if err := src.convertBuildTo(dst); err != nil {
			return err
		}
	}

	dst.Spec.Version = src.Spec.Version
	dst.Spec.Image = src.Spec.Image
	dst.Spec.Port = src.Spec.Port
	dst.Spec.ImageCredentials = src.Spec.ImageCredentials

	// Status
	if src.Status.Build != nil {
		dst.Status.Build = &v1alpha1.Condition{
			State:        src.Status.Build.State,
			ResourceRef:  src.Status.Build.ResourceRef,
			ResourceHash: src.Status.Build.ResourceHash,
		}
	}
	if src.Status.Serving != nil {
		dst.Status.Serving = &v1alpha1.Condition{
			State:        src.Status.Serving.State,
			ResourceRef:  src.Status.Serving.ResourceRef,
			ResourceHash: src.Status.Serving.ResourceHash,
		}
	}

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}

func (src *Function) convertBuildTo(dst *v1alpha1.Function) error {
	dst.Spec.Build.Builder = src.Spec.Build.Builder
	dst.Spec.Build.BuilderCredentials = src.Spec.Build.BuilderCredentials
	dst.Spec.Build.Env = src.Spec.Build.Env
	dst.Spec.Build.Params = src.Spec.Build.Params
	dst.Spec.Build.Dockerfile = src.Spec.Build.Dockerfile

	if src.Spec.Build.SrcRepo != nil {
		dst.Spec.Build.SrcRepo = &v1alpha1.GitRepo{}
		dst.Spec.Build.SrcRepo.Url = src.Spec.Build.SrcRepo.Url
		dst.Spec.Build.SrcRepo.SourceSubPath = src.Spec.Build.SrcRepo.SourceSubPath
		dst.Spec.Build.SrcRepo.Revision = src.Spec.Build.SrcRepo.Revision
		dst.Spec.Build.SrcRepo.Credentials = src.Spec.Build.SrcRepo.Credentials
	}

	if src.Spec.Build.Shipwright != nil {
		dst.Spec.Build.Shipwright = &v1alpha1.ShipwrightEngine{}
		if src.Spec.Build.Shipwright.Strategy != nil {
			dst.Spec.Build.Shipwright.Strategy.Name = src.Spec.Build.Shipwright.Strategy.Name
			dst.Spec.Build.Shipwright.Strategy.Kind = src.Spec.Build.Shipwright.Strategy.Kind
		}
		dst.Spec.Build.Shipwright.Timeout = src.Spec.Build.Shipwright.Timeout
	}
	return nil
}

func (src *Function) convertServingTo(dst *v1alpha1.Function) error {
	dst.Spec.Serving.Runtime = (*v1alpha1.Runtime)(src.Spec.Serving.Runtime)

	dst.Spec.Serving.OpenFuncAsync = &v1alpha1.OpenFuncAsyncRuntime{}
	if src.Spec.Serving.OpenFuncAsync != nil {
		if src.Spec.Serving.OpenFuncAsync.Dapr != nil {
			dst.Spec.Serving.OpenFuncAsync.Dapr = &v1alpha1.Dapr{}
			dst.Spec.Serving.OpenFuncAsync.Dapr.Annotations = src.Spec.Serving.OpenFuncAsync.Dapr.Annotations

			if src.Spec.Serving.OpenFuncAsync.Dapr.Components != nil {
				dst.Spec.Serving.OpenFuncAsync.Dapr.Components = []v1alpha1.DaprComponent{}
				for name, component := range src.Spec.Serving.OpenFuncAsync.Dapr.Components {
					dc := v1alpha1.DaprComponent{
						Name:          name,
						ComponentSpec: *component,
					}
					dst.Spec.Serving.OpenFuncAsync.Dapr.Components = append(dst.Spec.Serving.OpenFuncAsync.Dapr.Components, dc)
				}
			}

			if src.Spec.Serving.OpenFuncAsync.Dapr.Inputs != nil {
				dst.Spec.Serving.OpenFuncAsync.Dapr.Inputs = []*v1alpha1.DaprIO{}
				for _, input := range src.Spec.Serving.OpenFuncAsync.Dapr.Inputs {
					in := v1alpha1.DaprIO{
						Name:   input.Component,
						Topic:  input.Topic,
						Type:   input.Type,
						Params: input.Params,
					}
					dst.Spec.Serving.OpenFuncAsync.Dapr.Inputs = append(dst.Spec.Serving.OpenFuncAsync.Dapr.Inputs, &in)
				}
			}

			if src.Spec.Serving.OpenFuncAsync.Dapr.Outputs != nil {
				dst.Spec.Serving.OpenFuncAsync.Dapr.Outputs = []*v1alpha1.DaprIO{}
				for _, output := range src.Spec.Serving.OpenFuncAsync.Dapr.Outputs {
					output.Params = map[string]string{}
					output.Params["operation"] = output.Operation
					out := v1alpha1.DaprIO{
						Name:   output.Component,
						Topic:  output.Topic,
						Type:   output.Type,
						Params: output.Params,
					}
					dst.Spec.Serving.OpenFuncAsync.Dapr.Outputs = append(dst.Spec.Serving.OpenFuncAsync.Dapr.Outputs, &out)
				}
			}
		}

		if src.Spec.Serving.OpenFuncAsync.Keda != nil {
			dst.Spec.Serving.OpenFuncAsync.Keda = &v1alpha1.Keda{}
			if src.Spec.Serving.OpenFuncAsync.Keda.ScaledJob != nil {
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob = &v1alpha1.KedaScaledJob{}
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.ScalingStrategy = src.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.ScalingStrategy
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.Triggers = src.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.Triggers
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.FailedJobsHistoryLimit = src.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.FailedJobsHistoryLimit
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.SuccessfulJobsHistoryLimit = src.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.SuccessfulJobsHistoryLimit
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.MaxReplicaCount = src.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.MaxReplicaCount
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.PollingInterval = src.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.PollingInterval
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.RestartPolicy = src.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.RestartPolicy
			}

			if src.Spec.Serving.OpenFuncAsync.Keda.ScaledObject != nil {
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject = &v1alpha1.KedaScaledObject{}
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.PollingInterval = src.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.PollingInterval
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.CooldownPeriod = src.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.CooldownPeriod
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.WorkloadType = src.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.WorkloadType
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.Advanced = src.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.Advanced
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.MinReplicaCount = src.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.MinReplicaCount
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.MaxReplicaCount = src.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.MaxReplicaCount
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.Triggers = src.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.Triggers
			}
		}
	}
	return nil
}

/*
ConvertFrom is expected to modify its receiver to contain the converted object.
Most of the conversion is straightforward copying, except for converting our changed field.
*/

// ConvertFrom converts from the Hub version (v1alpha1) to this version.
func (dst *Function) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha1.Function)
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
	dst.Spec.Port = src.Spec.Port
	dst.Spec.ImageCredentials = src.Spec.ImageCredentials

	// Status
	if src.Status.Build != nil {
		dst.Status.Build = &Condition{
			State:        src.Status.Build.State,
			ResourceRef:  src.Status.Build.ResourceRef,
			ResourceHash: src.Status.Build.ResourceHash,
		}
	}
	if src.Status.Serving != nil {
		dst.Status.Serving = &Condition{
			State:          src.Status.Serving.State,
			ResourceRef:    src.Status.Serving.ResourceRef,
			OldResourceRef: src.Status.Serving.ResourceRef,
			ResourceHash:   src.Status.Serving.ResourceHash,
		}
	}

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}

func (dst *Function) convertBuildFrom(src *v1alpha1.Function) error {
	dst.Spec.Build.Builder = src.Spec.Build.Builder
	dst.Spec.Build.BuilderCredentials = src.Spec.Build.BuilderCredentials
	dst.Spec.Build.Env = src.Spec.Build.Env
	dst.Spec.Build.Params = src.Spec.Build.Params
	dst.Spec.Build.Dockerfile = src.Spec.Build.Dockerfile

	if src.Spec.Build.SrcRepo != nil {
		dst.Spec.Build.SrcRepo = &GitRepo{}
		dst.Spec.Build.SrcRepo.Url = src.Spec.Build.SrcRepo.Url
		dst.Spec.Build.SrcRepo.SourceSubPath = src.Spec.Build.SrcRepo.SourceSubPath
		dst.Spec.Build.SrcRepo.Revision = src.Spec.Build.SrcRepo.Revision
		dst.Spec.Build.SrcRepo.Credentials = src.Spec.Build.SrcRepo.Credentials
	}

	if src.Spec.Build.Shipwright != nil {
		dst.Spec.Build.Shipwright = &ShipwrightEngine{}
		if src.Spec.Build.Shipwright.Strategy != nil {
			dst.Spec.Build.Shipwright.Strategy.Name = src.Spec.Build.Shipwright.Strategy.Name
			dst.Spec.Build.Shipwright.Strategy.Kind = src.Spec.Build.Shipwright.Strategy.Kind
		}
		dst.Spec.Build.Shipwright.Timeout = src.Spec.Build.Shipwright.Timeout
	}
	return nil
}

func (dst *Function) convertServingFrom(src *v1alpha1.Function) error {
	dst.Spec.Serving.Runtime = (*Runtime)(src.Spec.Serving.Runtime)

	if src.Spec.Serving.Params != nil {
		dst.Spec.Serving.Params = src.Spec.Serving.Params
	}

	if src.Spec.Serving.Template != nil {
		dst.Spec.Serving.Template = src.Spec.Serving.Template
	}

	if src.Spec.Serving.OpenFuncAsync != nil {
		dst.Spec.Serving.OpenFuncAsync = &OpenFuncAsyncRuntime{}
		if src.Spec.Serving.OpenFuncAsync.Dapr != nil {
			dst.Spec.Serving.OpenFuncAsync.Dapr = &Dapr{}
			if src.Spec.Serving.OpenFuncAsync.Dapr.Annotations != nil {
				dst.Spec.Serving.OpenFuncAsync.Dapr.Annotations = src.Spec.Serving.OpenFuncAsync.Dapr.Annotations
			}

			if src.Spec.Serving.OpenFuncAsync.Dapr.Components != nil {
				dst.Spec.Serving.OpenFuncAsync.Dapr.Components = map[string]*componentsv1alpha1.ComponentSpec{}
				for _, component := range src.Spec.Serving.OpenFuncAsync.Dapr.Components {
					dst.Spec.Serving.OpenFuncAsync.Dapr.Components[component.Name] = &component.ComponentSpec
				}
			}

			if src.Spec.Serving.OpenFuncAsync.Dapr.Inputs != nil {
				for _, input := range src.Spec.Serving.OpenFuncAsync.Dapr.Inputs {
					in := DaprIO{
						Name:      fmt.Sprintf("%s-%s-%s", src.Namespace, src.Name, input.Name),
						Component: input.Name,
						Topic:     input.Topic,
						Params:    input.Params,
					}
					dst.Spec.Serving.OpenFuncAsync.Dapr.Inputs = append(dst.Spec.Serving.OpenFuncAsync.Dapr.Inputs, &in)
				}
			}

			if src.Spec.Serving.OpenFuncAsync.Dapr.Outputs != nil {
				for _, output := range src.Spec.Serving.OpenFuncAsync.Dapr.Outputs {
					operation, ok := output.Params["operation"]
					if !ok {
						return fmt.Errorf("cannot find opertion in params, output: %s", output.Name)
					}
					out := DaprIO{
						Name:      fmt.Sprintf("%s-%s-%s", src.Namespace, src.Name, output.Name),
						Component: output.Name,
						Topic:     output.Topic,
						Params:    output.Params,
						Operation: operation,
					}
					dst.Spec.Serving.OpenFuncAsync.Dapr.Outputs = append(dst.Spec.Serving.OpenFuncAsync.Dapr.Outputs, &out)
				}
			}
		}

		if src.Spec.Serving.OpenFuncAsync.Keda != nil {
			dst.Spec.Serving.OpenFuncAsync.Keda = &Keda{}
			if src.Spec.Serving.OpenFuncAsync.Keda.ScaledJob != nil {
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob = &KedaScaledJob{}
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.ScalingStrategy = src.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.ScalingStrategy
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.Triggers = src.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.Triggers
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.FailedJobsHistoryLimit = src.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.FailedJobsHistoryLimit
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.SuccessfulJobsHistoryLimit = src.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.SuccessfulJobsHistoryLimit
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.MaxReplicaCount = src.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.MaxReplicaCount
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.PollingInterval = src.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.PollingInterval
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.RestartPolicy = src.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.RestartPolicy
			}

			if src.Spec.Serving.OpenFuncAsync.Keda.ScaledObject != nil {
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject = &KedaScaledObject{}
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.PollingInterval = src.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.PollingInterval
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.CooldownPeriod = src.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.CooldownPeriod
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.WorkloadType = src.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.WorkloadType
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.Advanced = src.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.Advanced
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.MinReplicaCount = src.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.MinReplicaCount
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.MaxReplicaCount = src.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.MaxReplicaCount
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.Triggers = src.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.Triggers
			}
		}
	}
	return nil
}
