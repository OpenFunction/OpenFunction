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
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openfunction/pkg/constants"
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

func GetConfigOrDefault(cm map[string]string, key string, defaultVal string) string {
	if cm == nil {
		return defaultVal
	}
	if val, ok := cm[key]; ok {
		return val
	}
	return defaultVal
}

func GetDefaultConfig(ctx context.Context, c client.Client, log logr.Logger) map[string]string {
	log.WithName("Config").WithValues("ConfigMap", constants.DefaultConfigMapName)

	cm := &corev1.ConfigMap{}

	if err := c.Get(ctx, client.ObjectKey{
		Namespace: constants.DefaultControllerNamespace,
		Name:      constants.DefaultConfigMapName,
	}, cm); err == nil {
		if cm != nil {
			return cm.Data
		}
	}

	log.Info(fmt.Sprintf("Unable to get the default global configuration from ConfigMap %s in namespace %s",
		constants.DefaultConfigMapName, constants.DefaultControllerNamespace))
	return nil
}
