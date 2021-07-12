/*
Copyright 2021.

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

package event

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	openfunctionevent "github.com/openfunction/apis/event/v1alpha1"
	"github.com/openfunction/controllers/event/adapters"
	"github.com/openfunction/pkg/util"
)

const DefaultEventBusName = "default"

// EventBusReconciler reconciles a EventBus object
type EventBusReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	ctx    context.Context
}

//+kubebuilder:rbac:groups=event.openfunction.io,resources=eventbus,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=event.openfunction.io,resources=eventbus/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=event.openfunction.io,resources=eventbus/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EventBus object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *EventBusReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.ctx = ctx
	log := r.Log.WithValues("EventBus", req.NamespacedName)
	log.Info("eventbus reconcile starting...")

	var eb openfunctionevent.EventBus

	if err := r.Get(ctx, req.NamespacedName, &eb); err != nil {
		log.V(1).Info("EventBus deleted", "error", err)
		return ctrl.Result{}, util.IgnoreNotFound(err)
	}

	if _, err := r.createOrUpdateEventBus(&eb); err != nil {
		log.Error(err, "Failed")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// 1. Generate dapr component specs for the event bus
// 2. Create dapr component by using specs above
// 3. Set the annotation to assign the name of the dapr component associated with the event bus to "component-name"
func (r *EventBusReconciler) createOrUpdateEventBus(eb *openfunctionevent.EventBus) (ctrl.Result, error) {
	log := r.Log.WithName("createOrUpdate")

	if eb.Spec.Topic == "" {
		eb.Spec.Topic = DefaultEventBusName
	}

	// Converts the event bus specification into a connector
	// containing the specification for creating the dapr component.
	c, err := adapters.NewEventBusConnectors(eb)
	if err != nil {
		log.Error(err, "Failed to generate connectors")
		return ctrl.Result{}, err
	}

	// Create dapr component to adapter the event bus
	if err := CreateOrUpdateDaprComponent(r.Client, r.Scheme, r.Log, c, eb); err != nil {
		log.Error(err, "Failed to create dapr component", "namespace", eb.Namespace, "name", eb.Name)
		return ctrl.Result{}, err
	}

	eventBusAnnotations := eb.GetAnnotations()
	eventBusAnnotations["component-name"] = c.Component.Name
	eb.SetAnnotations(eventBusAnnotations)
	if err = r.Update(r.ctx, eb); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EventBusReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openfunctionevent.EventBus{}).
		Complete(r)
}
