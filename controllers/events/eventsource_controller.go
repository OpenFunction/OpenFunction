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

package events

import (
	"context"
	"errors"
	"fmt"
	"strings"

	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"github.com/go-logr/logr"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
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

	ofcore "github.com/openfunction/apis/core/v1beta1"
	ofevent "github.com/openfunction/apis/events/v1alpha1"
	"github.com/openfunction/pkg/event/eventbus/natsstreaming"
	"github.com/openfunction/pkg/event/eventsource/cron"
	"github.com/openfunction/pkg/event/eventsource/kafka"
	"github.com/openfunction/pkg/event/eventsource/mqtt"
	"github.com/openfunction/pkg/event/eventsource/redis"
	"github.com/openfunction/pkg/util"
)

const (
	eventSourceHandlerImage = "openfunction/eventsource-handler:v4"
)

// EventSourceReconciler reconciles a EventSource object
type EventSourceReconciler struct {
	client.Client
	Log               logr.Logger
	Scheme            *runtime.Scheme
	EventSourceConfig *EventSourceConfig
	Function          *ofcore.Function
	defaultConfig     map[string]string
	newSinkUri        string
}

//+kubebuilder:rbac:groups=events.openfunction.io,resources=eventsources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=events.openfunction.io,resources=eventsources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=events.openfunction.io,resources=eventsources/finalizers,verbs=update
//+kubebuilder:rbac:groups=events.openfunction.io,resources=eventbuses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=events.openfunction.io,resources=eventbuses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=events.openfunction.io,resources=eventbuses/finalizers,verbs=update
//+kubebuilder:rbac:groups=events.openfunction.io,resources=clustereventbuses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=events.openfunction.io,resources=clustereventbuses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=events.openfunction.io,resources=clustereventbuses/finalizers,verbs=update
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

	eventSource := &ofevent.EventSource{}
	r.EventSourceConfig = &EventSourceConfig{}
	r.EventSourceConfig.LogLevel = DefaultLogLevel

	// Get default global configuration from ConfigMap
	r.defaultConfig = util.GetDefaultConfig(ctx, r.Client, r.Log)

	if err := r.Get(ctx, req.NamespacedName, eventSource); err != nil {
		log.V(1).Info("EventSource deleted", "error", err)
		return ctrl.Result{}, util.IgnoreNotFound(err)
	}

	if err := r.createOrUpdateEventSource(ctx, log, eventSource); err != nil {
		log.Error(err, "Failed to create or update eventsource",
			"namespace", eventSource.Namespace, "name", eventSource.Name)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// createOrUpdateEventSource will do:
//  1. Generate a dapr component specification for the EventBus associated with the EventSource (if spec.eventBus is set)
//     and create the dapr component (will check if it needs to be updated)
//     and set the EventSourceConfig.EventBusComponent, EventSourceConfig.EventBusTopic, EventSourceConfig.EventBusSpecEncode.
//  2. Generate a dapr component specification for the Sink set in EventSource (if spec.sink is set)
//     and create the dapr component (will check if it needs to be updated)
//     and set the EventSourceConfig.SinkComponent, EventSourceConfig.SinkSpecEncode.
//  3. Generate dapr component specifications for the event sources
//     and create the dapr components (will check if they need to be updated)
//     and set the EventSourceConfig.EventSourceComponent, EventSourceConfig.EventSourceTopic, EventSourceConfig.EventSourceSpecEncode.
//  4. Generate EventSourceConfig and convert it to a base64-encoded string.
//  5. Create an EventSource workload for each event source (will check if they need to be updated)
//     and pass in the EventSourceConfig as an environment variable.
func (r *EventSourceReconciler) createOrUpdateEventSource(ctx context.Context, log logr.Logger, eventSource *ofevent.EventSource) error {
	defer eventSource.SaveStatus(ctx, log, r.Client)
	log = r.Log.WithName("createOrUpdateEventSource")

	eventSource.AddCondition(*ofevent.CreateCondition(
		ofevent.Pending, metav1.ConditionUnknown, ofevent.PendingCreation,
	).SetMessage("Identified EventSource creation signal"))

	image := util.GetConfigOrDefault(r.defaultConfig,
		"openfunction.eventsource-handler.image",
		eventSourceHandlerImage,
	)
	// Generate the eventsource function instance with image eventSourceHandlerImage.
	r.Function = InitFunction(image)

	if eventSource.Spec.LogLevel != nil {
		r.EventSourceConfig.LogLevel = *eventSource.Spec.LogLevel
	}

	if eventSource.Spec.EventBus == "" && eventSource.Spec.Sink == nil {
		err := errors.New("must set spec.eventBus or spec.sink")
		condition := ofevent.CreateCondition(
			ofevent.Error, metav1.ConditionFalse, ofevent.ErrorConfiguration,
		).SetMessage(err.Error())
		eventSource.AddCondition(*condition)
		log.Error(err, "Failed to find output configuration (eventBus or sink).",
			"namespace", eventSource.Namespace, "name", eventSource.Name)
		return err
	}

	// Handle EventBus(ClusterEventBus) reconcile.
	if eventSource.Spec.EventBus != "" {
		if err := r.handleEventBus(ctx, log, eventSource); err != nil {
			return err
		}
	}

	// Handle Sink reconcile.
	if eventSource.Spec.Sink != nil {
		r.newSinkUri = *eventSource.Spec.Sink.Uri
		if err := r.handleSink(ctx, log, eventSource); err != nil {
			return err
		}
	}

	// Handle EventSource reconcile.
	if err := r.handleEventSource(ctx, log, eventSource); err != nil {
		return err
	}

	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, eventSource, r.mutateEventSource(eventSource, r.newSinkUri)); err != nil {
		condition := ofevent.CreateCondition(
			ofevent.Error, metav1.ConditionFalse, ofevent.ErrorCreatingEventSource,
		).SetMessage(err.Error())
		eventSource.AddCondition(*condition)
		log.Error(err, "Failed to create or update EventSource",
			"namespace", eventSource.Namespace, "name", eventSource.Name)
		return err
	}
	condition := ofevent.CreateCondition(
		ofevent.Ready, metav1.ConditionTrue, ofevent.EventSourceIsReady,
	).SetMessage("EventSource is ready.")
	eventSource.AddCondition(*condition)
	eventSource.SaveStatus(ctx, log, r.Client)
	log.Info("EventSource reconcile success.",
		"namespace", eventSource.Namespace, "name", eventSource.Name)
	return nil
}

