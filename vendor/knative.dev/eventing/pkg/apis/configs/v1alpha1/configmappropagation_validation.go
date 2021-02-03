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

package v1alpha1

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"

	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmp"
)

// Validate the ConfigMapPropagation.
func (cmp *ConfigMapPropagation) Validate(ctx context.Context) *apis.FieldError {
	return cmp.Spec.Validate(ctx).ViaField("spec")
}

// Validate the ConfigMapPropagationSpec.
func (cmps *ConfigMapPropagationSpec) Validate(ctx context.Context) *apis.FieldError {
	var errs *apis.FieldError
	if cmps.OriginalNamespace == "" {
		fe := apis.ErrMissingField("originalNamespace")
		errs = errs.Also(fe)
	}

	if cmps.Selector != nil {
		for key, value := range cmps.Selector.MatchLabels {
			if err := validation.IsQualifiedName(key); len(err) != 0 {
				fe := &apis.FieldError{
					Message: fmt.Sprintf("Invalid selector matchLabels key: %v", key),
					Paths:   []string{"selector"},
					Details: strings.Join(err, "; "),
				}
				errs = errs.Also(fe)
			}
			if err := validation.IsValidLabelValue(value); len(err) != 0 {
				fe := &apis.FieldError{
					Message: fmt.Sprintf("Invalid selector matchLabels value: %v", value),
					Paths:   []string{"selector"},
					Details: strings.Join(err, "; "),
				}
				errs = errs.Also(fe)
			}
		}
		if cmps.Selector.MatchExpressions != nil {
			fe := &apis.FieldError{
				Message: "MatchExpressions isn't supported yet",
				Paths:   []string{"selector"},
			}
			errs = errs.Also(fe)
		}
	}
	return errs
}

// CheckImmutableFields checks that any immutable fields were not changed.
func (cmp *ConfigMapPropagation) CheckImmutableFields(ctx context.Context, original *ConfigMapPropagation) *apis.FieldError {
	if original == nil {
		return nil
	}

	if diff, err := kmp.ShortDiff(original.Spec, cmp.Spec); err != nil {
		return &apis.FieldError{
			Message: "Failed to diff ConfigMapPropagation",
			Paths:   []string{"spec"},
			Details: err.Error(),
		}
	} else if diff != "" {
		return &apis.FieldError{
			Message: "Immutable fields changed (-old +new)",
			Paths:   []string{"spec"},
			Details: diff,
		}
	}
	return nil
}
