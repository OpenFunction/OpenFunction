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

package events

import (
	"context"
	"errors"
	"fmt"

	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	openfunctioncore "github.com/openfunction/apis/core/v1alpha1"
	openfunctionevent "github.com/openfunction/apis/events/v1alpha1"
	"github.com/openfunction/pkg/util"
)

const (
	handlerContainerName    = "eventsource"
	eventSourceHandlerImage = "openfunctiondev/eventsource-handler:v1"
)

// EventSourceReconciler reconciles a EventSource object
type EventSourceReconciler struct {
	client.Client
	Log               logr.Logger
	Scheme            *runtime.Scheme
	EventSourceConfig *EventSourceConfig
	FunctionConfig    *openfunctioncore.Function
}

//+kubebuilder:rbac:groups=events.openfunction.io,resources=eventsources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=events.openfunction.io,resources=eventsources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=events.openfunction.io,resources=eventsources/finalizers,verbs=update
//+kubebuilder:rbac:groups=events.openfunction.io,resources=eventbus,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=events.openfunction.io,resources=eventbus/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=events.openfunction.io,resources=eventbus/finalizers,verbs=update
//+kubebuilder:rbac:groups=events.openfunction.io,resources=clustereventbus,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=events.openfunction.io,resources=clustereventbus/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=events.openfunction.io,resources=clustereventbus/finalizers,verbs=update
//+kubebuilder:rbac:groups=core.openfunction.io,resources=functions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.openfunction.io,resources=functions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.openfunction.io,resources=functions/finalizers,verbs=update
//+kubebuilder:rbac:groups=dapr.io,resources=components;subscriptions,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EventSource object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *EventSourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("EventSource", req.NamespacedName)
	log.Info("EventSource reconcile starting...")

	eventSource := &openfunctionevent.EventSource{}
	r.EventSourceConfig = &EventSourceConfig{}

	if err := r.Get(ctx, req.NamespacedName, eventSource); err != nil {
		log.V(1).Info("EventSource deleted", "error", err)
		return ctrl.Result{}, util.IgnoreNotFound(err)
	}

	r.initFunction()

	if err := r.createOrUpdateEventSource(ctx, log, eventSource); err != nil {
		log.Error(err, "Failed to create or update eventsource", "namespace", eventSource.Namespace, "name", eventSource.Name)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// createOrUpdateEventSource will do:
// 1. Generate a dapr component specification for the EventBus associated with the EventSource (if spec.eventBus is set)
//    and create the dapr component (will check if it needs to be updated)
//    and set the EventSourceConfig.EventBusComponent, EventSourceConfig.EventBusTopic, EventSourceConfig.EventBusSpecEncode.
// 2. Generate a dapr component specification for the Sink set in EventSource (if spec.sink is set)
//    and create the dapr component (will check if it needs to be updated)
//    and set the EventSourceConfig.SinkComponent, EventSourceConfig.SinkSpecEncode.
// 3. Generate dapr component specifications for the event sources
//    and create the dapr components (will check if they need to be updated)
//    and set the EventSourceConfig.EventSourceComponent, EventSourceConfig.EventSourceTopic, EventSourceConfig.EventSourceSpecEncode.
// 4. Generate EventSourceConfig and convert it to a base64-encoded string.
// 5. Create an EventSource workload for each event source (will check if they need to be updated)
//    and pass in the EventSourceConfig as an environment variable.
func (r *EventSourceReconciler) createOrUpdateEventSource(ctx context.Context, log logr.Logger, eventSource *openfunctionevent.EventSource) error {
	defer eventSource.SaveStatus(context.Background(), log, r.Client)
	log = r.Log.WithName("createOrUpdateEventSource")

	eventSource.AddCondition(*openfunctionevent.CreateCondition(
		openfunctionevent.Pending, metav1.ConditionUnknown, openfunctionevent.PendingCreation,
	).SetMessage("Identified EventSource creation signal"))

	if eventSource.Spec.EventBus == "" && eventSource.Spec.Sink == nil {
		err := errors.New("spec.evenBus or spec.sink must be set")
		condition := openfunctionevent.CreateCondition(openfunctionevent.Error, metav1.ConditionFalse, openfunctionevent.ErrorConfiguration).SetMessage(err.Error())
		eventSource.AddCondition(*condition)
		log.Error(err, "Failed to find output configuration (eventBus or sink).", "namespace", eventSource.Namespace, "name", eventSource.Name)
		return err
	}

	// Handle EventBus reconcile.
	if eventSource.Spec.EventBus != "" {
		if err := r.handleEventBus(ctx, log, eventSource); err != nil {
			return err
		}
	}

	// Handle Sink reconcile.
	if eventSource.Spec.Sink != nil {
		if err := r.handleSink(ctx, log, eventSource); err != nil {
			return err
		}
	}

	// Handle EventSource reconcile.
	if err := r.handleEventSource(ctx, log, eventSource); err != nil {
		return err
	}

	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, eventSource, r.mutateEventSource(eventSource)); err != nil {
		condition := openfunctionevent.CreateCondition(openfunctionevent.Error, metav1.ConditionFalse, openfunctionevent.ErrorCreatingEventSource).SetMessage(err.Error())
		eventSource.AddCondition(*condition)
		log.Error(err, "Failed to create or update EventSource", "namespace", eventSource.Namespace, "name", eventSource.Name)
		return err
	}
	condition := openfunctionevent.CreateCondition(openfunctionevent.Ready, metav1.ConditionTrue, openfunctionevent.EventSourceIsReady).SetMessage("EventSource is ready.")
	eventSource.AddCondition(*condition)
	eventSource.SaveStatus(ctx, log, r.Client)
	log.Info("EventSource reconcile success.", "namespace", eventSource.Namespace, "name", eventSource.Name)
	return nil
}

