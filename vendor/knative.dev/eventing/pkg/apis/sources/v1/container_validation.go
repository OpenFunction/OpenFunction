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

	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
)

func (c *ContainerSource) Validate(ctx context.Context) *apis.FieldError {
	return c.Spec.Validate(ctx).ViaField("spec")
}

func (cs *ContainerSourceSpec) Validate(ctx context.Context) *apis.FieldError {
	var errs *apis.FieldError
	if fe := cs.Sink.Validate(ctx); fe != nil {
		errs = errs.Also(fe.ViaField("sink"))
	}

	// Validate there is at least a container
	if cs.Template.Spec.Containers == nil || len(cs.Template.Spec.Containers) == 0 {
		fe := apis.ErrMissingField("containers")
		errs = errs.Also(fe)
	} else {
		for i := range cs.Template.Spec.Containers {
			if ce := isValidContainer(&cs.Template.Spec.Containers[i]); ce != nil {
				errs = errs.Also(ce.ViaFieldIndex("containers", i))
			}
		}
	}
	return errs
}

func isValidContainer(c *corev1.Container) *apis.FieldError {
	var errs *apis.FieldError
	if c.Name == "" {
		errs = errs.Also(apis.ErrMissingField("name"))
	}
	if c.Image == "" {
		errs = errs.Also(apis.ErrMissingField("image"))
	}
	return errs
}
