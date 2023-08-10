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

package redis

import (
	"sync"

	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"github.com/go-logr/logr"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ofcore "github.com/openfunction/apis/core/v1beta1"
	ofevent "github.com/openfunction/apis/events/v1alpha1"
	"github.com/openfunction/pkg/event"
)

const (
	ComponentType    = "bindings.redis"
	ComponentVersion = "v1"
)

type EventSource struct {
	mu       sync.Mutex
	log      logr.Logger
	Spec     *ofevent.RedisSpec
	Metadata map[string]interface{}
}

func NewRedisEventSource(log logr.Logger, spec *ofevent.RedisSpec) event.OpenFunctionEventSource {
	es := &EventSource{}

	es.log = log
	es.log.WithName("RedisEventSource")

	es.Spec = spec
	es.init()
	return es
}

func (es *EventSource) init() {
	m := map[string]interface{}{}

	// handle mandatory parameters
	m["redisHost"] = es.Spec.RedisHost
	m["redisPassword"] = es.Spec.RedisPassword

	// handle optional parameters
	if es.Spec.EnableTLS != nil {
		m["enableTLS"] = *es.Spec.EnableTLS
	}
	if es.Spec.Failover != nil {
		m["failover"] = *es.Spec.Failover
	}
	if es.Spec.SentinelMasterName != nil {
		m["sentinelMasterName"] = *es.Spec.SentinelMasterName
	}
	if es.Spec.RedeliverInterval != nil {
		m["redeliverInterval"] = *es.Spec.RedeliverInterval
	}
	if es.Spec.ProcessingTimeout != nil {
		m["processingTimeout"] = *es.Spec.ProcessingTimeout
	}
	if es.Spec.RedisType != nil {
		m["redisType"] = *es.Spec.RedisType
	}
	if es.Spec.RedisDB != nil {
		m["redisDB"] = *es.Spec.RedisDB
	}
	if es.Spec.RedisMaxRetries != nil {
		m["redisMaxRetries"] = *es.Spec.RedisMaxRetries
	}
	if es.Spec.RedisMinRetryInterval != nil {
		m["redisMinRetryInterval"] = *es.Spec.RedisMinRetryInterval
	}
	if es.Spec.RedisMaxRetryInterval != nil {
		m["redisMaxRetryInterval"] = *es.Spec.RedisMaxRetryInterval
	}
	if es.Spec.DialTimeout != nil {
		m["dialTimeout"] = *es.Spec.DialTimeout
	}
	if es.Spec.ReadTimeout != nil {
		m["readTimeout"] = *es.Spec.ReadTimeout
	}
	if es.Spec.WriteTimeout != nil {
		m["writeTimeout"] = *es.Spec.WriteTimeout
	}
	if es.Spec.PoolSize != nil {
		m["poolSize"] = *es.Spec.PoolSize
	}
	if es.Spec.PoolTimeout != nil {
		m["poolTimeout"] = *es.Spec.PoolTimeout
	}
	if es.Spec.MaxConnAge != nil {
		m["maxConnAge"] = *es.Spec.MaxConnAge
	}
	if es.Spec.MinIdleConns != nil {
		m["minIdleConns"] = *es.Spec.MinIdleConns
	}
	if es.Spec.IdleCheckFrequency != nil {
		m["idleCheckFrequency"] = *es.Spec.IdleCheckFrequency
	}
	if es.Spec.IdleTimeout != nil {
		m["idleTimeout"] = *es.Spec.IdleTimeout
	}

	es.Metadata = m
}

func (es *EventSource) SetMetadata(key string, value interface{}) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.Metadata[key] = value
}

func (es *EventSource) GetMetadata() map[string]interface{} {
	es.mu.Lock()
	defer es.mu.Unlock()
	return es.Metadata
}

func (es *EventSource) GenComponent(namespace string, name string) (*componentsv1alpha1.Component, error) {
	component := &componentsv1alpha1.Component{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	component.Spec.Type = ComponentType
	component.Spec.Version = ComponentVersion

	metadataItems, err := event.ConvertMetadata(es.GetMetadata())
	if err != nil {
		es.log.Error(err, "failed to generate component", "namespace", namespace, "name", name)
		return nil, err
	}
	component.Spec.Metadata = metadataItems
	return component, nil
}

func (es *EventSource) GenScaleOptions() (*ofcore.KedaScaledObject, *kedav1alpha1.ScaleTriggers) {
	return nil, nil
}
