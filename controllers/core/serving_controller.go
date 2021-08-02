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

package core

import (
	"context"
	"fmt"
	"strings"

	"github.com/openfunction/pkg/util"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kservingv1 "knative.dev/serving/pkg/apis/serving/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	openfunction "github.com/openfunction/apis/core/v1alpha1"
)

// ServingReconciler reconciles a Serving object
type ServingReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	ctx    context.Context
}

//+kubebuilder:rbac:groups=core.openfunction.io,resources=servings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.openfunction.io,resources=servings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.openfunction.io,resources=servings/finalizers,verbs=update
//+kubebuilder:rbac:groups=serving.knative.dev,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dapr.io,resources=components;subscriptions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keda.sh,resources=scaledjobs;scaledobjects,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Serving object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ServingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.ctx = ctx
	log := r.Log.WithValues("serving", req.NamespacedName)

	var s openfunction.Serving

	if err := r.Get(ctx, req.NamespacedName, &s); err != nil {
		log.V(1).Info("Serving deleted", "error", err)
		return ctrl.Result{}, util.IgnoreNotFound(err)
	}

	if _, err := r.createOrUpdateServing(&s); err != nil {
		log.Error(err, "Failed to create serving", "Serving", req.NamespacedName.String())
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ServingReconciler) mutateKsvc(ksvc *kservingv1.Service, s *openfunction.Serving) controllerutil.MutateFn {
	return func() error {

		template := s.Spec.Template
		if template == nil {
			template = &corev1.PodSpec{}
		}

		var container *corev1.Container
		for index := range template.Containers {
			if template.Containers[index].Name == FunctionContainer {
				container = &template.Containers[index]
			}
		}

		appended := false
		if container == nil {
			container = &corev1.Container{
				Name:            FunctionContainer,
				ImagePullPolicy: corev1.PullIfNotPresent,
			}
			appended = true
		}

		container.Image = s.Spec.Image

		port := corev1.ContainerPort{}
		if s.Spec.Port != nil {
			port.ContainerPort = *s.Spec.Port
			container.Ports = append(container.Ports, port)
		}

		if s.Spec.Params != nil {
			for k, v := range s.Spec.Params {
				container.Env = append(container.Env, corev1.EnvVar{
					Name:  k,
					Value: v,
				})
			}
		}

		if appended {
			template.Containers = append(template.Containers, *container)
		}

		objectMeta := metav1.ObjectMeta{
			Namespace: s.Namespace,
		}
		if s.Spec.Version != nil {
			objectMeta.Name = fmt.Sprintf("%s-%s", ksvc.Name, strings.ReplaceAll(*s.Spec.Version, ".", ""))
		}

		expected := kservingv1.Service{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "serving.knative.dev/v1",
				Kind:       "Service",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      s.Name,
				Namespace: s.Namespace,
			},
			Spec: kservingv1.ServiceSpec{
				ConfigurationSpec: kservingv1.ConfigurationSpec{
					Template: kservingv1.RevisionTemplateSpec{
						ObjectMeta: objectMeta,
						Spec: kservingv1.RevisionSpec{
							PodSpec: *template,
						},
					},
				},
			},
		}

		expected.Spec.DeepCopyInto(&ksvc.Spec)
		ksvc.SetOwnerReferences(nil)
		return ctrl.SetControllerReference(s, ksvc, r.Scheme)
	}
}

func (r *ServingReconciler) createOrUpdateServing(s *openfunction.Serving) (ctrl.Result, error) {
	if s.Status.Phase == openfunction.ServingPhase && s.Status.State == openfunction.Launched {
		return ctrl.Result{}, nil
	}

	log := r.Log.WithName("createOrUpdateServing")

	status := openfunction.ServingStatus{Phase: openfunction.ServingPhase, State: openfunction.Launching}
	if err := r.updateStatus(s, &status); err != nil {
		log.Error(err, "Failed to update serving Launching status", "name", s.Name, "namespace", s.Namespace)
		return ctrl.Result{}, err
	}

	switch *s.Spec.Runtime {
	case openfunction.Knative:
		if err := r.createOrUpdateKnativeService(s); err != nil {
			log.Error(err, "Failed to CreateOrUpdate knative service", "error", err.Error())
			return ctrl.Result{}, err
		}
		break
	case openfunction.OpenFuncAsync:
		if err := r.createOrUpdateOpenFuncAsyncService(s); err != nil {
			log.Error(err, "Failed to CreateOrUpdate dapr service", "error", err.Error())
			return ctrl.Result{}, err
		}
		break
	default:
		err := fmt.Errorf("unknow runtime %s", *s.Spec.Runtime)
		log.Error(err, "unknown runtime", "runtime", *s.Spec.Runtime)
		return ctrl.Result{}, err
	}

	status = openfunction.ServingStatus{Phase: openfunction.ServingPhase, State: openfunction.Launched}
	if err := r.updateStatus(s, &status); err != nil {
		log.Error(err, "Failed to update serving Launched status", "name", s.Name, "namespace", s.Namespace)
		return ctrl.Result{}, err
	}

	log.V(1).Info("Create serving", "name", s.Name, "namespace", s.Namespace)

	return ctrl.Result{}, nil
}

func (r *ServingReconciler) createOrUpdateKnativeService(s *openfunction.Serving) error {

	log := r.Log.WithName("createOrUpdateKnativeService")
	ksvc := kservingv1.Service{}
	ksvc.Name = fmt.Sprintf("%s-%s", s.Name, "ksvc")
	ksvc.Namespace = s.Namespace

	if err := r.Delete(r.ctx, &ksvc); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to delete old knative service", "name", ksvc.Name, "namespace", ksvc.Namespace)
		return err
	}

	if err := r.mutateKsvc(&ksvc, s)(); err != nil {
		log.Error(err, "Failed to mutate knative service", "name", s.Name, "namespace", s.Namespace)
		return err
	}

	if err := r.Create(r.ctx, &ksvc); err != nil {
		log.Error(err, "Failed to Create knative service", "name", s.Name, "namespace", s.Namespace)
		return err
	}

	return nil
}

func (r *ServingReconciler) updateStatus(s *openfunction.Serving, status *openfunction.ServingStatus) error {

	status.DeepCopyInto(&s.Status)
	if err := r.Status().Update(r.ctx, s); err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openfunction.Serving{}).
		Complete(r)
}
