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
