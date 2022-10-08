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
// ConvertTo converts this Function to the Hub version (v1beta1).
func (src *Function) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta1.Function)
	dst.ObjectMeta = src.ObjectMeta

	if src.Spec.Serving != nil {
		dst.Spec.Serving = &v1beta1.ServingImpl{}
		if err := src.convertServingTo(dst); err != nil {
			return err
		}
	}

	if src.Spec.Build != nil {
		dst.Spec.Build = &v1beta1.BuildImpl{}
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
		dst.Status.Build = &v1beta1.Condition{
			State:        src.Status.Build.State,
			ResourceRef:  src.Status.Build.ResourceRef,
			ResourceHash: src.Status.Build.ResourceHash,
		}
	}
	if src.Status.Serving != nil {
		dst.Status.Serving = &v1beta1.Condition{
			State:        src.Status.Serving.State,
			ResourceRef:  src.Status.Serving.ResourceRef,
			ResourceHash: src.Status.Serving.ResourceHash,
		}
	}

	if src.Status.URL != "" {
		addressType := v1beta1.InternalAddressType
		address := v1beta1.FunctionAddress{
			Type:  &addressType,
			Value: src.Status.URL,
		}
		dst.Status.Addresses = []v1beta1.FunctionAddress{address}
	}

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}

func (src *Function) convertBuildTo(dst *v1beta1.Function) error {
	dst.Spec.Build.Builder = src.Spec.Build.Builder
	dst.Spec.Build.BuilderCredentials = src.Spec.Build.BuilderCredentials
	dst.Spec.Build.Env = src.Spec.Build.Env
	dst.Spec.Build.Params = src.Spec.Build.Params
	dst.Spec.Build.Dockerfile = src.Spec.Build.Dockerfile

	if src.Spec.Build.SrcRepo != nil {
		dst.Spec.Build.SrcRepo = &v1beta1.GitRepo{}
		dst.Spec.Build.SrcRepo.Url = src.Spec.Build.SrcRepo.Url
		dst.Spec.Build.SrcRepo.SourceSubPath = src.Spec.Build.SrcRepo.SourceSubPath
		dst.Spec.Build.SrcRepo.Revision = src.Spec.Build.SrcRepo.Revision
		dst.Spec.Build.SrcRepo.Credentials = src.Spec.Build.SrcRepo.Credentials
	}

	if src.Spec.Build.Shipwright != nil {
		dst.Spec.Build.Shipwright = &v1beta1.ShipwrightEngine{}
		if src.Spec.Build.Shipwright.Strategy != nil {
			dst.Spec.Build.Shipwright.Strategy = &v1beta1.Strategy{}
			dst.Spec.Build.Shipwright.Strategy.Name = src.Spec.Build.Shipwright.Strategy.Name
			dst.Spec.Build.Shipwright.Strategy.Kind = src.Spec.Build.Shipwright.Strategy.Kind
		}
		dst.Spec.Build.Shipwright.Timeout = src.Spec.Build.Shipwright.Timeout
	}
	return nil
}

