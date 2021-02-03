/*
 * Copyright 2020 The Knative Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1beta1

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmp"

	corev1 "k8s.io/api/core/v1"
)

var (
	// Only allow lowercase alphanumeric, starting with letters.
	validAttributeName = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
)

// Validate the Trigger.
func (t *Trigger) Validate(ctx context.Context) *apis.FieldError {
	errs := t.Spec.Validate(ctx).ViaField("spec")
	errs = t.validateAnnotation(errs, DependencyAnnotation, t.validateDependencyAnnotation)
	errs = t.validateAnnotation(errs, InjectionAnnotation, t.validateInjectionAnnotation)
	return errs
}

// Validate the TriggerSpec.
func (ts *TriggerSpec) Validate(ctx context.Context) *apis.FieldError {
	var errs *apis.FieldError
	if ts.Broker == "" {
		fe := apis.ErrMissingField("broker")
		errs = errs.Also(fe)
	}

	if ts.Filter != nil {
		for attr := range map[string]string(ts.Filter.Attributes) {
			if !validAttributeName.MatchString(attr) {
				fe := &apis.FieldError{
					Message: fmt.Sprintf("Invalid attribute name: %q", attr),
					Paths:   []string{"filter.attributes"},
				}
				errs = errs.Also(fe)
			}
		}
	}

	if fe := ts.Subscriber.Validate(ctx); fe != nil {
		errs = errs.Also(fe.ViaField("subscriber"))
	}

	return errs
}

// CheckImmutableFields checks that any immutable fields were not changed.
func (t *Trigger) CheckImmutableFields(ctx context.Context, original *Trigger) *apis.FieldError {
	if original == nil {
		return nil
	}

	if diff, err := kmp.ShortDiff(original.Spec.Broker, t.Spec.Broker); err != nil {
		return &apis.FieldError{
			Message: "Failed to diff Trigger",
			Paths:   []string{"spec"},
			Details: err.Error(),
		}
	} else if diff != "" {
		return &apis.FieldError{
			Message: "Immutable fields changed (-old +new)",
			Paths:   []string{"spec", "broker"},
			Details: diff,
		}
	}
	return nil
}

func GetObjRefFromDependencyAnnotation(dependencyAnnotation string) (corev1.ObjectReference, error) {
	var objectRef corev1.ObjectReference
	if err := json.Unmarshal([]byte(dependencyAnnotation), &objectRef); err != nil {
		return objectRef, err
	}
	return objectRef, nil
}

func (t *Trigger) validateAnnotation(errs *apis.FieldError, annotation string, function func(string) *apis.FieldError) *apis.FieldError {
	if annotationValue, ok := t.GetAnnotations()[annotation]; ok {
		annotationPrefix := fmt.Sprintf("metadata.annotations[%s]", annotation)
		errs = errs.Also(function(annotationValue).ViaField(annotationPrefix))
	}
	return errs
}

func (t *Trigger) validateDependencyAnnotation(dependencyAnnotation string) *apis.FieldError {
	depObjRef, err := GetObjRefFromDependencyAnnotation(dependencyAnnotation)
	if err != nil {
		return &apis.FieldError{
			Message: fmt.Sprintf("The provided annotation was not a corev1.ObjectReference: %q", dependencyAnnotation),
			Details: err.Error(),
			Paths:   []string{""},
		}
	}
	var errs *apis.FieldError
	if depObjRef.Namespace != "" && depObjRef.Namespace != t.GetNamespace() {
		fe := &apis.FieldError{
			Message: fmt.Sprintf("Namespace must be empty or equal to the trigger namespace %q", t.GetNamespace()),
			Paths:   []string{"namespace"},
		}
		errs = errs.Also(fe)
	}
	if depObjRef.Kind == "" {
		fe := apis.ErrMissingField("kind")
		errs = errs.Also(fe)
	}
	if depObjRef.Name == "" {
		fe := apis.ErrMissingField("name")
		errs = errs.Also(fe)
	}
	if depObjRef.APIVersion == "" {
		fe := apis.ErrMissingField("apiVersion")
		errs = errs.Also(fe)
	}
	return errs
}

func (t *Trigger) validateInjectionAnnotation(injectionAnnotation string) *apis.FieldError {
	if injectionAnnotation != "enabled" && injectionAnnotation != "disabled" {
		return &apis.FieldError{
			Message: fmt.Sprintf(`The provided injection annotation value can only be "enabled" or "disabled", not %q`, injectionAnnotation),
			Paths:   []string{""},
		}
	}
	return nil
}