func (r *EventSourceReconciler) handleEventBus(ctx context.Context, log logr.Logger, eventSource *openfunctionevent.EventSource) error {

	// Retrieve the specification of EventBus associated with the EventSource
	var eventBusSpec openfunctionevent.EventBusSpec
	eventBus := retrieveEventBus(ctx, r.Client, eventSource.Namespace, eventSource.Spec.EventBus)
	if eventBus == nil {
		clusterEventBus := retrieveClusterEventBus(ctx, r.Client, eventSource.Spec.EventBus)
		if clusterEventBus == nil {
			err := errors.New("cannot retrieve eventBus or clusterEventBus")
			condition := openfunctionevent.CreateCondition(openfunctionevent.Error, metav1.ConditionFalse, openfunctionevent.ErrorToFindExistEventBus).SetMessage(err.Error())
			eventSource.AddCondition(*condition)
			log.Error(err, "Either eventBus nor clusterEventBus exists.", "namespace", eventSource.Namespace, "name", eventSource.Name)
			return err
		} else {
			eventBusSpec = clusterEventBus.Spec
		}
	} else {
		eventBusSpec = eventBus.Spec
	}

	componentName := fmt.Sprintf(EventSourceBusComponentNameTmpl, eventSource.Name)

	// Set EventSourceConfig.EventBusComponent.
	r.EventSourceConfig.EventBusComponent = componentName

	// Generate a dapr component spec based on the specification of EventBus.
	if eventBusSpec.NatsStreaming != nil {
		// Create the dapr component for EventSource to send event to EventBus.
		// We need to assign a separate consumerID name to each nats streaming component
		metadataMap := eventBusSpec.NatsStreaming.ConvertToMetadataMap()
		metadataMap = append(metadataMap, map[string]interface{}{
			"name":  "consumerID",
			"value": fmt.Sprintf("%s-%s", eventSource.Namespace, componentName),
		})

		component, err := eventBusSpec.NatsStreaming.GenComponent(eventSource.Namespace, componentName, metadataMap)
		if err != nil {
			condition := openfunctionevent.CreateCondition(openfunctionevent.Error, metav1.ConditionFalse, openfunctionevent.ErrorGenerateComponent).SetMessage(err.Error())
			eventSource.AddCondition(*condition)
			log.Error(err, "Failed to generate eventBus component of Nats Streaming.", "namespace", eventSource.Namespace, "name", eventSource.Name)
			return err
		}
		// Add the component spec to function.
		if function := addDaprComponent(r.FunctionConfig, component, "output"); function != nil {
			r.FunctionConfig = function
		}
		return nil
	}
	err := errors.New("no specification found for eventBus")
	condition := openfunctionevent.CreateCondition(openfunctionevent.Error, metav1.ConditionFalse, openfunctionevent.ErrorConfiguration).SetMessage(err.Error())
	eventSource.AddCondition(*condition)
	log.Error(err, "Failed to handle eventBus.", "namespace", eventSource.Namespace, "name", eventSource.Name)
	return err
}

