package util

import (
	"reflect"
)

func InterfaceIsNil(val interface{}) bool {

	if val == nil {
		return true
	}

	return reflect.ValueOf(val).IsNil()
}

func AppendLabels(src, dest map[string]string) map[string]string {
	if src == nil || len(src) == 0 {
		return dest
	}

	if dest == nil {
		dest = make(map[string]string)
	}

	for k, v := range src {
		dest[k] = v
	}

	return dest
}
