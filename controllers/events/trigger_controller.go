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
	"strings"

	openfunctioncontext "github.com/OpenFunction/functions-framework-go/openfunction-context"

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

	ofcore "github.com/openfunction/apis/core/v1alpha2"
	ofevent "github.com/openfunction/apis/events/v1alpha1"
	"github.com/openfunction/pkg/util"
)

const (
	triggerContainerName  = "trigger"
	triggerHandlerImage   = "openfunctiondev/trigger-handler:v1"
	triggerHandlerImageV2 = "zephyrfish/trigger-handler:v2.1"
)

// TriggerReconciler reconciles a Trigger object
type TriggerReconciler struct {
	client.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
	TriggerConfig *TriggerConfig
	Function      *ofcore.Function
}

type Subscribers struct {
	Sinks            []*ofevent.SinkSpec `json:"sinks,omitempty"`
	DeadLetterSinks  []*ofevent.SinkSpec `json:"deadLetterSinks,omitempty"`
	TotalSinks       []*ofevent.SinkSpec `json:"totalSinks,omitempty"`
	Topics           []string            `json:"topics,omitempty"`
	DeadLetterTopics []string            `json:"deadLetterTopics,omitempty"`
	TotalTopics      []string            `json:"totalTopics,omitempty"`
}

//+kubebuilder:rbac:groups=events.openfunction.io,resources=triggers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=events.openfunction.io,resources=triggers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=events.openfunction.io,resources=triggers/finalizers,verbs=update
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
// the Trigger object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *TriggerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("Trigger", req.NamespacedName)
	log.Info("trigger reconcile starting...")

	trigger := &ofevent.Trigger{}
	r.TriggerConfig = &TriggerConfig{}
	r.TriggerConfig.Subscribers = map[string]*Subscriber{}

	if err := r.Get(ctx, req.NamespacedName, trigger); err != nil {
		log.V(1).Info("Trigger deleted", "error", err)
		return ctrl.Result{}, util.IgnoreNotFound(err)
	}

	r.Function = InitFunction(triggerHandlerImageV2)

	if err := r.createOrUpdateTrigger(ctx, log, trigger); err != nil {
		log.Error(err, "Failed to create or update trigger",
			"namespace", trigger.Namespace, "name", trigger.Name,
		)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *TriggerReconciler) createOrUpdateTrigger(ctx context.Context, log logr.Logger, trigger *ofevent.Trigger) error {
	defer trigger.SaveStatus(ctx, log, r.Client)
	log = r.Log.WithName("createOrUpdateTrigger")

	trigger.AddCondition(*ofevent.CreateCondition(
		ofevent.Pending, metav1.ConditionUnknown, ofevent.PendingCreation,
	).SetMessage("Identified Trigger creation signal"))

	// Handle EventBus reconcile.
	if trigger.Spec.EventBus != "" {
		if err := r.handleEventBus(ctx, log, trigger); err != nil {
			return err
		}
	} else {
		err := errors.New("must set spec.eventBus")
		condition := ofevent.CreateCondition(
			ofevent.Error, metav1.ConditionFalse, ofevent.ErrorConfiguration,
		).SetMessage(err.Error())
		trigger.AddCondition(*condition)
		log.Error(err, "Failed to find event bus configuration.",
			"namespace", trigger.Namespace, "name", trigger.Name,
		)
		return err
	}

	// Handle Subscriber reconcile.
	if err := r.handleSubscriber(ctx, log, trigger); err != nil {
		return err
	}

	// Handle Trigger workload reconcile
	if err := r.createOrUpdateTriggerWorkload(ctx, log, trigger); err != nil {
		return err
	}

	// Create or update Trigger
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, trigger, r.mutateTrigger(trigger)); err != nil {
		condition := ofevent.CreateCondition(
			ofevent.Error, metav1.ConditionFalse, ofevent.ErrorCreatingTrigger,
		).SetMessage(err.Error())
		trigger.AddCondition(*condition)
		log.Error(err, "Failed to create or update Trigger",
			"namespace", trigger.Namespace,
			"name", trigger.Name,
		)
		return err
	}
	condition := ofevent.CreateCondition(
		ofevent.Ready, metav1.ConditionTrue, ofevent.TriggerIsReady,
	).SetMessage("Trigger is ready.")
	trigger.AddCondition(*condition)
	trigger.SaveStatus(ctx, log, r.Client)
	log.Info("Trigger reconcile success.",
		"namespace", trigger.Namespace,
		"name", trigger.Name,
	)
	return nil
}