func (r *EventSourceReconciler) handleSink(ctx context.Context, log logr.Logger, eventSource *openfunctionevent.EventSource) error {
	componentName := fmt.Sprintf(EventSourceSinkComponentNameTmpl, eventSource.Name)

	// Set EventSourceConfig.SinkComponent.
	r.EventSourceConfig.SinkComponent = componentName

	// Generate a dapr component spec based on the specification of Sink.
	component := &componentsv1alpha1.Component{
		ObjectMeta: metav1.ObjectMeta{
			Name:      componentName,
			Namespace: eventSource.Namespace,
		},
	}
	spec, err := newSinkComponentSpec(r.Client, r.Log, eventSource.Spec.Sink.Ref)
	if err != nil {
		condition := openfunctionevent.CreateCondition(openfunctionevent.Error, metav1.ConditionFalse, openfunctionevent.ErrorGenerateComponent).SetMessage(err.Error())
		eventSource.AddCondition(*condition)
		log.Error(err, "Failed to generate eventSource component of sink.", "namespace", eventSource.Namespace, "name", eventSource.Name)
		return err
	}
	component.Spec = *spec

	// Add the component spec to function.
	if function := addSinkComponent(r.FunctionConfig, component); function != nil {
		r.FunctionConfig = function
	}
	return nil
}

func (r *EventSourceReconciler) handleEventSource(ctx context.Context, log logr.Logger, eventSource *openfunctionevent.EventSource) error {
	var functions []*openfunctioncore.Function
	// Generate dapr components based on the specification of EventSource.
	if eventSource.Spec.Kafka != nil {
		for eventName, spec := range eventSource.Spec.Kafka {
			componentName := fmt.Sprintf(EventSourceComponentNameTmpl, eventSource.Name, SourceKindKafka, eventName)

			// We need to assign a separate consumerGroup name to each kafka component
			metadataMap := spec.ConvertToMetadataMap()
			metadataMap = append(metadataMap, map[string]interface{}{
				"name":  "consumerGroup",
				"value": fmt.Sprintf("%s-%s", eventSource.Namespace, componentName),
			})

			component, err := spec.GenComponent(eventSource.Namespace, componentName, metadataMap)
			if err != nil {
				condition := openfunctionevent.CreateCondition(openfunctionevent.Error, metav1.ConditionFalse, openfunctionevent.ErrorGenerateComponent).SetMessage(err.Error())
				eventSource.AddCondition(*condition)
				log.Error(err, "Failed to generate EventSource component of Kafka.", "namespace", eventSource.Namespace, "name", eventSource.Name)
				return err
			}
			// Generate keda scaledObject for Kafka EventSource.
			scaledObject, err := spec.GenScaledObject()
			if err != nil {
				condition := openfunctionevent.CreateCondition(openfunctionevent.Error, metav1.ConditionFalse, openfunctionevent.ErrorGenerateScaledObject).SetMessage(err.Error())
				eventSource.AddCondition(*condition)
				log.Error(err, "Failed to generate EventSource scaledObject of Kafka.", "namespace", eventSource.Namespace, "name", eventSource.Name)
			}
			if scaledObject != nil {
				scaledObject.Triggers[0].Metadata["consumerGroup"] = fmt.Sprintf("%s-%s", eventSource.Namespace, componentName)
			}
			function := r.addEventSourceForFunction(eventSource.Namespace, eventSource.Name, SourceKindKafka, eventName, component, scaledObject)
			function.SetLabels(map[string]string{EventBusTopicName: fmt.Sprintf(EventBusTopicNameTmpl, eventSource.Namespace, eventSource.Name, eventName)})
			functions = append(functions, function)
		}
	}

	if eventSource.Spec.Cron != nil {
		for eventName, spec := range eventSource.Spec.Cron {
			componentName := fmt.Sprintf(EventSourceComponentNameTmpl, eventSource.Name, SourceKindCron, eventName)

			metadataMap := spec.ConvertToMetadataMap()
			component, err := spec.GenComponent(eventSource.Namespace, componentName, metadataMap)
			if err != nil {
				condition := openfunctionevent.CreateCondition(openfunctionevent.Error, metav1.ConditionFalse, openfunctionevent.ErrorGenerateComponent).SetMessage(err.Error())
				eventSource.AddCondition(*condition)
				log.Error(err, "Failed to generate EventSource component of Cron.", "namespace", eventSource.Namespace, "name", eventSource.Name)
				return err
			}
			function := r.addEventSourceForFunction(eventSource.Namespace, eventSource.Name, SourceKindCron, eventName, component, nil)
			function.SetLabels(map[string]string{EventBusTopicName: fmt.Sprintf(EventBusTopicNameTmpl, eventSource.Namespace, eventSource.Name, eventName)})
			functions = append(functions, function)
		}
	}

	for _, f := range functions {
		l := f.GetLabels()
		r.EventSourceConfig.EventSourceComponent = f.Spec.Serving.OpenFuncAsync.Dapr.Inputs[0].Name
		r.EventSourceConfig.EventBusTopic = l[EventBusTopicName]

		// Create the workload for EventSource.
		if _, err := r.createOrUpdateEventSourceWorkload(ctx, log, eventSource, f); err != nil {
			return err
		}
	}
	return nil
}

