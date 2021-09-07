package v1alpha1

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ErrorConfiguration               EventSourceConditionReason = "ErrorConfiguration"
	ErrorToFindExistEventBus         EventSourceConditionReason = "ErrorToFindExistEventBus"
	ErrorGenerateComponent           EventSourceConditionReason = "ErrorGenerateComponent"
	ErrorGenerateScaledObject        EventSourceConditionReason = "ErrorGenerateScaledObject"
	ErrorCreatingEventSourceWorkload EventSourceConditionReason = "ErrorCreatingEventSourceWorkload"
	ErrorCreatingEventSource         EventSourceConditionReason = "ErrorCreatingEventSource"
	EventSourceWorkloadCreated       EventSourceConditionReason = "EventSourceWorkloadCreated"
	PendingCreation                  EventSourceConditionReason = "PendingCreation"
	EventSourceIsReady               EventSourceConditionReason = "EventSourceIsReady"
)

const (
	// Created indicates the resource has been created
	Created EventSourceCreationStatus = "Created"
	// Terminated indicates the resource has been terminated
	Terminated EventSourceCreationStatus = "Terminated"
	// Error indicates the resource had an error
	Error EventSourceCreationStatus = "Error"
	// Pending indicates the resource hasn't been created
	Pending EventSourceCreationStatus = "Pending"
	// Terminating indicates that the resource is marked for deletion but hasn't
	// been deleted yet
	Terminating EventSourceCreationStatus = "Terminating"
	// Unknown indicates the status is unavailable
	Unknown EventSourceCreationStatus = "Unknown"
	// Ready indicates the object is fully created
	Ready EventSourceCreationStatus = "Ready"
)

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

// AddCondition adds a new condition to the HTTPScaledObject
func (es *EventSource) AddCondition(condition EventSourceCondition) *EventSource {
	es.Status.Conditions = append(es.Status.Conditions, condition)
	return es
}

// CreateCondition initializes a new status condition
func CreateCondition(
	condType EventSourceCreationStatus,
	status metav1.ConditionStatus,
	reason EventSourceConditionReason,
) *EventSourceCondition {
	cond := EventSourceCondition{
		Timestamp: time.Now().Format(time.RFC3339),
		Type:      condType,
		Status:    status,
		Reason:    reason,
	}
	return &cond
}

// SetMessage sets the optional reason for the condition
func (c *EventSourceCondition) SetMessage(message string) *EventSourceCondition {
	c.Message = message
	return c
}
