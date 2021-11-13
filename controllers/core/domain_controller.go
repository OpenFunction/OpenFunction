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
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	openfunction "github.com/openfunction/apis/core/v1alpha2"
	"github.com/openfunction/pkg/util"
)

const (
	reloadTimestamp = "reloadtimestamp"
)

// DomainReconciler reconciles a Domain object
type DomainReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	ctx    context.Context
}

func NewDomainReconciler(mgr manager.Manager) *DomainReconciler {

	r := &DomainReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("Domain"),
	}

	return r
}

//+kubebuilder:rbac:groups=core.openfunction.io,resources=domains,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.openfunction.io,resources=domains/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Serving object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *DomainReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.ctx = ctx
	log := r.Log.WithValues("Domain", req.NamespacedName)

	var d openfunction.Domain

	if err := r.Get(ctx, req.NamespacedName, &d); err != nil {
		if util.IsNotFound(err) {
			log.V(1).Info("Domain deleted")
		}
		return ctrl.Result{}, util.IgnoreNotFound(err)
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.Name,
			Namespace: d.Namespace,
		},
	}

	if err := r.Get(r.ctx, client.ObjectKeyFromObject(svc), svc); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get service")
		return ctrl.Result{}, err
	}

	op, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, svc, r.mutateService(&d, svc))
	if err != nil {
		log.Error(err, "Failed to CreateOrUpdate domain")
		return ctrl.Result{}, err
	}

	if err := r.updateFunction(); err != nil {
		log.Error(err, "Failed to update function")
		return ctrl.Result{}, err
	}

	log.V(1).Info(fmt.Sprintf("Domain %s", op))
	return ctrl.Result{}, nil
}

func (r *DomainReconciler) mutateService(d *openfunction.Domain, svc *corev1.Service) controllerutil.MutateFn {

	return func() error {

		svc.Spec.ExternalName =
			fmt.Sprintf("%s.%s.svc.cluster.local", d.Spec.Ingress.Service.Name, d.Spec.Ingress.Service.Namespace)
		svc.Spec.Type = corev1.ServiceTypeExternalName

		port := d.Spec.Ingress.Service.Port
		if port == 0 {
			port = 80
		}

		svc.Spec.Ports = []corev1.ServicePort{
			{
				Name:     "http",
				Port:     port,
				Protocol: corev1.ProtocolTCP,
				TargetPort: intstr.IntOrString{
					IntVal: port,
				},
			},
		}

		return controllerutil.SetControllerReference(d, svc, r.Scheme)
	}
}

func (r *DomainReconciler) updateFunction() error {
	fnList := &openfunction.FunctionList{}
	if err := r.List(r.ctx, fnList); err != nil {
		return err
	}

	for _, fn := range fnList.Items {
		fn.Annotations[reloadTimestamp] = time.Now().String()
		if err := r.Update(r.ctx, &fn); err != nil {
			return err
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DomainReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&openfunction.Domain{}).Complete(r)
}