func (src *Function) convertServingTo(dst *v1beta1.Function) error {
	rtType := v1beta1.Knative
	if src.Spec.Serving.Runtime != nil && strings.EqualFold(string(*src.Spec.Serving.Runtime), string(OpenFuncAsync)) {
		rtType = v1beta1.Async
	}
	dst.Spec.Serving.Runtime = rtType

	if src.Spec.Serving.OpenFuncAsync != nil {
		if src.Spec.Serving.OpenFuncAsync.Dapr != nil {
			if src.Spec.Serving.OpenFuncAsync.Dapr.Components != nil {
				dst.Spec.Serving.Bindings = map[string]*componentsv1alpha1.ComponentSpec{}
				dst.Spec.Serving.Pubsub = map[string]*componentsv1alpha1.ComponentSpec{}
				for name, component := range src.Spec.Serving.OpenFuncAsync.Dapr.Components {
					c := component.DeepCopy()
					if strings.HasPrefix(c.Type, v1beta1.DaprBindings) {
						dst.Spec.Serving.Bindings[name] = c
					} else if strings.HasPrefix(c.Type, v1beta1.DaprPubsub) {
						dst.Spec.Serving.Pubsub[name] = c
					}
				}
			}

			if src.Spec.Serving.OpenFuncAsync.Dapr.Annotations != nil {
				dst.Spec.Serving.Annotations = src.Spec.Serving.OpenFuncAsync.Dapr.Annotations
			}

			if src.Spec.Serving.OpenFuncAsync.Dapr.Inputs != nil {
				dst.Spec.Serving.Inputs = []*v1beta1.DaprIO{}
				for _, input := range src.Spec.Serving.OpenFuncAsync.Dapr.Inputs {
					in := &v1beta1.DaprIO{
						Name:      input.Name,
						Component: input.Component,
						Topic:     input.Topic,
						Params:    input.Params,
						Operation: input.Operation,
					}
					dst.Spec.Serving.Inputs = append(dst.Spec.Serving.Inputs, in)
				}
			}

			if src.Spec.Serving.OpenFuncAsync.Dapr.Outputs != nil {
				dst.Spec.Serving.Outputs = []*v1beta1.DaprIO{}
				for _, output := range src.Spec.Serving.OpenFuncAsync.Dapr.Outputs {
					out := &v1beta1.DaprIO{
						Name:      output.Name,
						Component: output.Component,
						Topic:     output.Topic,
						Params:    output.Params,
						Operation: output.Operation,
					}
					dst.Spec.Serving.Outputs = append(dst.Spec.Serving.Outputs, out)
				}
			}
		}

		if src.Spec.Serving.OpenFuncAsync.Keda != nil {
			dst.Spec.Serving.ScaleOptions = &v1beta1.ScaleOptions{}
			dst.Spec.Serving.ScaleOptions.Keda = &v1beta1.KedaScaleOptions{}
			dst.Spec.Serving.Triggers = []v1beta1.Triggers{}
			if src.Spec.Serving.OpenFuncAsync.Keda.ScaledJob != nil {
				scaledJobKind := v1beta1.ScaledJob
				dst.Spec.Serving.ScaleOptions.Keda.ScaledJob = &v1beta1.KedaScaledJob{}
				sj := src.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.DeepCopy()
				dst.Spec.Serving.ScaleOptions.Keda.ScaledJob.ScalingStrategy = sj.ScalingStrategy
				dst.Spec.Serving.ScaleOptions.Keda.ScaledJob.FailedJobsHistoryLimit = sj.FailedJobsHistoryLimit
				dst.Spec.Serving.ScaleOptions.Keda.ScaledJob.SuccessfulJobsHistoryLimit = sj.SuccessfulJobsHistoryLimit
				dst.Spec.Serving.ScaleOptions.Keda.ScaledJob.MaxReplicaCount = sj.MaxReplicaCount
				dst.Spec.Serving.ScaleOptions.Keda.ScaledJob.PollingInterval = sj.PollingInterval
				dst.Spec.Serving.ScaleOptions.Keda.ScaledJob.RestartPolicy = sj.RestartPolicy
				for _, trigger := range sj.Triggers {
					t := trigger.DeepCopy()
					dst.Spec.Serving.Triggers = append(dst.Spec.Serving.Triggers, v1beta1.Triggers{
						ScaleTriggers: *t, TargetKind: &scaledJobKind,
					})
				}
			}

			if src.Spec.Serving.OpenFuncAsync.Keda.ScaledObject != nil {
				scaledObjectKind := v1beta1.ScaledObject
				dst.Spec.Serving.ScaleOptions.Keda.ScaledObject = &v1beta1.KedaScaledObject{}
				so := src.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.DeepCopy()
				dst.Spec.Serving.ScaleOptions.Keda.ScaledObject.PollingInterval = so.PollingInterval
				dst.Spec.Serving.ScaleOptions.Keda.ScaledObject.CooldownPeriod = so.CooldownPeriod
				dst.Spec.Serving.ScaleOptions.Keda.ScaledObject.WorkloadType = so.WorkloadType
				dst.Spec.Serving.ScaleOptions.Keda.ScaledObject.Advanced = so.Advanced
				dst.Spec.Serving.ScaleOptions.Keda.ScaledObject.MinReplicaCount = so.MinReplicaCount
				dst.Spec.Serving.ScaleOptions.Keda.ScaledObject.MaxReplicaCount = so.MaxReplicaCount
				for _, trigger := range so.Triggers {
					t := trigger.DeepCopy()
					dst.Spec.Serving.Triggers = append(dst.Spec.Serving.Triggers, v1beta1.Triggers{
						ScaleTriggers: *t, TargetKind: &scaledObjectKind,
					})
				}
			}
		}
	}

	if src.Spec.Serving.Annotations != nil {
		if dst.Spec.Serving.Annotations != nil {
			for k, v := range src.Spec.Serving.Annotations {
				dst.Spec.Serving.Annotations[k] = v
			}
		} else {
			dst.Spec.Serving.Annotations = src.Spec.Serving.Annotations
		}
	}
	return nil
}

