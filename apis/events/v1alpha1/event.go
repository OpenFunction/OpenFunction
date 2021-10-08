package v1alpha1

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
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
