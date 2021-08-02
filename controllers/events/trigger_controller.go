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
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/labels"

	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	openfunction "github.com/openfunction/apis/events/v1alpha1"
	openfunctionevent "github.com/openfunction/apis/events/v1alpha1"
	"github.com/openfunction/pkg/util"
)

const (
	triggerContainerName = "trigger"
	triggerHandlerImage  = "openfunctiondev/trigger-handler:latest"
)

// TriggerReconciler reconciles a Trigger object
type TriggerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	ctx    context.Context
	envs   *TriggerConfig
}

type Subscribers struct {
	Sinks            []*openfunctionevent.SinkSpec `json:"sinks,omitempty"`
	DeadLetterSinks  []*openfunctionevent.SinkSpec `json:"deadLetterSinks,omitempty"`
	TotalSinks       []*openfunctionevent.SinkSpec `json:"totalSinks,omitempty"`
	Topics           []string                      `json:"topics,omitempty"`
	DeadLetterTopics []string                      `json:"deadLetterTopics,omitempty"`
	TotalTopics      []string                      `json:"totalTopics,omitempty"`
}

//+kubebuilder:rbac:groups=events.openfunction.io,resources=triggers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=events.openfunction.io,resources=triggers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=events.openfunction.io,resources=triggers/finalizers,verbs=update

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
	r.ctx = ctx
	log := r.Log.WithValues("Trigger", req.NamespacedName)
	log.Info("trigger reconcile starting...")

	var trigger openfunctionevent.Trigger
	r.envs = &TriggerConfig{}

	if err := r.Get(ctx, req.NamespacedName, &trigger); err != nil {
		log.V(1).Info("Trigger deleted", "error", err)
		return ctrl.Result{}, util.IgnoreNotFound(err)
	}

	if _, err := r.createOrUpdateTrigger(&trigger); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// createOrUpdateTrigger will do:
