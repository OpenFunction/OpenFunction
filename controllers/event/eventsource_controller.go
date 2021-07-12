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
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	openfunctionevent "github.com/openfunction/apis/event/v1alpha1"
	"github.com/openfunction/controllers/event/adapters"
	"github.com/openfunction/pkg/util"
)

const (
	handlerContainerName    = "handler"
	eventSourceHandlerImage = "openfunctiondev/eventsource-handler:latest"
)

// EventSourceReconciler reconciles a EventSource object
type EventSourceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	ctx    context.Context
	envs   *SourceEnvConfig
}

//+kubebuilder:rbac:groups=event.openfunction.io,resources=eventsources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=event.openfunction.io,resources=eventsources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=event.openfunction.io,resources=eventsources/finalizers,verbs=update
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

	var es openfunctionevent.EventSource
	r.envs = &SourceEnvConfig{}

	if err := r.Get(ctx, req.NamespacedName, &es); err != nil {
		log.V(1).Info("EventSource deleted", "error", err)
		return ctrl.Result{}, util.IgnoreNotFound(err)
	}

	if _, err := r.createOrUpdateEventSource(&es); err != nil {
		log.Error(err, "Failed")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// 1. Retrieve the configuration of the event bus based on the value of spec.eventBusNames (Used to connect event sources to event bus)
// 2. Generate dapr component specs (source and sink) for the event source
// 3. Create dapr component by using specs above
// 4. Generate an event source configuration SourceEnvConfig
// 5. Create event source handler workload (pass SourceEnvConfig in)
func (r *EventSourceReconciler) createOrUpdateEventSource(es *openfunctionevent.EventSource) (ctrl.Result, error) {
	log := r.Log.WithName("createOrUpdate")

	// Retrieve the configuration of the event bus based on the value of spec.eventBusNames
	eventBusComponentConfigs, err := r.retrieveEventBusConfigs(es)
	if err != nil {
		log.Error(err, "Failed to retrieve event bus component names", "namespace", es.Namespace, "name", es.Name)
		return ctrl.Result{}, err
	}

	// Converts the event source specification into a connector
	// containing the specification for creating the dapr component.
	connectors, err := adapters.NewEventSourceConnectors(es)
	if err != nil {
		log.Error(err, "Failed to generate connectors")
		return ctrl.Result{}, err
	}

	// Create dapr component to adapter the event source sink
	sinkComponentName, err := r.createOrUpdateSink(es)
	if err != nil {
		log.Error(err, "Failed to create sink", "namespace", es.Namespace, "name", es.Name)
	}

	// At least a EventBus exists, or spec.sink needs to be configured in the EventSource for the EventSource to work properly
	// If spec.eventBusName is not set, then the EventBus named "default" will be checked to see if it exists
	if sinkComponentName == "" && eventBusComponentConfigs == nil {
		log.Error(err, "Failed to find eventbus nor sink", "namespace", es.Namespace, "name", es.Name)
		return ctrl.Result{}, err
	}

	// Create a one-to-one workload for each connector
	for _, c := range connectors {
		// Create dapr component to adapter the event source
		if err := CreateOrUpdateDaprComponent(r.Client, r.Scheme, r.Log, c, es); err != nil {
			log.Error(err, "Failed to create dapr component", "namespace", es.Namespace, "name", es.Name)
			return ctrl.Result{}, err
		}

		r.envs.SourceComponentName = c.Component.Name
		if es.Spec.Topic != "" {
			r.envs.SourceTopic = es.Spec.Topic
		}

		// Create event source handler workload
		if _, err := r.createOrUpdateEventSourceHandler(es, c.Component.Name); err != nil {
			log.Error(err, "Failed to create eventsource handler", "namespace", es.Namespace, "name", es.Name)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *EventSourceReconciler) retrieveEventBusConfigs(es *openfunctionevent.EventSource) ([]BusConfig, error) {
	log := r.Log.WithName("retrieveEventBusConfigs")

	var eb openfunctionevent.EventBus
	var eventBusConfigs []BusConfig
	es.Spec.EventBusNames = append(es.Spec.EventBusNames, DefaultEventBusName)

	// Need to remove duplicate name of event bus
	eventBuses := r.removeRepeatedElement(es.Spec.EventBusNames)
	for _, eventBus := range eventBuses {
		key := client.ObjectKey{Namespace: es.Namespace, Name: eventBus}
		err := r.Get(r.ctx, key, &eb)
		if err != nil {
			if e := util.IgnoreNotFound(err); e != nil {
				log.Error(err, "Failed to find eventbus", "namespace", es.Namespace, "name", es.Name)
				return nil, err
			}
			continue
		} else {
			// TODO: Check the status of the EventBus
			ebc := BusConfig{}
			if componentName, ok := eb.GetAnnotations()["component-name"]; ok {
				ebc.BusComponentName = componentName
			}
			ebc.BusTopic = eb.Spec.Topic
			eventBusConfigs = append(eventBusConfigs, ebc)
		}
	}
	if eventBusConfigs != nil {
		r.envs.BusConfigs = eventBusConfigs
	}
	return eventBusConfigs, nil
}

func (r *EventSourceReconciler) createOrUpdateSink(es *openfunctionevent.EventSource) (string, error) {
	log := r.Log.WithName("createOrUpdateSink")

	if es.Spec.Sink != nil {
		sinkConnector, err := adapters.NewEventSourceSinkConnectors(r.Client, r.Log, es)
		if err != nil {
			log.Error(err, "Failed to create event source sink connector", "namespace", es.Namespace, "name", es.Name)
			return "", err
		}

		sinkComponent := sinkConnector.Component

		if err = r.Get(r.ctx, client.ObjectKey{Namespace: sinkComponent.Namespace, Name: sinkComponent.Name}, &sinkComponent); util.IsNotFound(err) {
			log.Info("Need create dapr component", "namespace", es.Namespace, "name", es.Name)

			if err = CreateOrUpdateDaprComponent(r.Client, r.Scheme, r.Log, sinkConnector, es); err != nil {
				log.Error(err, "Failed to create dapr component", "namespace", es.Namespace, "name", es.Name)
				return "", err
			}
		}

		r.envs.SinkComponentName = sinkComponent.Name
		return sinkComponent.Name, nil
	}

	return "", nil
}

func (r *EventSourceReconciler) createOrUpdateEventSourceHandler(es *openfunctionevent.EventSource, sourceComponentName string) (runtime.Object, error) {
	log := r.Log.WithName("createOrUpdateEventSourceHandler")

	handler := &appsv1.Deployment{}

	accessor, _ := meta.Accessor(handler)
	accessor.SetName(es.Name)
	accessor.SetNamespace(es.Namespace)

	if err := r.Delete(r.ctx, handler); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to delete eventsource handler", "name", es.Name, "namespace", es.Namespace)
		return nil, err
	}

	if err := r.mutateHandler(handler, es)(); err != nil {
		log.Error(err, "Failed to mutate eventsource handler", "name", es.Name, "namespace", es.Namespace)
		return nil, err
	}

	if err := r.Create(r.ctx, handler); err != nil {
		log.Error(err, "Failed to create eventsource handler", "name", es.Name, "namespace", es.Namespace)
		return nil, err
	}

	log.V(1).Info("Create eventsource handler", "name", es.Name, "namespace", es.Namespace)
	return handler, nil
}

func (r *EventSourceReconciler) mutateHandler(obj runtime.Object, es *openfunctionevent.EventSource) controllerutil.MutateFn {
	return func() error {

		accessor, _ := meta.Accessor(obj)
		labels := map[string]string{
			"openfunction.io/managed": "true",
		}
		accessor.SetLabels(labels)

		selector := &metav1.LabelSelector{
			MatchLabels: labels,
		}

		var replicas int32 = 1

		var port int32 = 5050

		annotations := make(map[string]string)
		annotations["dapr.io/enabled"] = "true"
		annotations["dapr.io/app-id"] = fmt.Sprintf("%s-%s-handler", strings.TrimSuffix(es.Name, "-serving"), es.Namespace)
		annotations["dapr.io/log-as-json"] = "true"
		annotations["dapr.io/app-protocol"] = "grpc"
		annotations["dapr.io/app-port"] = fmt.Sprintf("%d", port)

		spec := &corev1.PodSpec{}

		envConfigEncode, err := r.envs.EncodeEnvConfig()
		if err != nil {
			return err
		}
		container := &corev1.Container{
			Name:            handlerContainerName,
			Image:           eventSourceHandlerImage,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Env: []corev1.EnvVar{
				{Name: "CONFIG", Value: envConfigEncode},
			},
		}

		spec.Containers = []corev1.Container{*container}

		template := corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: annotations,
				Labels:      labels,
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

		return controllerutil.SetControllerReference(es, accessor, r.Scheme)
	}
}

func (r *EventSourceReconciler) removeRepeatedElement(arr []string) (newArr []string) {
	newArr = make([]string, 0)
	for i := 0; i < len(arr); i++ {
		repeat := false
		for j := i + 1; j < len(arr); j++ {
			if arr[i] == arr[j] {
				repeat = true
				break
			}
		}
		if !repeat {
			newArr = append(newArr, arr[i])
		}
	}
	return
}

func (r *EventSourceReconciler) updateHash(es *openfunctionevent.EventSource, key string, val interface{}) error {

	if es.Annotations == nil {
		es.Annotations = make(map[string]string)
	}

	es.Annotations[key] = util.Hash(val)

	if err := r.Update(r.ctx, es); err != nil {
		return err
	}

	return nil
}

func (r *EventSourceReconciler) getHash(es *openfunctionevent.EventSource, key string) string {

	if es.Annotations == nil {
		return ""
	}

	return es.Annotations[key]
}

// SetupWithManager sets up the co ntroller with the Manager.
func (r *EventSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openfunctionevent.EventSource{}).
		Complete(r)
}