func (r *EventSourceReconciler) createOrUpdateEventSourceWorkload(ctx context.Context, log logr.Logger, eventSource *openfunctionevent.EventSource, function *openfunctioncore.Function) (*openfunctioncore.Function, error) {
	log = r.Log.WithName("createOrUpdateEventSourceWorkload")

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, function, r.mutateHandler(function, eventSource))
	if err != nil {
		condition := openfunctionevent.CreateCondition(openfunctionevent.Error, metav1.ConditionFalse, openfunctionevent.ErrorCreatingEventSourceWorkload).SetMessage(err.Error())
		eventSource.AddCondition(*condition)
		log.Error(err, "Failed to create or update EventSource handler", "namespace", eventSource.Namespace, "name", eventSource.Name)
		return nil, err
	}
	condition := openfunctionevent.CreateCondition(openfunctionevent.Created, metav1.ConditionTrue, openfunctionevent.EventSourceWorkloadCreated).SetMessage("EventSource workload is created")
	eventSource.AddCondition(*condition)
	log.Info("Create or update EventSource handler", "namespace", eventSource.Namespace, "name", eventSource.Name)
	return function, nil
}

func (r *EventSourceReconciler) mutateHandler(function *openfunctioncore.Function, eventSource *openfunctionevent.EventSource) controllerutil.MutateFn {
	return func() error {
		l := map[string]string{
			"openfunction.io/managed":  "true",
			EventSourceControlledLabel: eventSource.Name,
		}
		function.SetLabels(l)

		envEncode, err := r.EventSourceConfig.EncodeConfig()
		if err != nil {
			return err
		}
		function.Spec.Serving.Params = map[string]string{
			"CONFIG": envEncode,
		}

		function.SetOwnerReferences(nil)
		return controllerutil.SetControllerReference(eventSource, function, r.Scheme)
	}
}

func (r *EventSourceReconciler) mutateEventSource(eventSource *openfunctionevent.EventSource) controllerutil.MutateFn {
	return func() error {
		if eventSource.GetLabels() == nil {
			eventSource.SetLabels(make(map[string]string))
		}
		if eventSource.Spec.EventBus != "" {
			eventSource.Labels[EventBusNameLabel] = eventSource.Spec.EventBus
		}
		return nil
	}
}