func (r *TriggerReconciler) handleEventBus(ctx context.Context, log logr.Logger, trigger *ofevent.Trigger) error {

	// Retrieve the specification of EventBus associated with the Trigger
	var eventBusSpec ofevent.EventBusSpec
	eventBus := retrieveEventBus(ctx, r.Client, trigger.Namespace, trigger.Spec.EventBus)
	if eventBus == nil {
		clusterEventBus := retrieveClusterEventBus(ctx, r.Client, trigger.Spec.EventBus)
		if clusterEventBus == nil {
			err := errors.New("cannot retrieve eventBus or clusterEventBus")
			condition := ofevent.CreateCondition(
				ofevent.Error, metav1.ConditionFalse, ofevent.ErrorToFindExistEventBus,
			).SetMessage(err.Error())
			trigger.AddCondition(*condition)
			log.Error(err,
				"Neither eventBus nor clusterEventBus exists.",
				"namespace", trigger.Namespace,
				"name", trigger.Name,
			)
			return err
		} else {
			eventBusSpec = clusterEventBus.Spec
		}
	} else {
		eventBusSpec = eventBus.Spec
	}

	componentName := fmt.Sprintf(TriggerBusComponentNameTmpl, trigger.Name)
	consumerID := fmt.Sprintf("%s-%s", trigger.Namespace, componentName)
	var subjects []string

	// Set TriggerConfig.EventBusComponentName and TriggerConfig.EventBusTopics.
	r.TriggerConfig.EventBusComponent = componentName
	for inputName, input := range trigger.Spec.Inputs {
		in := input
		if in.Namespace == "" {
			in.Namespace = trigger.Namespace
		}
		r.TriggerConfig.Inputs = append(r.TriggerConfig.Inputs, &Input{
			Name:        inputName,
			Namespace:   in.Namespace,
			EventSource: in.EventSource,
			Event:       in.Event,
		})
		subjects = append(subjects, fmt.Sprintf(EventBusTopicNameTmpl, in.Namespace, in.EventSource, in.Event))
	}

	// Generate a dapr component based on the specification of EventBus.
	if eventBusSpec.NatsStreaming != nil {
		// Create the dapr component for Trigger to retrieve event from EventBus.
		// We need to assign a separate consumerID name to each nats streaming component
		metadataMap := eventBusSpec.NatsStreaming.ConvertToMetadataMap()
		metadataMap = append(metadataMap, map[string]interface{}{
			"name":  "consumerID",
			"value": consumerID,
		})
		component, err := eventBusSpec.NatsStreaming.GenComponent(trigger.Namespace, componentName, metadataMap)
		if err != nil {
			condition := ofevent.CreateCondition(
				ofevent.Error, metav1.ConditionFalse, ofevent.ErrorGenerateComponent,
			).SetMessage(err.Error())
			trigger.AddCondition(*condition)
			log.Error(err, "Failed to generate EventBus component for Nats Streaming.",
				"namespace", trigger.Namespace,
				"name", trigger.Name,
			)
			return err
		}

		// Generate keda scaledObject for Nats Streaming EventBus.
		scaledObject, err := eventBusSpec.NatsStreaming.GenEventBusScaledObject(subjects, consumerID)
		if err != nil {
			condition := ofevent.CreateCondition(
				ofevent.Error, metav1.ConditionFalse, ofevent.ErrorGenerateScaledObject,
			).SetMessage(err.Error())
			trigger.AddCondition(*condition)
			log.Error(err, "Failed to generate EventBus scaledObject for Nats Streaming.",
				"namespace", trigger.Namespace,
				"name", trigger.Name,
			)
		}
		r.Function = r.addEventBusForFunction(trigger, component, subjects, scaledObject)
	}
	return nil
}

