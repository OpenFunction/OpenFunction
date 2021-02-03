/*
Copyright 2020 The Knative Authors

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

package v1

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/tracker"
)

var sbCondSet = apis.NewLivingConditionSet(
	SinkBindingConditionSinkProvided,
)

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (*SinkBinding) GetConditionSet() apis.ConditionSet {
	return sbCondSet
}

// GetGroupVersionKind returns the GroupVersionKind.
func (*SinkBinding) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("SinkBinding")
}

// GetUntypedSpec implements apis.HasSpec
func (s *SinkBinding) GetUntypedSpec() interface{} {
	return s.Spec
}

// GetSubject implements psbinding.Bindable
func (sb *SinkBinding) GetSubject() tracker.Reference {
	return sb.Spec.Subject
}

// GetBindingStatus implements psbinding.Bindable
func (sb *SinkBinding) GetBindingStatus() duck.BindableStatus {
	return &sb.Status
}

// SetObservedGeneration implements psbinding.BindableStatus
func (sbs *SinkBindingStatus) SetObservedGeneration(gen int64) {
	sbs.ObservedGeneration = gen
}

// InitializeConditions populates the SinkBindingStatus's conditions field
// with all of its conditions configured to Unknown.
func (sbs *SinkBindingStatus) InitializeConditions() {
	sbCondSet.Manage(sbs).InitializeConditions()
}

// MarkBindingUnavailable marks the SinkBinding's Ready condition to False with
// the provided reason and message.
func (sbs *SinkBindingStatus) MarkBindingUnavailable(reason, message string) {
	sbCondSet.Manage(sbs).MarkFalse(SinkBindingConditionReady, reason, message)
}

// MarkBindingAvailable marks the SinkBinding's Ready condition to True.
func (sbs *SinkBindingStatus) MarkBindingAvailable() {
	sbCondSet.Manage(sbs).MarkTrue(SinkBindingConditionReady)
}

// MarkSink sets the condition that the source has a sink configured.
func (sbs *SinkBindingStatus) MarkSink(uri *apis.URL) {
	sbs.SinkURI = uri
	if uri != nil {
		sbCondSet.Manage(sbs).MarkTrue(SinkBindingConditionSinkProvided)
	} else {
		sbCondSet.Manage(sbs).MarkFalse(SinkBindingConditionSinkProvided, "SinkEmpty", "Sink has resolved to empty.%s", "")
	}
}

// Do implements psbinding.Bindable
func (sb *SinkBinding) Do(ctx context.Context, ps *duckv1.WithPod) {
	// First undo so that we can just unconditionally append below.
	sb.Undo(ctx, ps)

	resolver := GetURIResolver(ctx)
	if resolver == nil {
		logging.FromContext(ctx).Errorf("No Resolver associated with context for sink: %+v", sb)
		return
	}
	uri, err := resolver.URIFromDestinationV1(ctx, sb.Spec.Sink, sb)
	if err != nil {
		logging.FromContext(ctx).Errorw("URI could not be extracted from destination: ", zap.Error(err))
		return
	}
	sb.Status.MarkSink(uri)

	var ceOverrides string
	if sb.Spec.CloudEventOverrides != nil {
		if co, err := json.Marshal(sb.Spec.SourceSpec.CloudEventOverrides); err != nil {
			logging.FromContext(ctx).Errorw(fmt.Sprintf("Failed to marshal CloudEventOverrides into JSON for %+v", sb), zap.Error(err))
		} else if len(co) > 0 {
			ceOverrides = string(co)
		}
	}

	spec := ps.Spec.Template.Spec
	for i := range spec.InitContainers {
		spec.InitContainers[i].Env = append(spec.InitContainers[i].Env, corev1.EnvVar{
			Name:  "K_SINK",
			Value: uri.String(),
		})
		spec.InitContainers[i].Env = append(spec.InitContainers[i].Env, corev1.EnvVar{
			Name:  "K_CE_OVERRIDES",
			Value: ceOverrides,
		})
	}
	for i := range spec.Containers {
		spec.Containers[i].Env = append(spec.Containers[i].Env, corev1.EnvVar{
			Name:  "K_SINK",
			Value: uri.String(),
		})
		spec.Containers[i].Env = append(spec.Containers[i].Env, corev1.EnvVar{
			Name:  "K_CE_OVERRIDES",
			Value: ceOverrides,
		})
	}
}

func (sb *SinkBinding) Undo(ctx context.Context, ps *duckv1.WithPod) {
	spec := ps.Spec.Template.Spec
	for i, c := range spec.InitContainers {
		if len(c.Env) == 0 {
			continue
		}
		env := make([]corev1.EnvVar, 0, len(spec.InitContainers[i].Env))
		for j, ev := range c.Env {
			switch ev.Name {
			case "K_SINK", "K_CE_OVERRIDES":
				continue
			default:
				env = append(env, spec.InitContainers[i].Env[j])
			}
		}
		spec.InitContainers[i].Env = env
	}
	for i, c := range spec.Containers {
		if len(c.Env) == 0 {
			continue
		}
		env := make([]corev1.EnvVar, 0, len(spec.Containers[i].Env))
		for j, ev := range c.Env {
			switch ev.Name {
			case "K_SINK", "K_CE_OVERRIDES":
				continue
			default:
				env = append(env, spec.Containers[i].Env[j])
			}
		}
		spec.Containers[i].Env = env
	}
}
