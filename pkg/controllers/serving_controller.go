/*


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

package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	openfunction "github.com/openfunction/pkg/apis/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kservingv1 "knative.dev/serving/pkg/apis/serving/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ServingReconciler reconciles a Serving object
type ServingReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	ctx    context.Context
}

// +kubebuilder:rbac:groups=core.openfunction.io,resources=servings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.openfunction.io,resources=servings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=serving.knative.dev,resources=services,verbs=get;list;watch;create;update;patch;delete

func (r *ServingReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	r.ctx = ctx
	log := r.Log.WithValues("serving", req.NamespacedName)

	var s openfunction.Serving

	if err := r.Get(ctx, req.NamespacedName, &s); err != nil {
		log.V(10).Info("Serving deleted", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if _, err := r.createOrUpdateServing(&s); err != nil {
		log.Error(err, "Failed to create serving", "Serving", req.NamespacedName.String())
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ServingReconciler) mutateKsvc(ksvc *kservingv1.Service, s *openfunction.Serving) controllerutil.MutateFn {
	return func() error {
		container := corev1.Container{
			Image: s.Spec.Image,
		}
		port := corev1.ContainerPort{}
		if s.Spec.Port != nil {
			port.ContainerPort = *s.Spec.Port
			container.Ports = append(container.Ports, port)
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
						Spec: kservingv1.RevisionSpec{
							PodSpec: corev1.PodSpec{
								Containers: []corev1.Container{
									container,
								},
							},
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
	if s.Status.Phase != "" && s.Status.State != "" {
		return ctrl.Result{}, nil
	}

	log := r.Log.WithName("createOrUpdateServing")

	status := openfunction.ServingStatus{Phase: openfunction.ServingPhase, State: openfunction.Launching}
	if err := r.updateStatus(s, &status); err != nil {
		return ctrl.Result{}, err
	}

	ksvc := kservingv1.Service{}
	ksvc.Name = fmt.Sprintf("%s-%s", s.Name, "ksvc")
	ksvc.Namespace = s.Namespace

	if result, err := controllerutil.CreateOrUpdate(r.ctx, r.Client, &ksvc, r.mutateKsvc(&ksvc, s)); err != nil {
		log.Error(err, "Failed to CreateOrUpdate knative service", "result", result)
		return ctrl.Result{}, err
	}

	status = openfunction.ServingStatus{Phase: openfunction.ServingPhase, State: openfunction.Launched}
	if err := r.updateStatus(s, &status); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ServingReconciler) updateStatus(s *openfunction.Serving, status *openfunction.ServingStatus) error {
	status.DeepCopyInto(&s.Status)
	if err := r.Status().Update(r.ctx, s); err != nil {
		return err
	}
	return nil
}

func (r *ServingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openfunction.Serving{}).
		Complete(r)
}
