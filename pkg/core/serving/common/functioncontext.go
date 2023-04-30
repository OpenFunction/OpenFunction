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

package common

import (
	daprsdk "github.com/dapr/go-sdk/service/common"
)

const (
	bindings = "bindings"
	topic    = "pubsub"
)

type functionContext struct {
	Name           string                     `json:"name"`
	Version        string                     `json:"version"`
	Inputs         map[string]*functionInput  `json:"inputs,omitempty"`
	Outputs        map[string]*functionOutput `json:"outputs,omitempty"`
	States         map[string]*functionState  `json:"states,omitempty"`
	Runtime        string                     `json:"runtime"`
	Port           string                     `json:"port,omitempty"`
	State          interface{}                `json:"state,omitempty"`
	PrePlugins     []string                   `json:"prePlugins,omitempty"`
	PostPlugins    []string                   `json:"postPlugins,omitempty"`
	PluginsTracing *functionPluginsTracing    `json:"pluginsTracing,omitempty"`
	HttpPattern    string                     `json:"httpPattern,omitempty"`
}

type functionInput struct {
	Uri               string                `json:"uri,omitempty"`
	ComponentName     string                `json:"componentName"`
	ComponentType     string                `json:"componentType"`
	Metadata          map[string]string     `json:"metadata,omitempty"`
	PubSubRoutingRule *daprsdk.Subscription `json:"pubSubRoutingRule,omitempty"`
}

type functionOutput struct {
	Uri           string            `json:"uri,omitempty"`
	ComponentName string            `json:"componentName"`
	ComponentType string            `json:"componentType"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Operation     string            `json:"operation,omitempty"`
}

type functionState struct {
	ComponentName string `json:"componentName"`
	ComponentType string `json:"componentType"`
}

type functionPluginsTracing struct {
	Enabled  bool              `json:"enabled" yaml:"enabled"`
	Provider *tracingProvider  `json:"provider" yaml:"provider"`
	Tags     map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Baggage  map[string]string `json:"baggage" yaml:"baggage"`
}

type tracingProvider struct {
	Name      string    `json:"name" yaml:"name"`
	OapServer string    `json:"oapServer,omitempty" yaml:"oapServer,omitempty"`
	Exporter  *exporter `json:"exporter,omitempty" yaml:"exporter,omitempty"`
}

type exporter struct {
	Name        string `json:"name" yaml:"name"`
	Endpoint    string `json:"endpoint" yaml:"endpoint"`
	Headers     string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Compression string `json:"compression,omitempty" yaml:"compression,omitempty"`
	Timeout     string `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Protocol    string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
}

type plugins struct {
	Order []string `yaml:"order,omitempty"`
	Pre   []string `yaml:"pre,omitempty"`
	Post  []string `yaml:"post,omitempty"`
}
