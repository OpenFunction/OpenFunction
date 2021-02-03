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
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/apis"
)

const (
	// ReferenceMode produces payloads of ObjectReference
	ReferenceMode = "Reference"
	// ResourceMode produces payloads of ResourceEvent
	ResourceMode = "Resource"
)

func (c *ApiServerSource) Validate(ctx context.Context) *apis.FieldError {
	return c.Spec.Validate(ctx).ViaField("spec")
}

func (cs *ApiServerSourceSpec) Validate(ctx context.Context) *apis.FieldError {
	var errs *apis.FieldError

	// Validate mode, if can be empty or set as certain value
	switch cs.EventMode {
	case ReferenceMode, ResourceMode:
	// EventMode is valid.
	default:
		errs = errs.Also(apis.ErrInvalidValue(cs.EventMode, "mode"))
	}

	// Validate sink
	errs = errs.Also(cs.Sink.Validate(ctx).ViaField("sink"))

	if len(cs.Resources) == 0 {
		errs = errs.Also(apis.ErrMissingField("resources"))
	}
	for i, res := range cs.Resources {
		_, err := schema.ParseGroupVersion(res.APIVersion)
		if err != nil {
			errs = errs.Also(apis.ErrInvalidValue(res.APIVersion, "apiVersion").ViaFieldIndex("resources", i))
		}
		if strings.TrimSpace(res.Kind) == "" {
			errs = errs.Also(apis.ErrMissingField("kind").ViaFieldIndex("resources", i))
		}
	}

	if cs.ResourceOwner != nil {
		_, err := schema.ParseGroupVersion(cs.ResourceOwner.APIVersion)
		if err != nil {
			errs = errs.Also(apis.ErrInvalidValue(cs.ResourceOwner.APIVersion, "apiVersion").ViaField("owner"))
		}
		if strings.TrimSpace(cs.ResourceOwner.Kind) == "" {
			errs = errs.Also(apis.ErrMissingField("kind").ViaField("owner"))
		}
	}

	return errs
}