/*
ConvertFrom is expected to modify its receiver to contain the converted object.
Most of the conversion is straightforward copying, except for converting our changed field.
*/

// ConvertFrom converts from the Hub version (v1beta1) to this version.
func (dst *Function) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta1.Function)
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
			State:                     src.Status.Serving.State,
			ResourceRef:               src.Status.Serving.ResourceRef,
			LastSuccessfulResourceRef: src.Status.Serving.ResourceRef,
			ResourceHash:              src.Status.Serving.ResourceHash,
		}
	}

	for _, address := range src.Status.Addresses {
		if *address.Type == v1beta1.InternalAddressType {
			dst.Status.URL = address.Value
			break
		}
	}

	// +kubebuilder:docs-gen:collapse=rote conversion
	return nil
}

func (dst *Function) convertBuildFrom(src *v1beta1.Function) error {
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
			dst.Spec.Build.Shipwright.Strategy = &Strategy{}
			dst.Spec.Build.Shipwright.Strategy.Name = src.Spec.Build.Shipwright.Strategy.Name
			dst.Spec.Build.Shipwright.Strategy.Kind = src.Spec.Build.Shipwright.Strategy.Kind
		}
		dst.Spec.Build.Shipwright.Timeout = src.Spec.Build.Shipwright.Timeout
	}
	return nil
}

