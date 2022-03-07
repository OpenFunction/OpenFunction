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
	"strings"

	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/openfunction/apis/core/v1beta1"
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
// ConvertTo converts this Serving to the Hub version (v1beta1).
func (src *Serving) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta1.Serving)
	dst.ObjectMeta = src.ObjectMeta

	if src.Spec.OpenFuncAsync != nil {
		if err := src.convertServingTo(dst); err != nil {
			return err
		}
	}

	if src.Spec.Annotations != nil {
		if dst.Spec.Annotations != nil {
			for k, v := range src.Spec.Annotations {
				dst.Spec.Annotations[k] = v
			}
		} else {
			dst.Spec.Annotations = src.Spec.Annotations
		}
	}

	rtType := v1beta1.Knative
	if src.Spec.Runtime != nil && strings.EqualFold(string(*src.Spec.Runtime), string(OpenFuncAsync)) {
		rtType = v1beta1.Async
	}
	dst.Spec.Runtime = rtType

	dst.Spec.Version = src.Spec.Version
	dst.Spec.Params = src.Spec.Params
	dst.Spec.Image = src.Spec.Image
	dst.Spec.Port = src.Spec.Port
	dst.Spec.Template = src.Spec.Template
	dst.Spec.ImageCredentials = src.Spec.ImageCredentials

	// Status
	dst.Status.State = src.Status.State
	dst.Status.Phase = src.Status.Phase
	dst.Status.ResourceRef = src.Status.ResourceRef

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}

