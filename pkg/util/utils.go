package util

import "reflect"

func InterfaceIsNil(val interface{}) bool {

	if val == nil {
		return true
	}

	return reflect.ValueOf(val).IsNil()
}
