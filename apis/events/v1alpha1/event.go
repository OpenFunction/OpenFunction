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

package v1alpha1

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ErrorConfiguration               ConditionReason = "ErrorConfiguration"
	ErrorToFindTriggerSubscribers    ConditionReason = "ErrorToFindTriggerSubscribers"
	ErrorToFindExistEventBus         ConditionReason = "ErrorToFindExistEventBus"
	ErrorGenerateComponent           ConditionReason = "ErrorGenerateComponent"
	ErrorGenerateScaledObject        ConditionReason = "ErrorGenerateScaledObject"
	ErrorCreatingEventSourceFunction ConditionReason = "ErrorCreatingEventSourceFunction"
	ErrorCreatingTriggerFunction     ConditionReason = "ErrorCreatingTriggerFunction"
	ErrorCreatingEventSource         ConditionReason = "ErrorCreatingEventSource"
	ErrorCreatingTrigger             ConditionReason = "ErrorCreatingTrigger"
	EventSourceFunctionCreated       ConditionReason = "EventSourceFunctionCreated"
	TriggerFunctionCreated           ConditionReason = "TriggerFunctionCreated"
	PendingCreation                  ConditionReason = "PendingCreation"
	EventSourceIsReady               ConditionReason = "EventSourceIsReady"
	TriggerIsReady                   ConditionReason = "TriggerIsReady"
)

const (
	// Created indicates the resource has been created
	Created CreationStatus = "Created"
	// Terminated indicates the resource has been terminated
	Terminated CreationStatus = "Terminated"
	// Error indicates the resource had an error
	Error CreationStatus = "Error"
	// Pending indicates the resource hasn't been created
	Pending CreationStatus = "Pending"
	// Terminating indicates that the resource is marked for deletion but hasn't
	// been deleted yet
	Terminating CreationStatus = "Terminating"
	// Unknown indicates the status is unavailable
	Unknown CreationStatus = "Unknown"
	// Ready indicates the object is fully created
	Ready CreationStatus = "Ready"
)

// CreationStatus describes the creation status
// of the scaler's additional resources such as Services, Ingresses and Deployments
// +kubebuilder:validation:Enum=Created;Error;Pending;Unknown;Terminating;Terminated;Ready
type CreationStatus string

// ConditionReason describes the reason why the condition transitioned
// +kubebuilder:validation:Enum=EventSourceFunctionCreated;ErrorCreatingEventSource;ErrorCreatingEventSourceFunction;EventSourceIsReady;ErrorConfiguration;ErrorToFindExistEventBus;ErrorGenerateComponent;ErrorGenerateScaledObject;PendingCreation;ErrorToFindTriggerSubscribers;ErrorCreatingTrigger;TriggerIsReady;ErrorCreatingTriggerFunction;TriggerFunctionCreated
type ConditionReason string

type Condition struct {
	// Timestamp of the condition
	// +optional
	Timestamp string `json:"timestamp" description:"Timestamp of this condition"`
	// Type of condition
	// +required
	Type CreationStatus `json:"type" description:"type of status condition"`
	// Status of the condition, one of True, False, Unknown.
	// +required
	Status metav1.ConditionStatus `json:"status" description:"status of the condition, one of True, False, Unknown"`
	// The reason for the condition's last transition.
	// +optional
	Reason ConditionReason `json:"reason,omitempty" description:"one-word CamelCase reason for the condition's last transition"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty" description:"human-readable message indicating details about last transition"`
}

// SaveStatus will trigger an object update to save the current status conditions
func (es *EventSource) SaveStatus(ctx context.Context, logger logr.Logger, cl client.Client) {
	logger.Info("Updating status on EventSource", "resource version", es.ResourceVersion)

	err := cl.Status().Update(ctx, es)
	if err != nil {
		logger.Error(err, "failed to update status on EventSource", "EventSource", es)
	} else {
		logger.Info("Updated status on EventSource", "resource version", es.ResourceVersion)
	}
}

// AddCondition adds a new condition to the resource
func (es *EventSource) AddCondition(condition Condition) *EventSource {
	es.Status.Conditions = append(es.Status.Conditions, condition)
	return es
}

func (t *Trigger) AddCondition(condition Condition) *Trigger {
	t.Status.Conditions = append(t.Status.Conditions, condition)
	return t
}

// SaveStatus will trigger an object update to save the current status conditions
func (t *Trigger) SaveStatus(ctx context.Context, logger logr.Logger, cl client.Client) {
	saveStatus(ctx, logger, cl, "Trigger", t)
}

func saveStatus(ctx context.Context, logger logr.Logger, cl client.Client, kind string, object client.Object) {
	logger.Info(fmt.Sprintf("Updating status on %s", kind), "resource version", object.GetResourceVersion())

	err := cl.Status().Update(ctx, object)
	if err != nil {
		logger.Error(err, fmt.Sprintf("failed to update status on %s", kind), kind, object)
	} else {
		logger.Info(fmt.Sprintf("Updated status on %s", kind), "resource version", object.GetResourceVersion())
	}
}

func CreateCondition(
	condType CreationStatus,
	status metav1.ConditionStatus,
	reason ConditionReason,
) *Condition {
	cond := Condition{
		Timestamp: time.Now().Format(time.RFC3339),
		Type:      condType,
		Status:    status,
		Reason:    reason,
	}
	return &cond
}