func (r *TriggerReconciler) handleSubscriber(ctx context.Context, log logr.Logger, trigger *ofevent.Trigger) error {
	sinks := map[*ofevent.SinkSpec]bool{}
	deadLetterSinks := map[*ofevent.SinkSpec]bool{}
	totalSinks := map[*ofevent.SinkSpec]bool{}

	topics := map[string]bool{}
	deadLetterTopics := map[string]bool{}
	totalTopics := map[string]bool{}

	if trigger.Spec.Subscribers != nil {
		// For each subscriber, check its Sink and DeadLetterSink (for synchronous calls) and Topic and DeadLetterTopic (for asynchronous calls).
		// For Sink and DeadLetterSink, the specifications of corresponding dapr component will be generated
		// and then three slices will be created to store:
		//    1. component specifications for Sink
		//    2. component specifications for DeadLetterSink
		//    3. component specifications for aggregate de-duplication of above 1 and 2
		for _, subscriber := range trigger.Spec.Subscribers {
			sub := subscriber
			s := &Subscriber{}

			if sub.Sink != nil && !sinks[sub.Sink] {
				sinks[sub.Sink] = true
				s.SinkOutputName = fmt.Sprintf(SinkOutputNameTmpl, sub.Sink.Ref.Namespace, sub.Sink.Ref.Name)
				if !totalSinks[sub.Sink] {
					totalSinks[sub.Sink] = true
					component, err := createSinkComponent(ctx, r.Client, log, trigger, sub.Sink)
					if err != nil {
						condition := ofevent.CreateCondition(
							ofevent.Error, metav1.ConditionFalse, ofevent.ErrorGenerateComponent,
						).SetMessage(err.Error())
						trigger.AddCondition(*condition)
						log.Error(err, "Failed to generate Trigger component for subscriber.",
							"namespace", trigger.Namespace,
							"name", trigger.Name,
						)
						return err
					}
					if function := addSinkForFunction(s.SinkOutputName, r.Function, component); function != nil {
						r.Function = function
					}
				}
			}

			if sub.DeadLetterSink != nil && !deadLetterSinks[sub.DeadLetterSink] {
				deadLetterSinks[sub.DeadLetterSink] = true
				s.DLSinkOutputName = fmt.Sprintf(SinkOutputNameTmpl, sub.Sink.Ref.Namespace, sub.Sink.Ref.Name)
				if !totalSinks[sub.DeadLetterSink] {
					totalSinks[sub.DeadLetterSink] = true
					component, err := createSinkComponent(ctx, r.Client, log, trigger, sub.Sink)
					if err != nil {
						condition := ofevent.CreateCondition(
							ofevent.Error, metav1.ConditionFalse, ofevent.ErrorGenerateComponent,
						).SetMessage(err.Error())
						trigger.AddCondition(*condition)
						log.Error(err, "Failed to generate Trigger component for subscriber.",
							"namespace", trigger.Namespace,
							"name", trigger.Name,
						)
						return err
					}
					if function := addSinkForFunction(s.SinkOutputName, r.Function, component); function != nil {
						r.Function = function
					}
				}
			}

			if sub.Topic != "" && !topics[sub.Topic] {
				topics[sub.Topic] = true
				s.EventBusOutputName = fmt.Sprintf(EventBusOutputNameTmpl, sub.Topic)
				if !totalTopics[sub.Topic] {
					totalTopics[sub.Topic] = true
					d := r.Function.Spec.Serving.OpenFuncAsync.Dapr
					dio := &ofcore.DaprIO{
						Name:      s.EventBusOutputName,
						Component: r.TriggerConfig.EventBusComponent,
						Topic:     sub.Topic,
						Type:      string(openfunctioncontext.OpenFuncTopic),
					}
					d.Outputs = append(d.Outputs, dio)
					r.Function.Spec.Serving.OpenFuncAsync.Dapr = d
				}
			}

			if sub.DeadLetterTopic != "" && !deadLetterTopics[sub.DeadLetterTopic] {
				deadLetterTopics[sub.DeadLetterTopic] = true
				s.DLEventBusOutputName = fmt.Sprintf(EventBusOutputNameTmpl, sub.DeadLetterTopic)
				if !totalTopics[sub.DeadLetterTopic] {
					totalTopics[sub.DeadLetterTopic] = true
					d := r.Function.Spec.Serving.OpenFuncAsync.Dapr
					dio := &ofcore.DaprIO{
						Name:      s.DLEventBusOutputName,
						Component: r.TriggerConfig.EventBusComponent,
						Topic:     sub.DeadLetterTopic,
						Type:      string(openfunctioncontext.OpenFuncTopic),
					}
					d.Outputs = append(d.Outputs, dio)
					r.Function.Spec.Serving.OpenFuncAsync.Dapr = d
				}
			}
			r.TriggerConfig.Subscribers[sub.Condition] = s
		}
	} else {
		err := errors.New("no subscriber found")
		condition := ofevent.CreateCondition(
			ofevent.Error, metav1.ConditionFalse, ofevent.ErrorToFindTriggerSubscribers,
		).SetMessage(err.Error())
		trigger.AddCondition(*condition)
		log.Error(err, "Failed to find subscribers for Trigger.",
			"namespace", trigger.Namespace,
			"name", trigger.Name,
		)
		return err
	}
	return nil
}

