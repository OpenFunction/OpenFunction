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
func (src *Serving) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha1.Serving)
	dst.ObjectMeta = src.ObjectMeta

	if src.Spec.OpenFuncAsync != nil {
		dst.Spec.OpenFuncAsync = &v1alpha1.OpenFuncAsyncRuntime{}
		if err := src.convertOpenFuncAsyncTo(dst); err != nil {
			return err
		}
	}

	dst.Spec.Version = src.Spec.Version
	dst.Spec.Params = src.Spec.Params
	dst.Spec.Image = src.Spec.Image
	dst.Spec.Port = src.Spec.Port
	dst.Spec.Runtime = (*v1alpha1.Runtime)(src.Spec.Runtime)
	dst.Spec.Template = src.Spec.Template
	dst.Spec.ImageCredentials = src.Spec.ImageCredentials

	// Status
	dst.Status.State = src.Status.State
	dst.Status.Phase = src.Status.Phase
	dst.Status.ResourceRef = src.Status.ResourceRef

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}

func (src *Serving) convertOpenFuncAsyncTo(dst *v1alpha1.Serving) error {
	if src.Spec.OpenFuncAsync.Dapr != nil {
		dst.Spec.OpenFuncAsync.Dapr = &v1alpha1.Dapr{}
		dst.Spec.OpenFuncAsync.Dapr.Annotations = src.Spec.OpenFuncAsync.Dapr.Annotations

		if src.Spec.OpenFuncAsync.Dapr.Components != nil {
			dst.Spec.OpenFuncAsync.Dapr.Components = []v1alpha1.DaprComponent{}
			for name, component := range src.Spec.OpenFuncAsync.Dapr.Components {
				dc := v1alpha1.DaprComponent{
					Name:          name,
					ComponentSpec: *component,
				}
				dst.Spec.OpenFuncAsync.Dapr.Components = append(dst.Spec.OpenFuncAsync.Dapr.Components, dc)
			}
		}

		if src.Spec.OpenFuncAsync.Dapr.Inputs != nil {
			dst.Spec.OpenFuncAsync.Dapr.Inputs = []*v1alpha1.DaprIO{}
			for _, input := range src.Spec.OpenFuncAsync.Dapr.Inputs {
				in := v1alpha1.DaprIO{
					Name:   input.Name,
					Type:   input.Type,
					Topic:  input.Topic,
					Params: input.Params,
				}
				dst.Spec.OpenFuncAsync.Dapr.Inputs = append(dst.Spec.OpenFuncAsync.Dapr.Inputs, &in)
			}
		}

		if src.Spec.OpenFuncAsync.Dapr.Outputs != nil {
			dst.Spec.OpenFuncAsync.Dapr.Outputs = []*v1alpha1.DaprIO{}
			for _, output := range src.Spec.OpenFuncAsync.Dapr.Outputs {
				output.Params = map[string]string{}
				output.Params["operation"] = output.Operation
				out := v1alpha1.DaprIO{
					Name:   output.Name,
					Type:   output.Type,
					Topic:  output.Topic,
					Params: output.Params,
				}
				dst.Spec.OpenFuncAsync.Dapr.Outputs = append(dst.Spec.OpenFuncAsync.Dapr.Outputs, &out)
			}
		}

		if src.Spec.OpenFuncAsync.Keda != nil {
			dst.Spec.OpenFuncAsync.Keda = &v1alpha1.Keda{}
			if src.Spec.OpenFuncAsync.Keda.ScaledJob != nil {
				dst.Spec.OpenFuncAsync.Keda.ScaledJob = &v1alpha1.KedaScaledJob{}
				dst.Spec.OpenFuncAsync.Keda.ScaledJob.ScalingStrategy = src.Spec.OpenFuncAsync.Keda.ScaledJob.ScalingStrategy
				dst.Spec.OpenFuncAsync.Keda.ScaledJob.Triggers = src.Spec.OpenFuncAsync.Keda.ScaledJob.Triggers
				dst.Spec.OpenFuncAsync.Keda.ScaledJob.FailedJobsHistoryLimit = src.Spec.OpenFuncAsync.Keda.ScaledJob.FailedJobsHistoryLimit
				dst.Spec.OpenFuncAsync.Keda.ScaledJob.SuccessfulJobsHistoryLimit = src.Spec.OpenFuncAsync.Keda.ScaledJob.SuccessfulJobsHistoryLimit
				dst.Spec.OpenFuncAsync.Keda.ScaledJob.MaxReplicaCount = src.Spec.OpenFuncAsync.Keda.ScaledJob.MaxReplicaCount
				dst.Spec.OpenFuncAsync.Keda.ScaledJob.PollingInterval = src.Spec.OpenFuncAsync.Keda.ScaledJob.PollingInterval
				dst.Spec.OpenFuncAsync.Keda.ScaledJob.RestartPolicy = src.Spec.OpenFuncAsync.Keda.ScaledJob.RestartPolicy
			}

			if src.Spec.OpenFuncAsync.Keda.ScaledObject != nil {
				dst.Spec.OpenFuncAsync.Keda.ScaledObject = &v1alpha1.KedaScaledObject{}
				dst.Spec.OpenFuncAsync.Keda.ScaledObject.PollingInterval = src.Spec.OpenFuncAsync.Keda.ScaledObject.PollingInterval
				dst.Spec.OpenFuncAsync.Keda.ScaledObject.CooldownPeriod = src.Spec.OpenFuncAsync.Keda.ScaledObject.CooldownPeriod
				dst.Spec.OpenFuncAsync.Keda.ScaledObject.WorkloadType = src.Spec.OpenFuncAsync.Keda.ScaledObject.WorkloadType
				dst.Spec.OpenFuncAsync.Keda.ScaledObject.Advanced = src.Spec.OpenFuncAsync.Keda.ScaledObject.Advanced
				dst.Spec.OpenFuncAsync.Keda.ScaledObject.MinReplicaCount = src.Spec.OpenFuncAsync.Keda.ScaledObject.MinReplicaCount
				dst.Spec.OpenFuncAsync.Keda.ScaledObject.MaxReplicaCount = src.Spec.OpenFuncAsync.Keda.ScaledObject.MaxReplicaCount
				dst.Spec.OpenFuncAsync.Keda.ScaledObject.Triggers = src.Spec.OpenFuncAsync.Keda.ScaledObject.Triggers
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
func (dst *Serving) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha1.Serving)

	dst.ObjectMeta = src.ObjectMeta

	if src.Spec.OpenFuncAsync != nil {
		dst.Spec.OpenFuncAsync = &OpenFuncAsyncRuntime{}
		if err := dst.convertOpenFuncAsyncFrom(src); err != nil {
			return err
		}
	}

	dst.Spec.Version = src.Spec.Version
	dst.Spec.Params = src.Spec.Params
	dst.Spec.Image = src.Spec.Image
	dst.Spec.Port = src.Spec.Port
	dst.Spec.Runtime = (*Runtime)(src.Spec.Runtime)
	dst.Spec.Template = src.Spec.Template
	dst.Spec.ImageCredentials = src.Spec.ImageCredentials

	// Status
	dst.Status.State = src.Status.State
	dst.Status.Phase = src.Status.Phase
	dst.Status.ResourceRef = src.Status.ResourceRef

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}

func (dst *Serving) convertOpenFuncAsyncFrom(src *v1alpha1.Serving) error {
	if src.Spec.OpenFuncAsync.Dapr != nil {
		dst.Spec.OpenFuncAsync.Dapr = &Dapr{}
		dst.Spec.OpenFuncAsync.Dapr.Annotations = src.Spec.OpenFuncAsync.Dapr.Annotations

		if src.Spec.OpenFuncAsync.Dapr.Components != nil {
			dst.Spec.OpenFuncAsync.Dapr.Components = map[string]*componentsv1alpha1.ComponentSpec{}
			for _, component := range src.Spec.OpenFuncAsync.Dapr.Components {
				dst.Spec.OpenFuncAsync.Dapr.Components[component.Name] = &component.ComponentSpec
			}
		}

		if src.Spec.OpenFuncAsync.Dapr.Inputs != nil {
			dst.Spec.OpenFuncAsync.Dapr.Inputs = []*DaprIO{}
			for _, input := range src.Spec.OpenFuncAsync.Dapr.Inputs {
				in := DaprIO{
					Name:      fmt.Sprintf("%s-%s-%s", src.Namespace, src.Name, input.Name),
					Component: input.Name,
					Topic:     input.Topic,
					Params:    input.Params,
				}
				dst.Spec.OpenFuncAsync.Dapr.Inputs = append(dst.Spec.OpenFuncAsync.Dapr.Inputs, &in)
			}
		}

		if src.Spec.OpenFuncAsync.Dapr.Outputs != nil {
			dst.Spec.OpenFuncAsync.Dapr.Outputs = []*DaprIO{}
			for _, output := range src.Spec.OpenFuncAsync.Dapr.Outputs {
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
				dst.Spec.OpenFuncAsync.Dapr.Outputs = append(dst.Spec.OpenFuncAsync.Dapr.Outputs, &out)
			}
		}

		if src.Spec.OpenFuncAsync.Keda != nil {
			dst.Spec.OpenFuncAsync.Keda = &Keda{}
			if src.Spec.OpenFuncAsync.Keda.ScaledJob != nil {
				dst.Spec.OpenFuncAsync.Keda.ScaledJob = &KedaScaledJob{}
				dst.Spec.OpenFuncAsync.Keda.ScaledJob.ScalingStrategy = src.Spec.OpenFuncAsync.Keda.ScaledJob.ScalingStrategy
				dst.Spec.OpenFuncAsync.Keda.ScaledJob.Triggers = src.Spec.OpenFuncAsync.Keda.ScaledJob.Triggers
				dst.Spec.OpenFuncAsync.Keda.ScaledJob.FailedJobsHistoryLimit = src.Spec.OpenFuncAsync.Keda.ScaledJob.FailedJobsHistoryLimit
				dst.Spec.OpenFuncAsync.Keda.ScaledJob.SuccessfulJobsHistoryLimit = src.Spec.OpenFuncAsync.Keda.ScaledJob.SuccessfulJobsHistoryLimit
				dst.Spec.OpenFuncAsync.Keda.ScaledJob.MaxReplicaCount = src.Spec.OpenFuncAsync.Keda.ScaledJob.MaxReplicaCount
				dst.Spec.OpenFuncAsync.Keda.ScaledJob.PollingInterval = src.Spec.OpenFuncAsync.Keda.ScaledJob.PollingInterval
				dst.Spec.OpenFuncAsync.Keda.ScaledJob.RestartPolicy = src.Spec.OpenFuncAsync.Keda.ScaledJob.RestartPolicy
			}

			if src.Spec.OpenFuncAsync.Keda.ScaledObject != nil {
				dst.Spec.OpenFuncAsync.Keda.ScaledObject = &KedaScaledObject{}
				dst.Spec.OpenFuncAsync.Keda.ScaledObject.PollingInterval = src.Spec.OpenFuncAsync.Keda.ScaledObject.PollingInterval
				dst.Spec.OpenFuncAsync.Keda.ScaledObject.CooldownPeriod = src.Spec.OpenFuncAsync.Keda.ScaledObject.CooldownPeriod
				dst.Spec.OpenFuncAsync.Keda.ScaledObject.WorkloadType = src.Spec.OpenFuncAsync.Keda.ScaledObject.WorkloadType
				dst.Spec.OpenFuncAsync.Keda.ScaledObject.Advanced = src.Spec.OpenFuncAsync.Keda.ScaledObject.Advanced
				dst.Spec.OpenFuncAsync.Keda.ScaledObject.MinReplicaCount = src.Spec.OpenFuncAsync.Keda.ScaledObject.MinReplicaCount
				dst.Spec.OpenFuncAsync.Keda.ScaledObject.MaxReplicaCount = src.Spec.OpenFuncAsync.Keda.ScaledObject.MaxReplicaCount
				dst.Spec.OpenFuncAsync.Keda.ScaledObject.Triggers = src.Spec.OpenFuncAsync.Keda.ScaledObject.Triggers
			}
		}
	}
	return nil
}
