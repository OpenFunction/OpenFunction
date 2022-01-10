/*
Copyright 2022 The OpenFunction Authors.

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

package util

import (
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CRNotFound metav1.StatusReason = "not found"
)

// IgnoreNotFound returns nil on 'not found' errors.
// All other values that are not 'not found' errors or nil are returned unmodified.
func IgnoreNotFound(err error) error {

	if err == nil {
		return nil
	}

	if errors.ReasonForError(err) == CRNotFound || errors.ReasonForError(err) == metav1.StatusReasonNotFound {
		return nil
	}

	return err
}

func IsNotFound(err error) bool {

	if err == nil {
		return false
	}

	return errors.IsNotFound(err)
}