func (r *TriggerReconciler) createOrUpdateTriggerWorkload(ctx context.Context, log logr.Logger, trigger *ofevent.Trigger) error {
	log = r.Log.WithName("createOrUpdateTriggerWorkload")

	function := r.Function

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, function, r.mutateHandler(function, trigger))
	if err != nil {
		condition := ofevent.CreateCondition(
			ofevent.Error, metav1.ConditionFalse, ofevent.ErrorCreatingTriggerWorkload,
		).SetMessage(err.Error())
		trigger.AddCondition(*condition)
		log.Error(err, "Failed to create or update Trigger workload",
			"namespace", trigger.Namespace,
			"name", trigger.Name,
		)
		return err
	}
	condition := ofevent.CreateCondition(
		ofevent.Created, metav1.ConditionTrue, ofevent.TriggerWorkloadCreated,
	).SetMessage("Trigger workload is created")
	trigger.AddCondition(*condition)
	log.Info("Create or update Trigger workload",
		"namespace", trigger.Namespace,
		"name", trigger.Name,
	)
	return nil
}

func (r *TriggerReconciler) mutateHandler(function *ofcore.Function, trigger *ofevent.Trigger) controllerutil.MutateFn {
	return func() error {
		l := map[string]string{
			"openfunction.io/managed": "true",
			TriggerControlledLabel:    trigger.Name,
		}
		function.SetLabels(l)

		envEncode, err := r.TriggerConfig.EncodeConfig()
		if err != nil {
			return err
		}
		function.Spec.Serving.Params = map[string]string{
			"CONFIG": envEncode,
		}

		function.SetOwnerReferences(nil)
		return controllerutil.SetControllerReference(trigger, function, r.Scheme)
	}
}

func (r *TriggerReconciler) mutateTrigger(trigger *ofevent.Trigger) controllerutil.MutateFn {
	return func() error {
		if trigger.GetLabels() == nil {
			trigger.SetLabels(make(map[string]string))
		}
		trigger.Labels[EventBusNameLabel] = trigger.Spec.EventBus
		return nil
	}
}

func (r *TriggerReconciler) addEventBusForFunction(
	trigger *ofevent.Trigger,
	component *componentsv1alpha1.Component,
	subjects []string,
	scaledObject *ofcore.KedaScaledObject,
) *ofcore.Function {
	function := r.Function
	function.Name = fmt.Sprintf(TriggerWorkloadsNameTmpl, trigger.Name)
	function.Namespace = trigger.Namespace
	d := function.Spec.Serving.OpenFuncAsync.Dapr

	for _, subject := range subjects {
		dio := &ofcore.DaprIO{
			Name:      fmt.Sprintf(TriggerInputNameTmpl, subject),
			Component: component.Name,
			Topic:     subject,
			Type:      strings.Split(component.Spec.Type, ".")[0],
		}
		d.Inputs = append(d.Inputs, dio)
	}
	d.Components[component.Name] = &component.Spec
	function.Spec.Serving.OpenFuncAsync.Dapr = d
	function.Spec.Serving.OpenFuncAsync.Keda.ScaledObject = scaledObject
	return function
}

// SetupWithManager sets up the controller with the Manager.
func (r *TriggerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ofevent.Trigger{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(&source.Kind{Type: &ofevent.EventBus{}}, handler.EnqueueRequestsFromMapFunc(func(object client.Object) []reconcile.Request {
			triggerList := &ofevent.TriggerList{}
			c := mgr.GetClient()

			err := c.List(context.TODO(), triggerList, &client.ListOptions{Namespace: object.GetNamespace()})
			if err != nil {
				return []reconcile.Request{}
			}

			reconcileRequests := make([]reconcile.Request, len(triggerList.Items))
			for _, trigger := range triggerList.Items {
				if &trigger != nil {
					if trigger.Spec.EventBus == object.GetName() {
						reconcileRequests = append(reconcileRequests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Namespace: trigger.Namespace,
								Name:      trigger.Name,
							},
						})
					}
				}
			}
			return reconcileRequests
		})).
		Watches(&source.Kind{Type: &ofevent.ClusterEventBus{}}, handler.EnqueueRequestsFromMapFunc(func(object client.Object) []reconcile.Request {
			triggerList := &ofevent.TriggerList{}
			c := mgr.GetClient()

			selector := labels.SelectorFromSet(labels.Set(map[string]string{EventBusNameLabel: object.GetName()}))
			err := c.List(context.TODO(), triggerList, &client.ListOptions{LabelSelector: selector})
			if err != nil {
				return []reconcile.Request{}
			}

			reconcileRequests := make([]reconcile.Request, len(triggerList.Items))
			for _, trigger := range triggerList.Items {
				if &trigger != nil {
					var eventBus ofevent.EventBus
					if err := c.Get(context.TODO(), client.ObjectKey{Namespace: trigger.Namespace, Name: trigger.Spec.EventBus}, &eventBus); err == nil {
						continue
					}
					reconcileRequests = append(reconcileRequests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: trigger.Namespace,
							Name:      trigger.Name,
						},
					})
				}
			}
			return reconcileRequests
		})).
		Complete(r)
}
