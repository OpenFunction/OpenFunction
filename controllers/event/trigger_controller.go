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
	"encoding/json"
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
	triggerContainerName = "trigger"
	triggerHandlerImage  = "openfunctiondev/trigger-handler:latest"
)

// TriggerReconciler reconciles a Trigger object
type TriggerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	ctx    context.Context
	envs   *TriggerEnvConfig
}

type Subscribers struct {
	Sinks            []*openfunctionevent.SinkSpec `json:"sinks,omitempty"`
	DeadLetterSinks  []*openfunctionevent.SinkSpec `json:"deadLetterSinks,omitempty"`
	TotalSinks       []*openfunctionevent.SinkSpec `json:"totalSinks,omitempty"`
	Topics           []string                      `json:"topics,omitempty"`
	DeadLetterTopics []string                      `json:"deadLetterTopics,omitempty"`
	TotalTopics      []string                      `json:"totalTopics,omitempty"`
}

//+kubebuilder:rbac:groups=event.openfunction.io,resources=triggers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=event.openfunction.io,resources=triggers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=event.openfunction.io,resources=triggers/finalizers,verbs=update

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

	if err := r.Get(ctx, req.NamespacedName, &trigger); err != nil {
		log.V(1).Info("Trigger deleted", "error", err)
		return ctrl.Result{}, util.IgnoreNotFound(err)
	}

	if _, err := r.createOrUpdateTrigger(&trigger); err != nil {
		log.Error(err, "Failed")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// 1. Create dapr component spec for event bus
// 2. Parse subscribers, create dapr component specs for sink and deadLetterSink
// 3. Create dapr component by using specs above
// 4. Generate an trigger configuration TriggerEnvConfig
// 5. Create trigger handler workload (pass TriggerEnvConfig in)
func (r *TriggerReconciler) createOrUpdateTrigger(t *openfunctionevent.Trigger) (ctrl.Result, error) {
	log := r.Log.WithName("createOrUpdate")

	// Converts the event bus specification into a connector
	// containing the specification for creating the dapr component.
	connectors, triggerEnvConfigJson, err := adapters.NewTriggerConnectors(r.Client, r.Log, t)
	if err != nil {
		log.Error(err, "Failed to generate connectors")
		return ctrl.Result{}, err
	}

	err = json.Unmarshal(triggerEnvConfigJson, &r.envs)
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, c := range connectors {
		// Create dapr component for trigger sink and deadLetterSink
		if err := CreateOrUpdateDaprComponent(r.Client, r.Scheme, r.Log, c, t); err != nil {
			log.Error(err, "Failed to create dapr component", "namespace", t.Namespace, "name", t.Name)
			return ctrl.Result{}, err
		}
	}

	// Create trigger handler workload
	if _, err := r.createOrUpdatetTriggerHandler(t); err != nil {
		log.Error(err, "Failed to create trigger handler", "namespace", t.Namespace, "name", t.Name)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *TriggerReconciler) createOrUpdatetTriggerHandler(t *openfunctionevent.Trigger) (runtime.Object, error) {
	log := r.Log.WithName("createOrUpdateEventSourceHandler")

	handler := &appsv1.Deployment{}

	accessor, _ := meta.Accessor(handler)
	accessor.SetName(t.Name)
	accessor.SetNamespace(t.Namespace)

	if err := r.Delete(r.ctx, handler); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to delete trigger handler", "name", t.Name, "namespace", t.Namespace)
		return nil, err
	}

	if err := r.mutateHandler(handler, t)(); err != nil {
		log.Error(err, "Failed to mutate trigger handler", "name", t.Name, "namespace", t.Namespace)
		return nil, err
	}

	if err := r.Create(r.ctx, handler); err != nil {
		log.Error(err, "Failed to create eventsource handler", "name", t.Name, "namespace", t.Namespace)
		return nil, err
	}

	log.V(1).Info("Create eventsource handler", "name", t.Name, "namespace", t.Namespace)
	return handler, nil
}

func (r *TriggerReconciler) mutateHandler(obj runtime.Object, t *openfunctionevent.Trigger) controllerutil.MutateFn {
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
		annotations["dapr.io/app-id"] = fmt.Sprintf("%s-%s-handler", strings.TrimSuffix(t.Name, "-trigger"), t.Namespace)
		annotations["dapr.io/log-as-json"] = "true"
		annotations["dapr.io/app-protocol"] = "grpc"
		annotations["dapr.io/app-port"] = fmt.Sprintf("%d", port)

		spec := &corev1.PodSpec{}

		envConfigEncode, err := r.envs.EncodeEnvConfig()
		if err != nil {
			return err
		}
		container := &corev1.Container{
			Name:            triggerContainerName,
			Image:           triggerHandlerImage,
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

		return controllerutil.SetControllerReference(t, accessor, r.Scheme)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *TriggerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openfunctionevent.Trigger{}).
		Complete(r)
}
