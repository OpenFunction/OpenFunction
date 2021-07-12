package event

import (
	"context"
	"encoding/base64"
	"errors"

	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"github.com/go-logr/logr"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/openfunction/controllers/event/connector"
	"github.com/openfunction/pkg/util"
)

type SourceEnvConfig struct {
	SourceComponentName string      `json:"sourceComponentName"`
	SourceTopic         string      `json:"sourceTopic,omitempty"`
	BusConfigs          []BusConfig `json:"busConfigs,omitempty"`
	SinkComponentName   string      `json:"sinkComponentName,omitempty"`
	Port                string      `json:"port,omitempty"`
}

type BusConfig struct {
	BusComponentName string `json:"busComponentName"`
	BusTopic         string `json:"busTopic"`
}

type TriggerEnvConfig struct {
	BusComponentName string               `json:"busComponentName"`
	BusTopic         string               `json:"busTopic,omitempty"`
	Subscribers      []*SubscriberConfigs `json:"subscribers,omitempty"`
	Port             string               `json:"port,omitempty"`
}

type SubscriberConfigs struct {
	SinkComponentName           string `json:"sinkComponentName,omitempty"`
	DeadLetterSinkComponentName string `json:"deadLetterSinkComponentName,omitempty"`
	TopicName                   string `json:"topicName,omitempty"`
	DeadLetterTopicName         string `json:"deadLetterTopicName,omitempty"`
}

func (e *SourceEnvConfig) EncodeEnvConfig() (string, error) {
	return encodeEnvConfig(e)
}

func (e *TriggerEnvConfig) EncodeEnvConfig() (string, error) {
	return encodeEnvConfig(e)
}

func encodeEnvConfig(config interface{}) (string, error) {
	envConfigBytes, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	envConfigEncode := base64.StdEncoding.EncodeToString(envConfigBytes)
	return envConfigEncode, nil
}

func (e *SourceEnvConfig) DecodeEnvConfig(encodedConfig string) (*SourceEnvConfig, error) {
	var config *SourceEnvConfig
	envConifgSpec, err := decodeEnvConfig(encodedConfig)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(envConifgSpec, &config); err != nil {
		return nil, err
	}
	return config, nil
}

func (e *TriggerEnvConfig) DecodeEnvConfig(encodedConfig string) (*TriggerEnvConfig, error) {
	var config *TriggerEnvConfig
	envConifgSpec, err := decodeEnvConfig(encodedConfig)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(envConifgSpec, &config); err != nil {
		return nil, err
	}
	return config, nil
}

func decodeEnvConfig(encodedConfig string) ([]byte, error) {
	if len(encodedConfig) > 0 {
		envConifgSpec, err := base64.StdEncoding.DecodeString(encodedConfig)
		if err != nil {
			return nil, err
		}
		return envConifgSpec, nil
	}
	return nil, errors.New("string length is zero")
}

func CreateOrUpdateDaprComponent(client client.Client, schema *runtime.Scheme, log logr.Logger, connector *connector.Connector, object v1.Object) error {
	ctx := context.Background()
	// Check if component already exists
	var component componentsv1alpha1.Component
	if err := client.Get(ctx, types.NamespacedName{Namespace: connector.Component.Namespace, Name: connector.Component.Name}, &component); util.IsNotFound(err) {
		log.Info("Need create dapr component", "namespace", object.GetNamespace(), "name", object.GetName())
	}

	// TODO: Determine if the dapr component in the cluster is the same as the one in the request
	component = connector.Component

	if err := client.Delete(ctx, &component); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to delete dapr component", "name", component.Name, "namespace", component.Namespace)
		return err
	}

	if err := mutateDaprComponent(schema, &component, object)(); err != nil {
		log.Error(err, "Failed to mutate dapr component", "name", component.Name, "namespace", component.Namespace)
		return err
	}

	if err := client.Create(ctx, &component); err != nil {
		log.Error(err, "Failed to create dapr component", "name", component.Name, "namespace", component.Namespace)
		return err
	}

	log.V(1).Info("Create dapr component", "name", component.Name, "namespace", component.Namespace)
	return nil
}

func mutateDaprComponent(schema *runtime.Scheme, component *componentsv1alpha1.Component, object v1.Object) controllerutil.MutateFn {
	return func() error {
		component.SetOwnerReferences(nil)
		return ctrl.SetControllerReference(object, component, schema)
	}
}