func (r *EventSourceReconciler) initFunction() {
	r.FunctionConfig = &openfunctioncore.Function{
		Spec: openfunctioncore.FunctionSpec{
			Image:   "zephyrfish/eventsource-handler:v2",
			Serving: &openfunctioncore.ServingImpl{},
		},
	}

	servingRuntime := openfunctioncore.OpenFuncAsync
	version := "v1.0.0"
	r.FunctionConfig.Spec.Version = &version
	r.FunctionConfig.Spec.Serving.Runtime = &servingRuntime
	r.FunctionConfig.Spec.Serving.OpenFuncAsync = &openfunctioncore.OpenFuncAsyncRuntime{
		Dapr: &openfunctioncore.Dapr{
			Annotations:   map[string]string{},
			Components:    []openfunctioncore.DaprComponent{},
			Subscriptions: []openfunctioncore.DaprSubscription{},
			Inputs:        []*openfunctioncore.DaprIO{},
			Outputs:       []*openfunctioncore.DaprIO{},
		},
		Keda: &openfunctioncore.Keda{},
	}

	// If need to build the function image when creating EventSource, uncomment the following
	//r.FunctionConfig.Spec.Build = &openfunctioncore.BuildImpl{}
	//builder := "openfunctiondev/go115-builder:v0.2.0"
	//revision := "main"
	//subPath := "eventsource/function"
	//r.FunctionConfig.Spec.Build.Builder = &builder
	//r.FunctionConfig.Spec.Build.Env = map[string]string{"FUNC_NAME": "EventSourceHandler"}
	//repo := &openfunctioncore.GitRepo{
	//	Url:           "https://github.com/openfunction/events-handlers.git",
	//	Revision:      &revision,
	//	SourceSubPath: &subPath,
	//}
	//r.FunctionConfig.Spec.Build.SrcRepo = repo
}

func (r *EventSourceReconciler) addEventSourceForFunction(namespace string, eventsourceName string, sourceKind string, eventName string, component *componentsv1alpha1.Component, scaledObject *openfunctioncore.KedaScaledObject) *openfunctioncore.Function {
	function := r.FunctionConfig
	function.Name = fmt.Sprintf(EventSourceWorkloadsNameTmpl, eventsourceName, sourceKind, eventName)
	function.Namespace = namespace
	function = addDaprComponent(function, component, "input")
	function.Spec.Serving.OpenFuncAsync.Keda.ScaledObject = scaledObject
	return function
}

// SetupWithManager sets up the controller with the Manager.
func (r *EventSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openfunctionevent.EventSource{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&openfunctioncore.Function{}).
		Watches(&source.Kind{Type: &openfunctionevent.EventBus{}}, handler.EnqueueRequestsFromMapFunc(func(object client.Object) []reconcile.Request {
			eventSourceList := &openfunctionevent.EventSourceList{}
			c := mgr.GetClient()

			err := c.List(context.TODO(), eventSourceList, &client.ListOptions{Namespace: object.GetNamespace()})
			if err != nil {
				return []reconcile.Request{}
			}

			reconcileRequests := make([]reconcile.Request, len(eventSourceList.Items))
			for _, eventSource := range eventSourceList.Items {
				if &eventSource != nil {
					if eventSource.Spec.EventBus != "" && eventSource.Spec.EventBus == object.GetName() {
						reconcileRequests = append(reconcileRequests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Namespace: eventSource.Namespace,
								Name:      eventSource.Name,
							},
						})
					}

				}
			}
			return reconcileRequests
		})).
		Watches(&source.Kind{Type: &openfunctionevent.ClusterEventBus{}}, handler.EnqueueRequestsFromMapFunc(func(object client.Object) []reconcile.Request {
			eventSourceList := &openfunctionevent.EventSourceList{}
			c := mgr.GetClient()

			selector := labels.SelectorFromSet(labels.Set(map[string]string{EventBusNameLabel: object.GetName()}))
			err := c.List(context.TODO(), eventSourceList, &client.ListOptions{LabelSelector: selector})
			if err != nil {
				return []reconcile.Request{}
			}

			reconcileRequests := make([]reconcile.Request, len(eventSourceList.Items))
			for _, eventSource := range eventSourceList.Items {
				if &eventSource != nil {
					var eventBus openfunctionevent.EventBus
					if err := c.Get(context.TODO(), client.ObjectKey{Namespace: eventSource.Namespace, Name: eventSource.Spec.EventBus}, &eventBus); err == nil {
						continue
					}

					reconcileRequests = append(reconcileRequests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: eventSource.Namespace,
							Name:      eventSource.Name,
						},
					})
				}
			}
			return reconcileRequests
		})).
		Complete(r)
}