// SetMessage sets the optional reason for the condition
func (c *Condition) SetMessage(message string) *Condition {
	c.Message = message
	return c
}

type GenericScaleOption struct {
	PollingInterval *int32                            `json:"pollingInterval,omitempty"`
	CooldownPeriod  *int32                            `json:"cooldownPeriod,omitempty"`
	MinReplicaCount *int32                            `json:"minReplicaCount,omitempty"`
	MaxReplicaCount *int32                            `json:"maxReplicaCount,omitempty"`
	Advanced        *kedav1alpha1.AdvancedConfig      `json:"advanced,omitempty"`
	Metadata        map[string]string                 `json:"metadata,omitempty"`
	AuthRef         *kedav1alpha1.ScaledObjectAuthRef `json:"authRef,omitempty"`
}

type RedisSpec struct {
	RedisHost             string  `json:"redisHost"`
	RedisPassword         string  `json:"redisPassword"`
	EnableTLS             *bool   `json:"enableTLS,omitempty"`
	Failover              *bool   `json:"failover,omitempty"`
	SentinelMasterName    *string `json:"sentinelMasterName,omitempty"`
	RedeliverInterval     *string `json:"redeliverInterval,omitempty"`
	ProcessingTimeout     *string `json:"processingTimeout,omitempty"`
	RedisType             *string `json:"redisType,omitempty"`
	RedisDB               *int64  `json:"redisDB,omitempty"`
	RedisMaxRetries       *int64  `json:"redisMaxRetries,omitempty"`
	RedisMinRetryInterval *string `json:"redisMinRetryInterval,omitempty"`
	RedisMaxRetryInterval *string `json:"redisMaxRetryInterval,omitempty"`
	DialTimeout           *string `json:"dialTimeout,omitempty"`
	ReadTimeout           *string `json:"readTimeout,omitempty"`
	WriteTimeout          *string `json:"writeTimeout,omitempty"`
	PoolSize              *int64  `json:"poolSize,omitempty"`
	PoolTimeout           *string `json:"poolTimeout,omitempty"`
	MaxConnAge            *string `json:"maxConnAge,omitempty"`
	MinIdleConns          *int64  `json:"minIdleConns,omitempty"`
	IdleCheckFrequency    *string `json:"idleCheckFrequency,omitempty"`
	IdleTimeout           *string `json:"idleTimeout,omitempty"`
}

type KafkaSpec struct {
	Brokers         string            `json:"brokers"`
	AuthRequired    bool              `json:"authRequired"`
	Topic           string            `json:"topic,omitempty"`
	SaslUsername    *string           `json:"saslUsername,omitempty"`
	SaslPassword    *string           `json:"saslPassword,omitempty"`
	MaxMessageBytes *int64            `json:"maxMessageBytes,omitempty"`
	ScaleOption     *KafkaScaleOption `json:"scaleOption,omitempty"`
}

type KafkaScaleOption struct {
	*GenericScaleOption `json:",inline"`
	ConsumerGroup       string `json:"consumerGroup,omitempty"`
	Topic               string `json:"topic,omitempty"`
	LagThreshold        string `json:"lagThreshold"`
}

type CronSpec struct {
	Schedule string `json:"schedule"`
}

type MQTTSpec struct {
	Url          string  `json:"url"`
	Topic        string  `json:"topic"`
	ConsumerID   *string `json:"consumerID,omitempty"`
	Qos          *int64  `json:"qos,omitempty"`
	Retain       *bool   `json:"retain,omitempty"`
	CleanSession *bool   `json:"cleanSession,omitempty"`
	CaCert       *string `json:"caCert,omitempty"`
	ClientCert   *string `json:"clientCert,omitempty"`
	ClientKey    *string `json:"clientKey,omitempty"`
}

type NatsStreamingSpec struct {
	NatsURL                 string                    `json:"natsURL"`
	NatsStreamingClusterID  string                    `json:"natsStreamingClusterID"`
	SubscriptionType        string                    `json:"subscriptionType"`
	AckWaitTime             *string                   `json:"ackWaitTime,omitempty"`
	MaxInFlight             *int64                    `json:"maxInFlight,omitempty"`
	DurableSubscriptionName string                    `json:"durableSubscriptionName"`
	DeliverNew              *bool                     `json:"deliverNew,omitempty"`
	StartAtSequence         *int64                    `json:"startAtSequence,omitempty"`
	StartWithLastReceived   *bool                     `json:"startWithLastReceived,omitempty"`
	DeliverAll              *bool                     `json:"deliverAll,omitempty"`
	StartAtTimeDelta        *string                   `json:"startAtTimeDelta,omitempty"`
	StartAtTime             *string                   `json:"startAtTime,omitempty"`
	StartAtTimeFormat       *string                   `json:"startAtTimeFormat,omitempty"`
	ConsumerID              *string                   `json:"consumerID,omitempty"`
	ScaleOption             *NatsStreamingScaleOption `json:"scaleOption,omitempty"`
}

type NatsStreamingScaleOption struct {
	*GenericScaleOption          `json:",inline"`
	NatsServerMonitoringEndpoint string `json:"natsServerMonitoringEndpoint"`
	QueueGroup                   string `json:"queueGroup,omitempty"`
	DurableName                  string `json:"durableName,omitempty"`
	Subject                      string `json:"subject,omitempty"`
	LagThreshold                 string `json:"lagThreshold"`
}