func (r *EventSourceReconciler) handleEventBus(ctx context.Context, log logr.Logger, eventSource *ofevent.EventSource) error {
	var eventBusSpec ofevent.EventBusSpec
	// Retrieve the specification of EventBus associated with the EventSource.
	eventBus := retrieveEventBus(ctx, r.Client, eventSource.Namespace, eventSource.Spec.EventBus)
	if eventBus == nil {
		// Retrieve the specification of ClusterEventBus associated with the EventSource.
		clusterEventBus := retrieveClusterEventBus(ctx, r.Client, eventSource.Spec.EventBus)
		if clusterEventBus == nil {
			err := errors.New("cannot retrieve eventBus or clusterEventBus")
			condition := ofevent.CreateCondition(
				ofevent.Error, metav1.ConditionFalse, ofevent.ErrorToFindExistEventBus,
			).SetMessage(err.Error())
			eventSource.AddCondition(*condition)
			log.Error(err, "Neither eventBus nor clusterEventBus exists.",
				"namespace", eventSource.Namespace, "name", eventSource.Name)
			return err
		} else {
			eventBusSpec = clusterEventBus.Spec
		}
	} else {
		eventBusSpec = eventBus.Spec
	}

	componentName := fmt.Sprintf(EventSourceBusComponentNameTmpl, eventSource.Name)

	// Set EventSourceConfig.
	r.EventSourceConfig.EventBusComponent = componentName
	r.EventSourceConfig.EventBusOutputName = fmt.Sprintf(EventBusOutputNameTmpl, eventSource.Name)

	// Generate a dapr component spec based on the specification of EventBus(ClusterEventBus).
	if eventBusSpec.NatsStreaming != nil {
		eb := natsstreaming.NewNatsStreamingEventBus(log, eventBusSpec.NatsStreaming)
		// Create the dapr component for EventSource to send event to EventBus(ClusterEventBus).
		// We need to assign a separate consumerID name to each nats streaming component
		eb.SetMetadata("consumerID", fmt.Sprintf("%s-%s", eventSource.Namespace, componentName))
		component, err := eb.GenComponent(eventSource.Namespace, componentName)
		if err != nil {
			condition := ofevent.CreateCondition(
				ofevent.Error, metav1.ConditionFalse, ofevent.ErrorGenerateComponent,
			).SetMessage(err.Error())
			eventSource.AddCondition(*condition)
			log.Error(err, "Failed to generate eventBus component of Nats Streaming.",
				"namespace", eventSource.Namespace, "name", eventSource.Name)
			return err
		}
		// Add the component spec to function.
		r.Function.Spec.Serving.Pubsub[componentName] = &component.Spec
		return nil
	}
	err := errors.New("no specification found for eventBus")
	condition := ofevent.CreateCondition(
		ofevent.Error, metav1.ConditionFalse, ofevent.ErrorConfiguration,
	).SetMessage(err.Error())
	eventSource.AddCondition(*condition)
	log.Error(err, "Failed to handle eventBus.",
		"namespace", eventSource.Namespace, "name", eventSource.Name)
	return err
}

