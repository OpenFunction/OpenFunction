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
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	openfunctionevent "github.com/openfunction/apis/events/v1alpha1"
	"github.com/openfunction/pkg/util"
)

const (
	handlerContainerName    = "eventsource"
	eventSourceHandlerImage = "openfunctiondev/eventsource-handler:latest"
)

// EventSourceReconciler reconciles a EventSource object
type EventSourceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	ctx    context.Context
	envs   *EventSourceConfig
}

//+kubebuilder:rbac:groups=events.openfunction.io,resources=eventsources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=events.openfunction.io,resources=eventsources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=events.openfunction.io,resources=eventsources/finalizers,verbs=update
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
	r.ctx = ctx
	log := r.Log.WithValues("EventSource", req.NamespacedName)
	log.Info("eventsource reconcile starting...")

	var eventSource openfunctionevent.EventSource
	r.envs = &EventSourceConfig{}

	if err := r.Get(ctx, req.NamespacedName, &eventSource); err != nil {
		log.V(1).Info("EventSource deleted", "error", err)
		return ctrl.Result{}, util.IgnoreNotFound(err)
	}

	if _, err := r.createOrUpdateEventSource(&eventSource); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// createOrUpdateEventSource will do:
// 1. Generate a dapr component specification for the EventBus associated with the EventSource (if spec.eventBus is set)
//    and create the dapr component (will check if it needs to be updated)
//    and set the SourceConfig.EventBusComponentName, SourceConfig.EventBusTopic, SourceConfig.EventBusSpecEncode.
// 2. Generate a dapr component specification for the Sink set in EventSource (if spec.sink is set)
//    and create the dapr component (will check if it needs to be updated)
//    and set the SourceConfig.SinkComponentName, SourceConfig.SinkSpecEncode.
// 3. Generate dapr component specifications for the event sources
//    and create the dapr components (will check if they need to be updated)
//    and set the SourceConfig.EventSourceComponentName, SourceConfig.EventSourceTopic, SourceConfig.EventSourceSpecEncode.
// 4. Generate SourceConfig and convert it to a base64-encoded string.
// 5. Create an EventSource workload for each event source (will check if they need to be updated)
//    and pass in the SourceConfig as an environment variable.
func (r *EventSourceReconciler) createOrUpdateEventSource(eventSource *openfunctionevent.EventSource) (ctrl.Result, error) {
	log := r.Log.WithName("createOrUpdateEventSource")

	if eventSource.Spec.EventBus == "" && eventSource.Spec.Sink == nil {
		err := errors.New("no output configuration (eventBus or sink) found ")
		log.Error(err, "Failed to find output configuration (eventBus or sink).", "namespace", eventSource.Namespace, "name", eventSource.Name)
		return ctrl.Result{}, err
	}

	// Get all exist components and workloads owned by the EventSource and mark them with status of pending deletion (set to true)
	// In the later reconcile, the resources that still need to be kept are set to non-pending deletion status (set to false) based on the latest list of resources to be created
	var existComponents componentsv1alpha1.ComponentList
	var existWorkloads appsv1.DeploymentList
	controlledLabelSelector := labels.SelectorFromSet(labels.Set(map[string]string{EventSourceControlledLabel: eventSource.Name}))

	if err := r.List(r.ctx, &existComponents, &client.ListOptions{LabelSelector: controlledLabelSelector}); err != nil {
		log.Error(err, "Failed to retrieve exist controlled components", "namespace", eventSource.Namespace, "name", eventSource.Name)
		return ctrl.Result{}, err
	}
	componentsPendingDelete := map[string]bool{}
	for _, component := range existComponents.Items {
		c := component
		componentsPendingDelete[c.Name] = true
	}

	if err := r.List(r.ctx, &existWorkloads, &client.ListOptions{LabelSelector: controlledLabelSelector}); err != nil {
		log.Error(err, "Failed to retrieve exist controlled workloads", "namespace", eventSource.Namespace, "name", eventSource.Name)
		return ctrl.Result{}, err
	}
	workloadsPendingDelete := map[string]bool{}
	for _, workload := range existWorkloads.Items {
		w := workload
		workloadsPendingDelete[w.Name] = true
	}

	// Handle EventBus reconcile.
	if eventSource.Spec.EventBus != "" {
		if err := r.handleEventBus(eventSource, componentsPendingDelete); err != nil {
			log.Error(err, "Failed to handle EventBus", "namespace", eventSource.Namespace, "name", eventSource.Name)
			return ctrl.Result{}, err
		}
	}

	// Handle Sink reconcile.
	if eventSource.Spec.Sink != nil {
		if err := r.handleSink(eventSource, componentsPendingDelete); err != nil {
			log.Error(err, "Failed to handle Sink", "namespace", eventSource.Namespace, "name", eventSource.Name)
			return ctrl.Result{}, err
		}
	}

	// Handle EventSource reconcile.
	if err := r.handleEventSource(eventSource, componentsPendingDelete, workloadsPendingDelete); err != nil {
		log.Error(err, "Failed to handle EventSource", "namespace", eventSource.Namespace, "name", eventSource.Name)
		return ctrl.Result{}, err
	}

	// Clean up resources to be deprecated
	if err := r.handleDeprecatedComponents(eventSource, componentsPendingDelete); err != nil {
		log.Error(err, "Failed to handle deprecated components", "namespace", eventSource.Namespace, "name", eventSource.Name)
		return ctrl.Result{}, err
	}

	if err := r.handleDeprecatedWorkloads(eventSource, workloadsPendingDelete); err != nil {
		log.Error(err, "Failed to handle deprecated workloads", "namespace", eventSource.Namespace, "name", eventSource.Name)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *EventSourceReconciler) handleEventBus(eventSource *openfunctionevent.EventSource, componentsPendingDelete map[string]bool) error {

	// Retrieve the specification of EventBus associated with the EventSource
	var eventBusSpec openfunctionevent.EventBusSpec
	eventBus := retrieveEventBus(r.ctx, r.Client, eventSource.Namespace, eventSource.Spec.EventBus)
	if eventBus == nil {
		clusterEventBus := retrieveClusterEventBus(r.ctx, r.Client, eventSource.Spec.EventBus)
		if clusterEventBus == nil {
			return errors.New("cannot retrieve eventBus and clusterEventBus")
		} else {
			eventBusSpec = clusterEventBus.Spec
		}
	} else {
		eventBusSpec = eventBus.Spec
	}

	componentName := fmt.Sprintf(EventSourceBusComponentNameTmpl, eventSource.Name)
	if componentsPendingDelete[componentName] {
		componentsPendingDelete[componentName] = false
	}

	// Set SourceConfig.EventBusComponentName.
	r.envs.EventBusComponentName = componentName

	// Generate a dapr component based on the specification of EventBus.
	component := &componentsv1alpha1.Component{
		ObjectMeta: metav1.ObjectMeta{
			Name:      componentName,
			Namespace: eventSource.Namespace,
		},
	}

	if eventBusSpec.Nats != nil {
		specBytes, err := json.Marshal(eventBusSpec.Nats)
		if err != nil {
			return err
		}

		// Set the SourceConfig.EventBusSpecEncode which will reflect the specification content changes of dapr component back to the EventSource workload,
		// informing it that it needs to rebuild.
		r.envs.EventBusSpecEncode = base64.StdEncoding.EncodeToString(specBytes)

		// Create the dapr component for EventSource to send event to EventBus.
		component.Spec = *eventBusSpec.Nats
		if _, err := ctrl.CreateOrUpdate(r.ctx, r.Client, component, mutateDaprComponent(r.Scheme, component, eventSource)); err != nil {
			return err
		}
		return nil
	}

	return errors.New("no specification found for create dapr component")
}

func (r *EventSourceReconciler) handleSink(eventSource *openfunctionevent.EventSource, cpd map[string]bool) error {
	componentName := fmt.Sprintf(EventSourceSinkComponentNameTmpl, eventSource.Name)
	if cpd[componentName] {
		cpd[componentName] = false
	}

	// Set SourceConfig.SinkComponentName.
	r.envs.SinkComponentName = componentName

	// Generate a dapr component based on the specification of Sink.
	component := &componentsv1alpha1.Component{
		ObjectMeta: metav1.ObjectMeta{
			Name:      componentName,
			Namespace: eventSource.Namespace,
		},
	}

	spec, err := newSinkComponentSpec(r.Client, r.Log, eventSource.Spec.Sink.Ref)
	if err != nil {
		return err
	}
	component.Spec = *spec
	specBytes, err := json.Marshal(spec)
	if err != nil {
		return err
	}

	// Set the SourceConfig.SinkSpecEncode which will reflect the specification content changes of dapr component back to the EventSource workload,
	// informing it that it needs to rebuild.
	r.envs.SinkSpecEncode = base64.StdEncoding.EncodeToString(specBytes)

	// Create the dapr component for EventSource to retrieve event from EventBus.
	if _, err := ctrl.CreateOrUpdate(r.ctx, r.Client, component, mutateDaprComponent(r.Scheme, component, eventSource)); err != nil {
		return err
	}

	return nil
}

func (r *EventSourceReconciler) handleEventSource(eventSource *openfunctionevent.EventSource, componentsPendingDelete map[string]bool, workloadsPendingDelete map[string]bool) error {
	type sourceSpec struct {
		*componentsv1alpha1.Component
		SourceTopic string
		SourceKind  string
		EventName   string
	}
	var sourceSpecs []*sourceSpec

	// Generate dapr components based on the specification of EventSource.
	if eventSource.Spec.Kafka != nil {
		for eventName, spec := range eventSource.Spec.Kafka {
			ss := &sourceSpec{}
			componentSpec := spec.ComponentSpec
			componentName := fmt.Sprintf(EventSourceComponentNameTmpl, eventSource.Name, SourceKindKafka, eventName)

			// We need to assign a separate consumerGroup name to each kafka component
			var newMetadataItem []componentsv1alpha1.MetadataItem
			consumerGroup := map[string]string{"name": "consumerGroup", "value": fmt.Sprintf("%s-%s", eventSource.Namespace, componentName)}
			consumerGroupBytes, err := json.Marshal(consumerGroup)
			if err != nil {
				return err
			}
			var consumerGroupItem componentsv1alpha1.MetadataItem
			err = json.Unmarshal(consumerGroupBytes, &consumerGroupItem)
			if err != nil {
				return err
			}

			found := false
			for _, md := range componentSpec.Metadata {
				if md.Name == "consumerGroup" {
					newMetadataItem = append(newMetadataItem, consumerGroupItem)
					found = true
					continue
				}
				newMetadataItem = append(newMetadataItem, md)
			}
			if !found {
				newMetadataItem = append(newMetadataItem, consumerGroupItem)
			}
			componentSpec.Metadata = newMetadataItem

			component := &componentsv1alpha1.Component{
				ObjectMeta: metav1.ObjectMeta{
					Name:      componentName,
					Namespace: eventSource.Namespace,
				},
			}
			component.Spec = *componentSpec
			ss.Component = component
			ss.SourceKind = SourceKindKafka
			ss.EventName = eventName
			if spec.SourceTopic != "" {
				ss.SourceTopic = spec.SourceTopic
			}
			sourceSpecs = append(sourceSpecs, ss)
		}
	}

	if eventSource.Spec.Redis != nil {
		for eventName, spec := range eventSource.Spec.Cron {
			ss := &sourceSpec{}
			componentSpec := spec.ComponentSpec
			componentName := fmt.Sprintf(EventSourceComponentNameTmpl, eventSource.Name, SourceKindRedis, eventName)
			component := &componentsv1alpha1.Component{
				ObjectMeta: metav1.ObjectMeta{
					Name:      componentName,
					Namespace: eventSource.Namespace,
				},
			}
			component.Spec = *componentSpec
			ss.Component = component
			ss.SourceKind = SourceKindRedis
			ss.EventName = eventName
			sourceSpecs = append(sourceSpecs, ss)
		}
	}

	if eventSource.Spec.Cron != nil {
		for eventName, spec := range eventSource.Spec.Cron {
			ss := &sourceSpec{}
			componentSpec := spec.ComponentSpec
			componentName := fmt.Sprintf(EventSourceComponentNameTmpl, eventSource.Name, SourceKindCron, eventName)
			component := &componentsv1alpha1.Component{
				ObjectMeta: metav1.ObjectMeta{
					Name:      componentName,
					Namespace: eventSource.Namespace,
				},
			}
			component.Spec = *componentSpec
			ss.Component = component
			ss.SourceKind = SourceKindCron
			ss.EventName = eventName
			sourceSpecs = append(sourceSpecs, ss)
		}
	}

	for _, ss := range sourceSpecs {
		component := ss.Component
		spec := &component.Spec

		if componentsPendingDelete[component.Name] {
			componentsPendingDelete[component.Name] = false
		}
		r.envs.EventSourceComponentName = component.Name
		r.envs.EventSourceTopic = ss.SourceTopic
		r.envs.EventBusTopic = fmt.Sprintf(EventBusTopicNameTmpl, eventSource.Namespace, eventSource.Name, ss.EventName)
		specBytes, err := json.Marshal(spec)
		if err != nil {
			return err
		}

		// Set the SourceConfig.EventSourceSpecEncode which will reflect the specification content changes of dapr components back to the EventSource workload,
		// informing it that it needs to rebuild.
		r.envs.EventSourceSpecEncode = base64.StdEncoding.EncodeToString(specBytes)

		// Create the dapr component for EventSource to retrieve event from EventBus.
		if _, err := ctrl.CreateOrUpdate(r.ctx, r.Client, component, mutateDaprComponent(r.Scheme, component, eventSource)); err != nil {
			return err
		}

		// Create the workload for EventSource.
		if _, err := r.createOrUpdateEventSourceWorkload(eventSource, ss.SourceKind, ss.EventName, workloadsPendingDelete); err != nil {
			return err
		}
	}
	return nil
}

func (r *EventSourceReconciler) handleDeprecatedComponents(eventSource *openfunctionevent.EventSource, cpd map[string]bool) error {
	log := r.Log.WithName("handleDeprecatedComponents")
	var component componentsv1alpha1.Component
	for componentName, isDeprecated := range cpd {
		if isDeprecated {
			err := r.Get(r.ctx, types.NamespacedName{Namespace: eventSource.Namespace, Name: componentName}, &component)
			if err != nil {
				if util.IsNotFound(err) {
					continue

				} else {
					log.Error(err, "Failed to get deprecated component", "namespace", eventSource.Namespace, "name", componentName)
					return err
				}
			}
			if err := r.Delete(r.ctx, &component); err != nil {
				log.Error(err, "Failed to delete deprecated component", "namespace", eventSource.Namespace, "name", componentName)
				return err
			}
		}
	}
	return nil
}

func (r *EventSourceReconciler) handleDeprecatedWorkloads(eventSource *openfunctionevent.EventSource, wpd map[string]bool) error {
	log := r.Log.WithName("handleDeprecatedWorkloads")
	var workload appsv1.Deployment
	for workloadName, isDeprecated := range wpd {
		if isDeprecated {
			err := r.Get(r.ctx, types.NamespacedName{Namespace: eventSource.Namespace, Name: workloadName}, &workload)
			if err != nil {
				if util.IsNotFound(err) {
					continue

				} else {
					log.Error(err, "Failed to get deprecated workload", "namespace", eventSource.Namespace, "name", workloadName)
					return err
				}
			}
			if err := r.Delete(r.ctx, &workload); err != nil {
				log.Error(err, "Failed to delete deprecated workload", "namespace", eventSource.Namespace, "name", workloadName)
				return err
			}
		}
	}
	return nil
}

func (r *EventSourceReconciler) createOrUpdateEventSourceWorkload(eventSource *openfunctionevent.EventSource, sourceKind string, eventName string, workloadsPendingDelete map[string]bool) (runtime.Object, error) {
	log := r.Log.WithName("createOrUpdateEventSourceWorkload")

	obj := &appsv1.Deployment{}

	workloadName := fmt.Sprintf(EventSourceWorkloadsNameTmpl, eventSource.Name, sourceKind, eventName)
	accessor, _ := meta.Accessor(obj)
	accessor.SetName(workloadName)
	accessor.SetNamespace(eventSource.Namespace)
	if workloadsPendingDelete[workloadName] {
		workloadsPendingDelete[workloadName] = false
	}

	_, err := controllerutil.CreateOrUpdate(r.ctx, r.Client, obj, r.mutateHandler(obj, eventSource))
	if err != nil {
		log.Error(err, "Failed to create or update eventsource handler", "namespace", eventSource.Namespace, "name", eventSource.Name)
	}

	log.V(1).Info("Create eventsource handler", "namespace", eventSource.Namespace, "name", eventSource.Name)
	return obj, nil
}

func (r *EventSourceReconciler) mutateHandler(obj runtime.Object, eventSource *openfunctionevent.EventSource) controllerutil.MutateFn {
	return func() error {
		accessor, _ := meta.Accessor(obj)
		workloadLabels := map[string]string{
			"openfunction.io/managed":  "true",
			EventSourceControlledLabel: eventSource.Name,
		}
		accessor.SetLabels(workloadLabels)

		selector := &metav1.LabelSelector{
			MatchLabels: workloadLabels,
		}

		var replicas int32 = 1

		var port int32 = 5050

		annotations := make(map[string]string)
		annotations["dapr.io/enabled"] = "true"
		annotations["dapr.io/app-id"] = fmt.Sprintf("%s-%s-handler", strings.TrimSuffix(eventSource.Name, "-serving"), eventSource.Namespace)
		annotations["dapr.io/log-as-json"] = "true"
		annotations["dapr.io/app-protocol"] = "grpc"
		annotations["dapr.io/app-port"] = fmt.Sprintf("%d", port)

		spec := &corev1.PodSpec{}

		envEncode, err := r.envs.EncodeConfig()
		if err != nil {
			return err
		}
		container := &corev1.Container{
			Name:            handlerContainerName,
			Image:           eventSourceHandlerImage,
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

		accessor.SetOwnerReferences(nil)
		return controllerutil.SetControllerReference(eventSource, accessor, r.Scheme)
	}
}

func (r *EventSourceReconciler) updateStatus(eventSource *openfunctionevent.EventSource, status *openfunctionevent.EventSourceStatus) error {

	status.DeepCopyInto(&eventSource.Status)
	if err := r.Status().Update(r.ctx, eventSource); err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EventSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openfunctionevent.EventSource{}).
		Owns(&componentsv1alpha1.Component{}).
		Owns(&appsv1.Deployment{}).
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

			err := c.List(context.TODO(), eventSourceList)
			if err != nil {
				return []reconcile.Request{}
			}

			reconcileRequests := make([]reconcile.Request, len(eventSourceList.Items))
			for _, eventSource := range eventSourceList.Items {
				if &eventSource != nil {
					if eventSource.Spec.EventBus != "" {
						var eventBus openfunctionevent.EventBus
						if err := c.Get(context.TODO(), client.ObjectKey{Namespace: eventSource.Namespace, Name: eventSource.Spec.EventBus}, &eventBus); util.IgnoreNotFound(err) != nil {
							continue
						}
						if &eventBus != nil {
							continue
						}
						if eventSource.Spec.EventBus == object.GetName() {
							reconcileRequests = append(reconcileRequests, reconcile.Request{
								NamespacedName: types.NamespacedName{
									Namespace: eventSource.Namespace,
									Name:      eventSource.Name,
								},
							})
						}
					}
				}
			}
			return reconcileRequests
		})).
		Complete(r)
}
