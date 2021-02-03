/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"encoding/json"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

const (
	// ChannelDefaultsConfigName is the name of config map for the default
	// configs that channels should use.
	ChannelDefaultsConfigName = "default-ch-webhook"

	// ChannelDefaulterKey is the key in the ConfigMap to get the name of the default
	// Channel CRD.
	ChannelDefaulterKey = "default-ch-config"
)

// NewChannelDefaultsConfigFromMap creates a Defaults from the supplied Map
func NewChannelDefaultsConfigFromMap(data map[string]string) (*ChannelDefaults, error) {
	nc := &ChannelDefaults{}

	// Parse out the Broker Configuration Cluster default section
	value, present := data[ChannelDefaulterKey]
	if !present || value == "" {
		return nil, fmt.Errorf("ConfigMap is missing (or empty) key: %q : %v", ChannelDefaulterKey, data)
	}
	if err := parseEntry(value, nc); err != nil {
		return nil, fmt.Errorf("Failed to parse the entry: %s", err)
	}
	return nc, nil
}

func parseEntry(entry string, out interface{}) error {
	j, err := yaml.YAMLToJSON([]byte(entry))
	if err != nil {
		return fmt.Errorf("ConfigMap's value could not be converted to JSON: %s : %v", err, entry)
	}
	return json.Unmarshal(j, &out)
}

// NewChannelDefaultsConfigFromConfigMap creates a ChannelDefaults from the supplied configMap
func NewChannelDefaultsConfigFromConfigMap(config *corev1.ConfigMap) (*ChannelDefaults, error) {
	return NewChannelDefaultsConfigFromMap(config.Data)
}

// ChannelDefaults includes the default values to be populated by the webhook.
type ChannelDefaults struct {
	// NamespaceDefaults are the default Channels CRDs for each namespace. namespace is the
	// key, the value is the default ChannelTemplate to use.
	NamespaceDefaults map[string]*ChannelTemplateSpec `json:"namespaceDefaults,omitempty"`
	// ClusterDefaultChannel is the default Channel CRD for all namespaces that are not in
	// NamespaceDefaultChannels.
	ClusterDefault *ChannelTemplateSpec `json:"clusterDefault,omitempty"`
}

// GetChannelConfig returns a namespace specific Channel Configuration, and if
// that doesn't exist, return a Cluster Default and if that doesn't exist
// return an error.
func (d *ChannelDefaults) GetChannelConfig(ns string) (*ChannelTemplateSpec, error) {
	if d == nil {
		return nil, errors.New("Defaults are nil")
	}
	value, present := d.NamespaceDefaults[ns]
	if present {
		return value, nil
	}
	if d.ClusterDefault != nil {
		return d.ClusterDefault, nil
	}
	return nil, errors.New("Defaults for Channel Configurations have not been set up.")
}
