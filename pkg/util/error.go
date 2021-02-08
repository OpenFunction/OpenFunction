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
	if errors.ReasonForError(err) == CRNotFound {
		return nil
	}
	return err
}