func (r *EventSourceReconciler) handleSink(ctx context.Context, log logr.Logger, eventSource *ofevent.EventSource) error {
	sink := eventSource.Spec.Sink
	component, err := createSinkComponent(ctx, r.Client, log, eventSource, sink)
	if err != nil {
		condition := ofevent.CreateCondition(
			ofevent.Error, metav1.ConditionFalse, ofevent.ErrorGenerateComponent,
		).SetMessage(err.Error())
		eventSource.AddCondition(*condition)
		log.Error(err, "Failed to generate eventSource component for sink.",
			"namespace", eventSource.Namespace, "name", eventSource.Name)
		return err
	}

	sinkOutputName := fmt.Sprintf(SinkOutputNameTmpl, "es", eventSource.Name, "1")
	// Set EventSourceConfig.SinkOutputName.
	r.EventSourceConfig.SinkOutputName = sinkOutputName

	// Add the component spec to function.
	if function := addSinkForFunction(sinkOutputName, r.Function, component); function != nil {
		r.Function = function
	}
	return nil
}

func (r *EventSourceReconciler) handleEventSource(ctx context.Context, log logr.Logger, eventSource *ofevent.EventSource) error {
	var functions []*ofcore.Function
	// Generate dapr components, keda scaleOptions and triggers based on the specification of EventSource.
	if eventSource.Spec.Kafka != nil {
		for eventName, spec := range eventSource.Spec.Kafka {
			esSpec := spec
			es := kafka.NewKafkaEventSource(log, esSpec)
			componentName := fmt.Sprintf(EventSourceComponentNameTmpl, SourceKindKafka, eventName)

			// We need to assign a separate consumerGroup name to each kafka component.
			es.SetMetadata("consumerGroup", fmt.Sprintf("%s-%s", eventSource.Namespace, componentName))

			// Generate Dapr component for Kafka EventSource.
			component, err := es.GenComponent(eventSource.Namespace, componentName)
			if err != nil {
				condition := ofevent.CreateCondition(
					ofevent.Error, metav1.ConditionFalse, ofevent.ErrorGenerateComponent,
				).SetMessage(err.Error())
				eventSource.AddCondition(*condition)
				log.Error(err, "Failed to generate eventSource component for Kafka.",
					"namespace", eventSource.Namespace,
					"name", eventSource.Name,
				)
				return err
			}

			// Generate Keda scaledObject and scaleTriggers for Kafka EventSource.
			scaledObject, trigger := es.GenScaleOptions()
			function := r.addEventSourceForFunction(eventSource, SourceKindKafka, eventName, component, scaledObject, trigger)
			functions = append(functions, function)
		}
	}

	if eventSource.Spec.Cron != nil {
		for eventName, spec := range eventSource.Spec.Cron {
			esSpec := spec
			es := cron.NewCronEventSource(log, esSpec)
			componentName := fmt.Sprintf(EventSourceComponentNameTmpl, SourceKindCron, eventName)

			// Generate Dapr component for Cron EventSource.
			component, err := es.GenComponent(eventSource.Namespace, componentName)
			if err != nil {
				condition := ofevent.CreateCondition(
					ofevent.Error, metav1.ConditionFalse, ofevent.ErrorGenerateComponent,
				).SetMessage(err.Error())
				eventSource.AddCondition(*condition)
				log.Error(err, "Failed to generate eventSource component for Cron.",
					"namespace", eventSource.Namespace, "name", eventSource.Name)
				return err
			}

			function := r.addEventSourceForFunction(eventSource, SourceKindCron, eventName, component, nil, nil)
			functions = append(functions, function)
		}
	}

	if eventSource.Spec.Mqtt != nil {
		for eventName, spec := range eventSource.Spec.Mqtt {
			esSpec := spec
			es := mqtt.NewMQTTEventSource(log, esSpec)
			componentName := fmt.Sprintf(EventSourceComponentNameTmpl, SourceKindMQTT, eventName)

			// We need to assign a separate consumerID name to each MQTT component
			es.SetMetadata("consumerID", fmt.Sprintf("%s-%s", eventSource.Namespace, componentName))

			// Generate Dapr component for MQTT EventSource.
			component, err := es.GenComponent(eventSource.Namespace, componentName)
			if err != nil {
				condition := ofevent.CreateCondition(
					ofevent.Error, metav1.ConditionFalse, ofevent.ErrorGenerateComponent,
				).SetMessage(err.Error())
				eventSource.AddCondition(*condition)
				log.Error(err, "Failed to generate eventSource component for MQTT.",
					"namespace", eventSource.Namespace, "name", eventSource.Name)
				return err
			}

			function := r.addEventSourceForFunction(eventSource, SourceKindMQTT, eventName, component, nil, nil)
			functions = append(functions, function)
		}
	}

	if eventSource.Spec.Redis != nil {
		for eventName, spec := range eventSource.Spec.Redis {
			esSpec := spec
			es := redis.NewRedisEventSource(log, esSpec)
			componentName := fmt.Sprintf(EventSourceComponentNameTmpl, SourceKindRedis, eventName)

			// Generate Dapr component for Redis EventSource.
			component, err := es.GenComponent(eventSource.Namespace, componentName)
			if err != nil {
				condition := ofevent.CreateCondition(
					ofevent.Error, metav1.ConditionFalse, ofevent.ErrorGenerateComponent,
				).SetMessage(err.Error())
				eventSource.AddCondition(*condition)
				log.Error(err, "Failed to generate eventSource component for MQTT.",
					"namespace", eventSource.Namespace, "name", eventSource.Name)
				return err
			}

			function := r.addEventSourceForFunction(eventSource, SourceKindRedis, eventName, component, nil, nil)
			functions = append(functions, function)
		}
	}

	for _, f := range functions {
		l := f.GetLabels()
		r.EventSourceConfig.EventBusTopic = l[EventBusTopicName]

		// Create the workload for EventSource.
		if err := r.createOrUpdateEventSourceFunction(ctx, log, eventSource, f); err != nil {
			return err
		}
	}
	return nil
}