func (dst *Function) convertServingFrom(src *v1beta1.Function) error {
	rt := Knative
	if strings.EqualFold(string(src.Spec.Serving.Runtime), string(v1beta1.Async)) {
		rt = OpenFuncAsync
	}
	dst.Spec.Serving.Runtime = &rt

	if dst.Spec.Serving.Annotations != nil {
		src.Spec.Serving.Annotations = dst.Spec.Service.Annotations
	}

	if src.Spec.Serving.Params != nil {
		dst.Spec.Serving.Params = src.Spec.Serving.Params
	}

	if src.Spec.Serving.Template != nil {
		dst.Spec.Serving.Template = src.Spec.Serving.Template
	}

	if src.Spec.Serving.Runtime == v1beta1.Async {
		dst.Spec.Serving.OpenFuncAsync = &OpenFuncAsyncRuntime{
			Dapr: &Dapr{},
		}
		dst.Spec.Serving.OpenFuncAsync.Dapr.Annotations = map[string]string{}
		dst.Spec.Serving.OpenFuncAsync.Dapr.Components = map[string]*componentsv1alpha1.ComponentSpec{}

		if src.Spec.Serving.ScaleOptions != nil && src.Spec.Serving.ScaleOptions.Keda != nil {
			dst.Spec.Serving.OpenFuncAsync.Keda = &Keda{}
		}

		if src.Spec.Serving.Bindings != nil {
			for name, component := range src.Spec.Serving.Bindings {
				c := component.DeepCopy()
				dst.Spec.Serving.OpenFuncAsync.Dapr.Components[name] = c
			}
		}

		if src.Spec.Serving.Pubsub != nil {
			for name, component := range src.Spec.Serving.Pubsub {
				c := component.DeepCopy()
				dst.Spec.Serving.OpenFuncAsync.Dapr.Components[name] = c
			}
		}
	}

	if src.Spec.Serving.Annotations != nil && src.Spec.Serving.Runtime == v1beta1.Async {
		for k, v := range src.Spec.Serving.Annotations {
			if strings.HasPrefix(k, "dapr.io") {
				dst.Spec.Serving.OpenFuncAsync.Dapr.Annotations[k] = v
			}
		}
	}

	if src.Spec.Serving.ScaleOptions != nil && src.Spec.Serving.ScaleOptions.Keda != nil {
		dst.Spec.Serving.OpenFuncAsync.Keda = &Keda{}

		if src.Spec.Serving.ScaleOptions.Keda.ScaledJob != nil {
			dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob = &KedaScaledJob{}
			dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.Triggers = []kedav1alpha1.ScaleTriggers{}
			sj := src.Spec.Serving.ScaleOptions.Keda.ScaledJob.DeepCopy()
			dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.ScalingStrategy = sj.ScalingStrategy
			dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.FailedJobsHistoryLimit = sj.FailedJobsHistoryLimit
			dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.SuccessfulJobsHistoryLimit = sj.SuccessfulJobsHistoryLimit
			dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.MaxReplicaCount = sj.MaxReplicaCount
			dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.PollingInterval = sj.PollingInterval
			dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.RestartPolicy = sj.RestartPolicy
			for _, trigger := range src.Spec.Serving.Triggers {
				t := trigger.DeepCopy()
				if t.TargetKind != nil && *t.TargetKind == v1beta1.ScaledJob {
					dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.Triggers = append(dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.Triggers, t.ScaleTriggers)
				}
			}

			// If no triggers are found, there is no need to set up the ScaledJob.
			if dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob.Triggers == nil {
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledJob = nil
			}
		}

		if src.Spec.Serving.ScaleOptions.Keda.ScaledObject != nil {
			dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject = &KedaScaledObject{}
			dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.Triggers = []kedav1alpha1.ScaleTriggers{}
			so := src.Spec.Serving.ScaleOptions.Keda.ScaledObject.DeepCopy()
			dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.PollingInterval = so.PollingInterval
			dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.CooldownPeriod = so.CooldownPeriod
			dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.WorkloadType = so.WorkloadType
			dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.Advanced = so.Advanced
			dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.MinReplicaCount = so.MinReplicaCount
			dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.MaxReplicaCount = so.MaxReplicaCount
			for _, trigger := range src.Spec.Serving.Triggers {
				t := trigger.DeepCopy()
				if t.TargetKind != nil && *t.TargetKind == v1beta1.ScaledJob {
					continue
				}
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.Triggers = append(dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.Triggers, t.ScaleTriggers)
			}

			// If no triggers are found, there is no need to set up the ScaledObject.
			if dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject.Triggers == nil {
				dst.Spec.Serving.OpenFuncAsync.Keda.ScaledObject = nil
			}
		}
	}

	if src.Spec.Serving.Inputs != nil && src.Spec.Serving.Runtime == v1beta1.Async {
		dst.Spec.Serving.OpenFuncAsync.Dapr.Inputs = []*DaprIO{}
		for _, input := range src.Spec.Serving.Inputs {
			in := &DaprIO{
				Name:      input.Name,
				Component: input.Component,
				Topic:     input.Topic,
				Params:    input.Params,
				Operation: input.Operation,
			}
			dst.Spec.Serving.OpenFuncAsync.Dapr.Inputs = append(dst.Spec.Serving.OpenFuncAsync.Dapr.Inputs, in)
		}
	}

	if src.Spec.Serving.Outputs != nil && src.Spec.Serving.Runtime == v1beta1.Async {
		dst.Spec.Serving.OpenFuncAsync.Dapr.Outputs = []*DaprIO{}
		for _, output := range src.Spec.Serving.Outputs {
			out := &DaprIO{
				Name:      output.Name,
				Component: output.Component,
				Topic:     output.Topic,
				Params:    output.Params,
				Operation: output.Operation,
			}
			dst.Spec.Serving.OpenFuncAsync.Dapr.Outputs = append(dst.Spec.Serving.OpenFuncAsync.Dapr.Outputs, out)
		}
	}
	return nil
}