// 1. Generate a dapr component specification for the EventBus associated with the Trigger
//    and create the dapr component (will check if it needs to be updated)
//    and set the TriggerConfig.EventBusComponentName, TriggerConfig.EventBusTopic, TriggerConfig.EventBusSpecEncode.
// 2. Generate dapr component specifications for the subscribers
//    and create the dapr components (will check if they need to be updated)
//    and set the TriggerConfig.Subscribers, TriggerConfig.SinkSpecEncode.
// 3. Create a workload for Trigger (will check if they need to be updated)
//    and pass in the TriggerConfig as an environment variable.
func (r *TriggerReconciler) createOrUpdateTrigger(trigger *openfunctionevent.Trigger) (ctrl.Result, error) {
	log := r.Log.WithName("createOrUpdate")

	var existComponents componentsv1alpha1.ComponentList
	var existWorkloads appsv1.DeploymentList
	controlledLabelSelector := labels.SelectorFromSet(labels.Set(map[string]string{TriggerControlledLabel: trigger.Name}))

	if err := r.List(r.ctx, &existComponents, &client.ListOptions{LabelSelector: controlledLabelSelector}); err != nil {
		log.Error(err, "Failed to retrieve exist controlled components", "namespace", trigger.Namespace, "name", trigger.Name)
		return ctrl.Result{}, err
	}
	componentsPendingDelete := map[string]bool{}
	for _, component := range existComponents.Items {
		c := component
		componentsPendingDelete[c.Name] = true
	}

	if err := r.List(r.ctx, &existWorkloads, &client.ListOptions{LabelSelector: controlledLabelSelector}); err != nil {
		log.Error(err, "Failed to retrieve exist controlled workloads", "namespace", trigger.Namespace, "name", trigger.Name)
		return ctrl.Result{}, err
	}
	workloadsPendingDelete := map[string]bool{}
	for _, workload := range existWorkloads.Items {
		w := workload
		workloadsPendingDelete[w.Name] = true
	}

	// Handle EventBus reconcile.
	if trigger.Spec.EventBus != "" {
		if err := r.handleEventBus(trigger, componentsPendingDelete); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Handle Subscriber reconcile.
	if err := r.handleSubscriber(trigger, componentsPendingDelete, workloadsPendingDelete); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.handleDeprecatedComponents(trigger, componentsPendingDelete); err != nil {
		log.Error(err, "Failed to handle deprecated components", "namespace", trigger.Namespace, "name", trigger.Name)
		return ctrl.Result{}, err
	}

	if err := r.handleDeprecatedWorkloads(trigger, workloadsPendingDelete); err != nil {
		log.Error(err, "Failed to handle deprecated workloads", "namespace", trigger.Namespace, "name", trigger.Name)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *TriggerReconciler) handleEventBus(trigger *openfunctionevent.Trigger, componentsPendingDelete map[string]bool) error {

	// Retrieve the specification of EventBus associated with the Trigger
	var eventBusSpec openfunctionevent.EventBusSpec
	eventBus := retrieveEventBus(r.ctx, r.Client, trigger.Namespace, trigger.Spec.EventBus)
	if eventBus == nil {
		clusterEventBus := retrieveClusterEventBus(r.ctx, r.Client, trigger.Spec.EventBus)
		if clusterEventBus == nil {
			return errors.New("cannot retrieve eventBus and clusterEventBus")
		} else {
			eventBusSpec = clusterEventBus.Spec
		}
	} else {
		eventBusSpec = eventBus.Spec
	}

	componentName := fmt.Sprintf(TriggerBusComponentNameTmpl, trigger.Name)
	if componentsPendingDelete[componentName] {
		componentsPendingDelete[componentName] = false
	}

	// Set TriggerConfig.EventBusComponentName and TriggerConfig.EventBusTopics.
	r.envs.EventBusComponentName = componentName
	for _, input := range trigger.Spec.Inputs {
		if input.EventSourceNamespace == "" {
			input.EventSourceNamespace = trigger.Namespace
		}
		r.envs.EventBusTopics = append(r.envs.EventBusTopics, fmt.Sprintf(EventBusTopicNameTmpl, input.EventSourceNamespace, input.EventSourceName, input.EventName))
	}

	// Generate a dapr component based on the specification of EventBus.
	component := &componentsv1alpha1.Component{
		ObjectMeta: metav1.ObjectMeta{
			Name:      componentName,
			Namespace: trigger.Namespace,
		},
	}

	if eventBusSpec.Nats != nil {
		specBytes, err := json.Marshal(eventBusSpec.Nats)
		if err != nil {
			return err
		}

		// Set the TriggerConfig.EventBusSpecEncode which will reflect the specification content changes of dapr component back to the Trigger workload,
		// informing it that it needs to rebuild.
		r.envs.EventBusSpecEncode = base64.StdEncoding.EncodeToString(specBytes)

		// Create the dapr component for Trigger to retrieve event from EventBus.
		component.Spec = *eventBusSpec.Nats
		if _, err := ctrl.CreateOrUpdate(r.ctx, r.Client, component, mutateDaprComponent(r.Scheme, component, trigger)); err != nil {
			return err
		}
	} else {
		return nil
	}
	return nil
}

func (r *TriggerReconciler) handleSubscriber(trigger *openfunctionevent.Trigger, componentsPendingDelete map[string]bool, workloadsPendingDelete map[string]bool) error {
	var components []*componentsv1alpha1.Component

	sinks := map[*openfunction.SinkSpec]bool{}
	deadLetterSinks := map[*openfunction.SinkSpec]bool{}
	totalSinks := map[*openfunction.SinkSpec]bool{}

	topics := map[string]bool{}
	deadLetterTopics := map[string]bool{}
	totalTopics := map[string]bool{}

	if trigger.Spec.Subscribers != nil {
		var sinkSpecList []*componentsv1alpha1.ComponentSpec

		// For each subscriber, check its Sink and DeadLetterSink (for synchronous calls) and Topic and DeadLetterTopic (for asynchronous calls).
		// For Sink and DeadLetterSink, the specifications of corresponding dapr component will be generated
		// and then three slices will be created to store:
		//    1. component specifications for Sink
		//    2. component specifications for DeadLetterSink
		//    3. component specifications for aggregate de-duplication of above 1 and 2
		for _, subscriber := range trigger.Spec.Subscribers {
			sub := subscriber
			sc := &SubscriberConfigs{}

			if sub.Sink != nil && !sinks[sub.Sink] {
				sinks[sub.Sink] = true
				componentName := fmt.Sprintf(TriggerSinkComponentNameTmpl, trigger.Name, sub.Sink.Ref.Namespace, sub.Sink.Ref.Name)
				sc.SinkComponentName = componentName
				if !totalSinks[sub.Sink] {
					totalSinks[sub.Sink] = true
					component := &componentsv1alpha1.Component{
						ObjectMeta: metav1.ObjectMeta{
							Name:      componentName,
							Namespace: trigger.Namespace,
						},
					}
					spec, err := newSinkComponentSpec(r.Client, r.Log, sub.Sink.Ref)
					if err != nil {
						return err
					}
					sinkSpecList = append(sinkSpecList, spec)
					component.Spec = *spec
					components = append(components, component)
				}
			}

			if sub.DeadLetterSink != nil && !deadLetterSinks[sub.DeadLetterSink] {
				deadLetterSinks[sub.DeadLetterSink] = true
				componentName := fmt.Sprintf(TriggerSinkComponentNameTmpl, trigger.Name, sub.DeadLetterSink.Ref.Namespace, sub.DeadLetterSink.Ref.Name)
				sc.DeadLetterSinkComponentName = componentName
				if !totalSinks[sub.DeadLetterSink] {
					totalSinks[sub.DeadLetterSink] = true
					component := &componentsv1alpha1.Component{
						ObjectMeta: metav1.ObjectMeta{
							Name:      componentName,
							Namespace: trigger.Namespace,
						},
					}
					spec, err := newSinkComponentSpec(r.Client, r.Log, sub.DeadLetterSink.Ref)
					if err != nil {
						return err
					}
					sinkSpecList = append(sinkSpecList, spec)
					component.Spec = *spec
					components = append(components, component)
				}
			}

			if sub.Topic != "" && !topics[sub.Topic] {
				topics[sub.Topic] = true
				sc.TopicName = sub.Topic
				if !totalTopics[sub.Topic] {
					totalTopics[sub.Topic] = true
				}
			}

			if sub.DeadLetterTopic != "" && !deadLetterTopics[sub.DeadLetterTopic] {
				deadLetterTopics[sub.DeadLetterTopic] = true
				sc.DeadLetterTopicName = sub.DeadLetterTopic
				if !totalTopics[sub.DeadLetterTopic] {
					totalTopics[sub.DeadLetterTopic] = true
				}
			}

			r.envs.Subscribers = append(r.envs.Subscribers, sc)
		}
		specListBytes, err := json.Marshal(sinkSpecList)
		if err != nil {
			return err
		}

		// Set the TriggerConfig.SinkSpecEncode which will reflect the specification content changes of dapr component back to the Trigger workload,
		// informing it that it needs to rebuild.
		r.envs.SinkSpecEncode = base64.StdEncoding.EncodeToString(specListBytes)

	} else {
		return errors.New("no subscriber found")
	}

	for _, component := range components {
		c := component
		if componentsPendingDelete[component.Name] {
			componentsPendingDelete[component.Name] = false
		}
		if _, err := ctrl.CreateOrUpdate(r.ctx, r.Client, c, mutateDaprComponent(r.Scheme, c, trigger)); err != nil {
			return err
		}
		// Create the workload for Trigger.
		if _, err := r.createOrUpdateTriggerWorkload(trigger, workloadsPendingDelete); err != nil {
			return err
		}
	}
	return nil
}

func (r *TriggerReconciler) handleDeprecatedComponents(trigger *openfunctionevent.Trigger, cpd map[string]bool) error {
	log := r.Log.WithName("handleDeprecatedComponents")
	var component componentsv1alpha1.Component
	for componentName, isDeprecated := range cpd {
		if isDeprecated {
			err := r.Get(r.ctx, types.NamespacedName{Namespace: trigger.Namespace, Name: componentName}, &component)
			if err != nil {
				if util.IsNotFound(err) {
					continue

				} else {
					log.Error(err, "Failed to get deprecated component", "namespace", trigger.Namespace, "name", componentName)
					return err
				}
			}
			if err := r.Delete(r.ctx, &component); err != nil {
				log.Error(err, "Failed to delete deprecated component", "namespace", trigger.Namespace, "name", componentName)
				return err
			}
		}
	}
	return nil
}

func (r *TriggerReconciler) handleDeprecatedWorkloads(trigger *openfunctionevent.Trigger, wpd map[string]bool) error {
	log := r.Log.WithName("handleDeprecatedWorkloads")
	var workload appsv1.Deployment
	for workloadName, isDeprecated := range wpd {
		if isDeprecated {
			err := r.Get(r.ctx, types.NamespacedName{Namespace: trigger.Namespace, Name: workloadName}, &workload)
			if err != nil {
				if util.IsNotFound(err) {
					continue

				} else {
					log.Error(err, "Failed to get deprecated workload", "namespace", trigger.Namespace, "name", workloadName)
					return err
				}
			}
			if err := r.Delete(r.ctx, &workload); err != nil {
				log.Error(err, "Failed to delete deprecated workload", "namespace", trigger.Namespace, "name", workloadName)
				return err
			}
		}
	}
	return nil
}

func (r *TriggerReconciler) createOrUpdateTriggerWorkload(trigger *openfunctionevent.Trigger, workloadsPendingDelete map[string]bool) (runtime.Object, error) {
	log := r.Log.WithName("createOrUpdateTriggerWorkload")

	obj := &appsv1.Deployment{}

	workloadName := fmt.Sprintf(TriggerWorkloadsNameTmpl, trigger.Name)
	accessor, _ := meta.Accessor(obj)
	accessor.SetName(workloadName)
	accessor.SetNamespace(trigger.Namespace)
	if workloadsPendingDelete[workloadName] {
		workloadsPendingDelete[workloadName] = false
	}

	_, err := controllerutil.CreateOrUpdate(r.ctx, r.Client, obj, r.mutateHandler(obj, trigger))
	if err != nil {
		log.Error(err, "Failed to create or update trigger handler", "namespace", trigger.Namespace, "name", trigger.Name)
	}

	log.V(1).Info("Create trigger handler", "namespace", trigger.Namespace, "name", trigger.Name)
	return obj, nil
}

func (r *TriggerReconciler) mutateHandler(obj runtime.Object, trigger *openfunctionevent.Trigger) controllerutil.MutateFn {
	return func() error {
		accessor, _ := meta.Accessor(obj)
		workloadLabels := map[string]string{
			"openfunction.io/managed": "true",
			TriggerControlledLabel:    trigger.Name,
		}
		accessor.SetLabels(workloadLabels)

		selector := &metav1.LabelSelector{
			MatchLabels: workloadLabels,
		}

		var replicas int32 = 1

		var port int32 = 5050

		annotations := make(map[string]string)
		annotations["dapr.io/enabled"] = "true"
		annotations["dapr.io/app-id"] = fmt.Sprintf("%s-%s-handler", strings.TrimSuffix(trigger.Name, "-trigger"), trigger.Namespace)
		annotations["dapr.io/log-as-json"] = "true"
		annotations["dapr.io/app-protocol"] = "grpc"
		annotations["dapr.io/app-port"] = fmt.Sprintf("%d", port)

		spec := &corev1.PodSpec{}

		envEncode, err := r.envs.EncodeConfig()
		if err != nil {
			return err
		}
		container := &corev1.Container{
			Name:            triggerContainerName,
			Image:           triggerHandlerImage,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Env: []corev1.EnvVar{
				{Name: "CONFIG", Value: envEncode},
			},
		}

		spec.Containers = []corev1.Container{*container}

		template := corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: annotations,
				Labels:      workloadLabels,
			},
			Spec: *spec,
		}

		switch obj.(type) {
		case *appsv1.Deployment:
			deploy := obj.(*appsv1.Deployment)
			deploy.Spec.Selector = selector
			deploy.Spec.Replicas = &replicas
			deploy.Spec.Template = template
		}

		return controllerutil.SetControllerReference(trigger, accessor, r.Scheme)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *TriggerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openfunctionevent.Trigger{}).
		Owns(&componentsv1alpha1.Component{}).
		Owns(&appsv1.Deployment{}).
		Watches(&source.Kind{Type: &openfunctionevent.EventBus{}}, handler.EnqueueRequestsFromMapFunc(func(object client.Object) []reconcile.Request {
			triggerList := &openfunctionevent.TriggerList{}
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
		Watches(&source.Kind{Type: &openfunctionevent.ClusterEventBus{}}, handler.EnqueueRequestsFromMapFunc(func(object client.Object) []reconcile.Request {
			triggerList := &openfunctionevent.TriggerList{}
			c := mgr.GetClient()

			err := c.List(context.TODO(), triggerList)
			if err != nil {
				return []reconcile.Request{}
			}

			reconcileRequests := make([]reconcile.Request, len(triggerList.Items))
			for _, trigger := range triggerList.Items {
				if &trigger != nil {
					var eventBus openfunctionevent.EventBus
					if err := c.Get(context.TODO(), client.ObjectKey{Namespace: trigger.Namespace, Name: trigger.Spec.EventBus}, &eventBus); util.IgnoreNotFound(err) != nil {
						continue
					}
					if &eventBus != nil {
						continue
					}
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
		Complete(r)
}