func (src *Serving) convertServingTo(dst *v1beta1.Serving) error {
	if src.Spec.OpenFuncAsync.Dapr != nil {
		if src.Spec.OpenFuncAsync.Dapr.Components != nil {
			dst.Spec.Bindings = map[string]*componentsv1alpha1.ComponentSpec{}
			dst.Spec.Pubsub = map[string]*componentsv1alpha1.ComponentSpec{}
			for name, component := range src.Spec.OpenFuncAsync.Dapr.Components {
				c := component.DeepCopy()
				if strings.HasPrefix(component.Type, v1beta1.DaprBindings) {
					dst.Spec.Bindings[name] = c
				} else if strings.HasPrefix(component.Type, v1beta1.DaprPubsub) {
					dst.Spec.Pubsub[name] = c
				}
			}
		}

		if src.Spec.OpenFuncAsync.Dapr.Annotations != nil {
			dst.Spec.Annotations = src.Spec.OpenFuncAsync.Dapr.Annotations
		}

		if src.Spec.OpenFuncAsync.Dapr.Inputs != nil {
			dst.Spec.Inputs = []*v1beta1.DaprIO{}
			for _, input := range src.Spec.OpenFuncAsync.Dapr.Inputs {
				in := &v1beta1.DaprIO{
					Name:      input.Name,
					Component: input.Component,
					Type:      input.Type,
					Topic:     input.Topic,
					Params:    input.Params,
					Operation: input.Operation,
				}
				dst.Spec.Inputs = append(dst.Spec.Inputs, in)
			}
		}

		if src.Spec.OpenFuncAsync.Dapr.Outputs != nil {
			dst.Spec.Outputs = []*v1beta1.DaprIO{}
			for _, output := range src.Spec.OpenFuncAsync.Dapr.Outputs {
				out := &v1beta1.DaprIO{
					Name:      output.Name,
					Component: output.Component,
					Type:      output.Type,
					Topic:     output.Topic,
					Params:    output.Params,
					Operation: output.Operation,
				}
				dst.Spec.Outputs = append(dst.Spec.Outputs, out)
			}
		}
	}

	if src.Spec.OpenFuncAsync.Keda != nil {
		dst.Spec.ScaleOptions = &v1beta1.ScaleOptions{}
		dst.Spec.ScaleOptions.Keda = &v1beta1.KedaScaleOptions{}
		dst.Spec.Triggers = []v1beta1.Triggers{}
		if src.Spec.OpenFuncAsync.Keda.ScaledJob != nil {
			scaledJobKind := v1beta1.ScaledJob
			dst.Spec.ScaleOptions.Keda.ScaledJob = &v1beta1.KedaScaledJob{}
			sj := src.Spec.OpenFuncAsync.Keda.ScaledJob.DeepCopy()
			dst.Spec.ScaleOptions.Keda.ScaledJob.ScalingStrategy = sj.ScalingStrategy
			dst.Spec.ScaleOptions.Keda.ScaledJob.FailedJobsHistoryLimit = sj.FailedJobsHistoryLimit
			dst.Spec.ScaleOptions.Keda.ScaledJob.SuccessfulJobsHistoryLimit = sj.SuccessfulJobsHistoryLimit
			dst.Spec.ScaleOptions.Keda.ScaledJob.MaxReplicaCount = sj.MaxReplicaCount
			dst.Spec.ScaleOptions.Keda.ScaledJob.PollingInterval = sj.PollingInterval
			dst.Spec.ScaleOptions.Keda.ScaledJob.RestartPolicy = sj.RestartPolicy
			for _, trigger := range sj.Triggers {
				t := trigger.DeepCopy()
				dst.Spec.Triggers = append(dst.Spec.Triggers, v1beta1.Triggers{
					ScaleTriggers: *t, TargetKind: &scaledJobKind,
				})
			}
		}

		if src.Spec.OpenFuncAsync.Keda.ScaledObject != nil {
			scaledObjectKind := v1beta1.ScaledObject
			dst.Spec.ScaleOptions.Keda.ScaledObject = &v1beta1.KedaScaledObject{}
			so := src.Spec.OpenFuncAsync.Keda.ScaledObject.DeepCopy()
			dst.Spec.ScaleOptions.Keda.ScaledObject.PollingInterval = so.PollingInterval
			dst.Spec.ScaleOptions.Keda.ScaledObject.CooldownPeriod = so.CooldownPeriod
			dst.Spec.ScaleOptions.Keda.ScaledObject.WorkloadType = so.WorkloadType
			dst.Spec.ScaleOptions.Keda.ScaledObject.Advanced = so.Advanced
			dst.Spec.ScaleOptions.Keda.ScaledObject.MinReplicaCount = so.MinReplicaCount
			dst.Spec.ScaleOptions.Keda.ScaledObject.MaxReplicaCount = so.MaxReplicaCount
			for _, trigger := range so.Triggers {
				t := trigger.DeepCopy()
				dst.Spec.Triggers = append(dst.Spec.Triggers, v1beta1.Triggers{
					ScaleTriggers: *t, TargetKind: &scaledObjectKind,
				})
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
	src := srcRaw.(*v1beta1.Serving)

	dst.ObjectMeta = src.ObjectMeta

	rt := Knative
	if strings.EqualFold(string(src.Spec.Runtime), string(v1beta1.Async)) {
		rt = OpenFuncAsync
	}
	dst.Spec.Runtime = &rt

	if dst.Spec.Annotations != nil {
		src.Spec.Annotations = dst.Spec.Annotations
	}

	if err := dst.convertServingFrom(src); err != nil {
		return err
	}

	dst.Spec.Version = src.Spec.Version
	dst.Spec.Params = src.Spec.Params
	dst.Spec.Image = src.Spec.Image
	dst.Spec.Port = src.Spec.Port
	dst.Spec.Template = src.Spec.Template
	dst.Spec.ImageCredentials = src.Spec.ImageCredentials

	// Status
	dst.Status.State = src.Status.State
	dst.Status.Phase = src.Status.Phase
	dst.Status.ResourceRef = src.Status.ResourceRef

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}

func (dst *Serving) convertServingFrom(src *v1beta1.Serving) error {
	if src.Spec.Runtime == v1beta1.Async {
		dst.Spec.OpenFuncAsync = &OpenFuncAsyncRuntime{
			Dapr: &Dapr{},
		}
		dst.Spec.OpenFuncAsync.Dapr.Annotations = map[string]string{}
		dst.Spec.OpenFuncAsync.Dapr.Components = map[string]*componentsv1alpha1.ComponentSpec{}

		if src.Spec.ScaleOptions != nil && src.Spec.ScaleOptions.Keda != nil {
			dst.Spec.OpenFuncAsync.Keda = &Keda{}
		}

		if src.Spec.Bindings != nil {
			for name, component := range src.Spec.Bindings {
				c := component.DeepCopy()
				dst.Spec.OpenFuncAsync.Dapr.Components[name] = c
			}
		}

		if src.Spec.Pubsub != nil {
			for name, component := range src.Spec.Pubsub {
				c := component.DeepCopy()
				dst.Spec.OpenFuncAsync.Dapr.Components[name] = c
			}
		}
	}

	if src.Spec.Annotations != nil {
		for k, v := range src.Spec.Annotations {
			if strings.HasPrefix(k, "dapr.io") {
				dst.Spec.OpenFuncAsync.Dapr.Annotations[k] = v
			}
		}
	}

	if src.Spec.Triggers != nil && src.Spec.ScaleOptions != nil && src.Spec.ScaleOptions.Keda != nil {
		dst.Spec.OpenFuncAsync.Keda = &Keda{}

		if src.Spec.ScaleOptions.Keda.ScaledJob != nil {
			dst.Spec.OpenFuncAsync.Keda.ScaledJob = &KedaScaledJob{}
			dst.Spec.OpenFuncAsync.Keda.ScaledJob.Triggers = []kedav1alpha1.ScaleTriggers{}
			sj := src.Spec.ScaleOptions.Keda.ScaledJob.DeepCopy()
			dst.Spec.OpenFuncAsync.Keda.ScaledJob.ScalingStrategy = sj.ScalingStrategy
			dst.Spec.OpenFuncAsync.Keda.ScaledJob.FailedJobsHistoryLimit = sj.FailedJobsHistoryLimit
			dst.Spec.OpenFuncAsync.Keda.ScaledJob.SuccessfulJobsHistoryLimit = sj.SuccessfulJobsHistoryLimit
			dst.Spec.OpenFuncAsync.Keda.ScaledJob.MaxReplicaCount = sj.MaxReplicaCount
			dst.Spec.OpenFuncAsync.Keda.ScaledJob.PollingInterval = sj.PollingInterval
			dst.Spec.OpenFuncAsync.Keda.ScaledJob.RestartPolicy = sj.RestartPolicy
			for _, trigger := range src.Spec.Triggers {
				t := trigger.DeepCopy()
				if t.TargetKind != nil && *t.TargetKind == v1beta1.ScaledJob {
					dst.Spec.OpenFuncAsync.Keda.ScaledJob.Triggers = append(dst.Spec.OpenFuncAsync.Keda.ScaledJob.Triggers, t.ScaleTriggers)
				}
			}

			// If no triggers are found, there is no need to set up the ScaledJob.
			if dst.Spec.OpenFuncAsync.Keda.ScaledJob.Triggers == nil {
				dst.Spec.OpenFuncAsync.Keda.ScaledJob = nil
			}
		}

		if src.Spec.ScaleOptions.Keda.ScaledObject != nil {
			dst.Spec.OpenFuncAsync.Keda.ScaledObject = &KedaScaledObject{}
			dst.Spec.OpenFuncAsync.Keda.ScaledObject.Triggers = []kedav1alpha1.ScaleTriggers{}
			so := src.Spec.ScaleOptions.Keda.ScaledObject.DeepCopy()
			dst.Spec.OpenFuncAsync.Keda.ScaledObject.PollingInterval = so.PollingInterval
			dst.Spec.OpenFuncAsync.Keda.ScaledObject.CooldownPeriod = so.CooldownPeriod
			dst.Spec.OpenFuncAsync.Keda.ScaledObject.WorkloadType = so.WorkloadType
			dst.Spec.OpenFuncAsync.Keda.ScaledObject.Advanced = so.Advanced
			dst.Spec.OpenFuncAsync.Keda.ScaledObject.MinReplicaCount = so.MinReplicaCount
			dst.Spec.OpenFuncAsync.Keda.ScaledObject.MaxReplicaCount = so.MaxReplicaCount
			for _, trigger := range src.Spec.Triggers {
				t := trigger.DeepCopy()
				if t.TargetKind != nil && *t.TargetKind == v1beta1.ScaledJob {
					continue
				}
				dst.Spec.OpenFuncAsync.Keda.ScaledObject.Triggers = append(dst.Spec.OpenFuncAsync.Keda.ScaledObject.Triggers, t.ScaleTriggers)
			}

			// If no triggers are found, there is no need to set up the ScaledObject.
			if dst.Spec.OpenFuncAsync.Keda.ScaledObject.Triggers == nil {
				dst.Spec.OpenFuncAsync.Keda.ScaledObject = nil
			}
		}
	}

	if src.Spec.Inputs != nil {
		dst.Spec.OpenFuncAsync.Dapr.Inputs = []*DaprIO{}
		for _, input := range src.Spec.Inputs {
			in := &DaprIO{
				Name:      input.Name,
				Component: input.Component,
				Type:      input.Type,
				Topic:     input.Topic,
				Params:    input.Params,
				Operation: input.Operation,
			}
			dst.Spec.OpenFuncAsync.Dapr.Inputs = append(dst.Spec.OpenFuncAsync.Dapr.Inputs, in)
		}
	}

	if src.Spec.Outputs != nil && src.Spec.Runtime == v1beta1.Async {
		dst.Spec.OpenFuncAsync.Dapr.Outputs = []*DaprIO{}
		for _, output := range src.Spec.Outputs {
			out := &DaprIO{
				Name:      output.Name,
				Component: output.Component,
				Type:      output.Type,
				Topic:     output.Topic,
				Params:    output.Params,
				Operation: output.Operation,
			}
			dst.Spec.OpenFuncAsync.Dapr.Outputs = append(dst.Spec.OpenFuncAsync.Dapr.Outputs, out)
		}
	}
	return nil
}