func (r *EventSourceReconciler) createOrUpdateEventSourceFunction(ctx context.Context, log logr.Logger, eventSource *ofevent.EventSource, function *ofcore.Function) error {
	log = r.Log.WithName("createOrUpdateEventSourceFunction")

	// resourceVersion should not be set on objects to be created
	function.ResourceVersion = ""

	newServingSpec := function.Spec.Serving.DeepCopy()

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, function, r.mutateHandler(function, eventSource, newServingSpec))
	if err != nil {
		condition := ofevent.CreateCondition(
			ofevent.Error, metav1.ConditionFalse, ofevent.ErrorCreatingEventSourceFunction,
		).SetMessage(err.Error())
		eventSource.AddCondition(*condition)
		log.Error(err, "Failed to create or update EventSource function",
			"namespace", eventSource.Namespace, "name", eventSource.Name)
		return err
	}
	condition := ofevent.CreateCondition(
		ofevent.Created, metav1.ConditionTrue, ofevent.EventSourceFunctionCreated,
	).SetMessage("EventSource function is created")
	eventSource.AddCondition(*condition)
	log.Info("Create or update EventSource function",
		"namespace", eventSource.Namespace, "name", eventSource.Name)
	return nil
}

func (r *EventSourceReconciler) mutateHandler(function *ofcore.Function, eventSource *ofevent.EventSource, serving *ofcore.ServingImpl) controllerutil.MutateFn {
	return func() error {
		l := map[string]string{
			"openfunction.io/managed":  "true",
			EventSourceControlledLabel: eventSource.Name,
		}
		function.SetLabels(l)

		function.Spec.Serving = serving

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

func (r *EventSourceReconciler) mutateEventSource(eventSource *ofevent.EventSource, uri string) controllerutil.MutateFn {
	return func() error {
		if eventSource.GetLabels() == nil {
			eventSource.SetLabels(make(map[string]string))
		}
		if eventSource.Spec.EventBus != "" {
			eventSource.Labels[EventBusNameLabel] = eventSource.Spec.EventBus
		}
		if eventSource.Spec.Sink != nil {
			eventSource.Spec.Sink.Uri = &uri
		}
		return nil
	}
}

func (r *EventSourceReconciler) addEventSourceForFunction(
	eventSource *ofevent.EventSource,
	sourceKind string,
	eventName string,
	component *componentsv1alpha1.Component,
	scaledObject *ofcore.KedaScaledObject,
	trigger *kedav1alpha1.ScaleTriggers,
) *ofcore.Function {
	function := r.Function.DeepCopy()
	function.Name = fmt.Sprintf(EventSourceWorkloadsNameTmpl, eventSource.Name, sourceKind, eventName)
	function.Namespace = eventSource.Namespace

	// add eventsource input
	eventSourceInput := &ofcore.DaprIO{
		Name:      fmt.Sprintf(EventSourceInputNameTmpl, eventName),
		Component: component.Name,
	}
	function.Spec.Serving.Inputs = append(function.Spec.Serving.Inputs, eventSourceInput)

	// add eventbus output
	if r.EventSourceConfig.EventBusComponent != "" {
		eventBusOutput := &ofcore.DaprIO{
			Name:      fmt.Sprintf(EventBusOutputNameTmpl, eventSource.Name),
			Component: r.EventSourceConfig.EventBusComponent,
			Topic:     fmt.Sprintf(EventBusTopicNameTmpl, eventSource.Namespace, eventSource.Name, eventName),
		}
		function.Spec.Serving.Outputs = append(function.Spec.Serving.Outputs, eventBusOutput)
	}

	// add eventsource component
	if strings.Split(component.Spec.Type, ".")[0] == ofcore.DaprBindings {
		function.Spec.Serving.Bindings[component.Name] = &component.Spec
	} else {
		function.Spec.Serving.Pubsub[component.Name] = &component.Spec
	}

	// add eventsource scaleOption and trigger
	if scaledObject != nil && trigger != nil {
		if function.Spec.Serving.ScaleOptions == nil {
			function.Spec.Serving.ScaleOptions = &ofcore.ScaleOptions{}
		}

		if function.Spec.Serving.ScaleOptions.Keda == nil {
			function.Spec.Serving.ScaleOptions.Keda = &ofcore.KedaScaleOptions{}
		}

		if function.Spec.Serving.ScaleOptions.Keda.ScaledObject == nil {
			function.Spec.Serving.ScaleOptions.Keda.ScaledObject = scaledObject
		}

		if function.Spec.Serving.Triggers == nil {
			function.Spec.Serving.Triggers = []ofcore.Triggers{}
		}

		function.Spec.Serving.Triggers = append(function.Spec.Serving.Triggers, ofcore.Triggers{
			ScaleTriggers: *trigger,
		})
	}

	if eventSource.Spec.EventBus != "" {
		function.SetLabels(map[string]string{
			EventBusTopicName: fmt.Sprintf(EventBusTopicNameTmpl, eventSource.Namespace, eventSource.Name, eventName),
		})
	}
	return function
}

// SetupWithManager sets up the controller with the Manager.
func (r *EventSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ofevent.EventSource{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(&source.Kind{Type: &ofevent.EventBus{}}, handler.EnqueueRequestsFromMapFunc(func(object client.Object) []reconcile.Request {
			eventSourceList := &ofevent.EventSourceList{}
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
		Watches(&source.Kind{Type: &ofevent.ClusterEventBus{}}, handler.EnqueueRequestsFromMapFunc(func(object client.Object) []reconcile.Request {
			eventSourceList := &ofevent.EventSourceList{}
			c := mgr.GetClient()

			selector := labels.SelectorFromSet(labels.Set(map[string]string{EventBusNameLabel: object.GetName()}))
			err := c.List(context.TODO(), eventSourceList, &client.ListOptions{LabelSelector: selector})
			if err != nil {
				return []reconcile.Request{}
			}

			reconcileRequests := make([]reconcile.Request, len(eventSourceList.Items))
			for _, eventSource := range eventSourceList.Items {
				if &eventSource != nil {
					var eventBus ofevent.EventBus
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
