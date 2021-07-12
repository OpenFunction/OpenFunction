package connector

import (
	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"

	openfunction "github.com/openfunction/apis/event/v1alpha1"
)

type Connector struct {
	EventSource *openfunction.EventSource    `json:"eventSource,omitempty"`
	EventBus    *openfunction.EventBus       `json:"eventBus,omitempty"`
	Trigger     *openfunction.Trigger        `json:"trigger,omitempty"`
	Component   componentsv1alpha1.Component `json:"component"`
}
