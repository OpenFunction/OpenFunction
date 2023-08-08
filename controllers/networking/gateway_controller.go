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

package networking

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sgatewayapiv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	networkingv1alpha1 "github.com/openfunction/apis/networking/v1alpha1"
	ofngateway "github.com/openfunction/pkg/networking/gateway"
	"github.com/openfunction/pkg/util"
)

const (
	GatewayField         = ".spec.gatewayRef"
	GatewayFinalizerName = "networking.openfunction.io/finalizer"
)

// GatewayReconciler reconciles a Gateway object
type GatewayReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	ctx        context.Context
	k8sGateway *k8sgatewayapiv1alpha2.Gateway
}

//+kubebuilder:rbac:groups=networking.openfunction.io,resources=gateways,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.openfunction.io,resources=gateways/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=networking.openfunction.io,resources=gateways/finalizers,verbs=update
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Gateway object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//_ = log.FromContext(ctx)
	log := r.Log.WithValues("Gateway", req.NamespacedName)
	r.ctx = ctx

	gateway := &networkingv1alpha1.Gateway{}

	if err := r.Get(ctx, req.NamespacedName, gateway); err != nil {
		if util.IsNotFound(err) {
			log.V(1).Info("Gateway deleted", "error", err)
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if gateway.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(gateway, GatewayFinalizerName) {
			controllerutil.AddFinalizer(gateway, GatewayFinalizerName)
			if err := r.Update(ctx, gateway); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(gateway, GatewayFinalizerName) {
			if err := r.cleanK8sGatewayResources(gateway); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(gateway, GatewayFinalizerName)
			if err := r.Update(ctx, gateway); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if err := r.cleanExternalResources(gateway); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.createOrUpdateGateway(gateway); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.updateGatewayAnnotations(gateway); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.createOrUpdateService(gateway); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *GatewayReconciler) createOrUpdateGateway(gateway *networkingv1alpha1.Gateway) error {
	defer r.updateGatewayStatus(gateway.Status.DeepCopy(), gateway)
	log := r.Log.WithName("createOrUpdateGateway")
	k8sGateway := &k8sgatewayapiv1alpha2.Gateway{}

	if gateway.Spec.GatewayRef != nil {
		key := client.ObjectKey{Namespace: gateway.Spec.GatewayRef.Namespace, Name: gateway.Spec.GatewayRef.Name}
		if err := r.Get(r.ctx, key, k8sGateway); err != nil {
			log.Error(err, "Failed to get k8s Gateway",
				"namespace", gateway.Spec.GatewayRef.Namespace, "name", gateway.Spec.GatewayRef.Name)
			reason := k8sgatewayapiv1alpha2.GatewayReasonNotReconciled
			if util.IsNotFound(err) {
				reason = networkingv1alpha1.GatewayReasonNotFound
			}
			gateway.Status.Conditions = []networkingv1alpha1.Condition{
				{
					Type:    string(k8sgatewayapiv1alpha2.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(reason),
					Message: err.Error(),
				},
			}
			return err
		} else {
			r.k8sGateway = k8sGateway
			if r.needReconcileK8sGateway(gateway) {
				if err := r.reconcileK8sGateway(gateway); err != nil {
					return err
				}
			}
		}
	}

	if gateway.Spec.GatewayDef != nil {
		key := client.ObjectKey{Namespace: gateway.Spec.GatewayDef.Namespace, Name: gateway.Spec.GatewayDef.Name}
		if err := r.Get(r.ctx, key, k8sGateway); err == nil {
			r.k8sGateway = k8sGateway
			if r.needReconcileK8sGateway(gateway) {
				if err := r.reconcileK8sGateway(gateway); err != nil {
					return err
				}
			}
		} else if util.IsNotFound(err) {
			if err := r.createK8sGateway(gateway); err != nil {
				return err
			}
		} else {
			log.Error(err, "Failed to reconcile k8s Gateway",
				"namespace", gateway.Spec.GatewayDef.Namespace, "name", gateway.Spec.GatewayDef.Name)
			gateway.Status.Conditions = []networkingv1alpha1.Condition{
				{
					Type:    string(k8sgatewayapiv1alpha2.GatewayConditionReady),
					Status:  metav1.ConditionFalse,
					Reason:  string(k8sgatewayapiv1alpha2.GatewayReasonNotReconciled),
					Message: err.Error(),
				},
			}
			return err
		}
	}

	if r.k8sGateway != nil {
		r.syncStatusFromK8sGateway(gateway)
	}

	return nil
}

func (r *GatewayReconciler) createK8sGateway(gateway *networkingv1alpha1.Gateway) error {
	log := r.Log.WithName("createK8sGateway")
	listenersAnnotation, _ := json.Marshal(gateway.Spec.GatewaySpec.Listeners)
	k8sGateway := &k8sgatewayapiv1alpha2.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:        gateway.Spec.GatewayDef.Name,
			Namespace:   gateway.Spec.GatewayDef.Namespace,
			Annotations: map[string]string{networkingv1alpha1.GatewayListenersAnnotation: string(listenersAnnotation)},
		},
		Spec: k8sgatewayapiv1alpha2.GatewaySpec{
			GatewayClassName: gateway.Spec.GatewayDef.GatewayClassName,
			Listeners:        gateway.Spec.GatewaySpec.Listeners,
		},
	}

	if err := r.Create(r.ctx, k8sGateway); err != nil {
		log.Error(err, "Failed to create k8s Gateway",
			"namespace", gateway.Spec.GatewayDef.Namespace, "name", gateway.Spec.GatewayDef.Name)
		gateway.Status.Conditions = []networkingv1alpha1.Condition{
			{
				Type:    string(k8sgatewayapiv1alpha2.GatewayConditionReady),
				Status:  metav1.ConditionFalse,
				Reason:  string(networkingv1alpha1.GatewayReasonCreationFailure),
				Message: err.Error(),
			},
		}
		return err
	}

	gateway.Status.Conditions = append(gateway.Status.Conditions, networkingv1alpha1.Condition{
		Type:    string(k8sgatewayapiv1alpha2.GatewayConditionScheduled),
		Status:  metav1.ConditionTrue,
		Reason:  string(networkingv1alpha1.GatewayReasonResourcesAvailable),
		Message: "Deployed k8s gateway to the cluster",
	})
	r.k8sGateway = k8sGateway
	log.Info("K8s Gateway Deployed", "namespace", k8sGateway.Namespace, "name", k8sGateway.Name)
	return nil
}

func (r *GatewayReconciler) reconcileK8sGateway(gateway *networkingv1alpha1.Gateway) error {
	log := r.Log.WithName("reconcileK8sGateway")
	var oldGateway networkingv1alpha1.Gateway
	oldGatewayListenersMapping := make(map[k8sgatewayapiv1alpha2.SectionName]k8sgatewayapiv1alpha2.Listener)

	gatewayConfigAnnotation := []byte(gateway.Annotations[networkingv1alpha1.GatewayConfigAnnotation])
	if err := json.Unmarshal(gatewayConfigAnnotation, &oldGateway); err != nil {
		log.Error(err, "Failed to Unmarshal GatewayConfigAnnotation")
	} else {
		oldGatewayListenersMapping = ofngateway.ConvertListenersListToMapping(oldGateway.Spec.GatewaySpec.Listeners)
	}

	newGatewayListenersMapping := ofngateway.ConvertListenersListToMapping(gateway.Spec.GatewaySpec.Listeners)
	k8sGatewayListenersMapping := ofngateway.ConvertListenersListToMapping(r.k8sGateway.Spec.Listeners)
	for name := range oldGatewayListenersMapping {
		if _, ok := newGatewayListenersMapping[name]; !ok {
			delete(k8sGatewayListenersMapping, name)
		}
	}
	for name, listener := range newGatewayListenersMapping {
		k8sGatewayListenersMapping[name] = listener
	}
	r.k8sGateway.Spec.Listeners = ofngateway.ConvertListenersMappingToList(k8sGatewayListenersMapping)
	listenersAnnotation, _ := json.Marshal(gateway.Spec.GatewaySpec.Listeners)
	if r.k8sGateway.Annotations == nil {
		r.k8sGateway.Annotations = make(map[string]string)
	}
	r.k8sGateway.Annotations[networkingv1alpha1.GatewayListenersAnnotation] = string(listenersAnnotation)

	if err := r.Update(r.ctx, r.k8sGateway); err != nil {
		log.Error(err, "Failed to reconcile k8s Gateway",
			"namespace", r.k8sGateway.Namespace, "name", r.k8sGateway.Name)
		gateway.Status.Conditions = []networkingv1alpha1.Condition{
			{
				Type:    string(k8sgatewayapiv1alpha2.GatewayConditionReady),
				Status:  metav1.ConditionFalse,
				Reason:  string(k8sgatewayapiv1alpha2.GatewayReasonNotReconciled),
				Message: err.Error(),
			},
		}
		return err
	}
	log.Info("K8s Gateway Reconciled", "namespace", r.k8sGateway.Namespace, "name", r.k8sGateway.Name)
	return nil
}

func (r *GatewayReconciler) cleanExternalResources(gateway *networkingv1alpha1.Gateway) error {
	log := r.Log.WithName("cleanK8sGatewayResources")
	var oldGateway networkingv1alpha1.Gateway

	gatewayConfigAnnotation := []byte(gateway.Annotations[networkingv1alpha1.GatewayConfigAnnotation])
	if err := json.Unmarshal(gatewayConfigAnnotation, &oldGateway); err != nil {
		log.Error(err, "Failed to Unmarshal GatewayConfigAnnotation")
		return nil
	}

	if !equality.Semantic.DeepEqual(oldGateway.Spec.GatewayRef, gateway.Spec.GatewayRef) ||
		!equality.Semantic.DeepEqual(oldGateway.Spec.GatewayDef, gateway.Spec.GatewayDef) {
		if err := r.cleanK8sGatewayResources(&oldGateway); err != nil {
			return err
		}
	}
	return nil
}

func (r *GatewayReconciler) cleanK8sGatewayResources(gateway *networkingv1alpha1.Gateway) error {
	log := r.Log.WithName("cleanK8sGatewayResources")
	if gateway.Spec.GatewayRef != nil {
		k8sGateway := &k8sgatewayapiv1alpha2.Gateway{}
		key := client.ObjectKey{Namespace: gateway.Spec.GatewayRef.Namespace, Name: gateway.Spec.GatewayRef.Name}
		if err := r.Get(r.ctx, key, k8sGateway); err != nil {
			if !util.IsNotFound(err) {
				log.Error(err, "Failed to get k8s gateway",
					"namespace", gateway.Spec.GatewayRef.Namespace, "name", gateway.Spec.GatewayRef.Name)
			}
			return util.IgnoreNotFound(err)
		}
		needRemoveListenersMapping := ofngateway.ConvertListenersListToMapping(gateway.Spec.GatewaySpec.Listeners)
		k8sGatewayListenersMapping := ofngateway.ConvertListenersListToMapping(k8sGateway.Spec.Listeners)
		for name := range needRemoveListenersMapping {
			delete(k8sGatewayListenersMapping, name)
		}
		k8sGateway.Spec.Listeners = ofngateway.ConvertListenersMappingToList(k8sGatewayListenersMapping)
		if k8sGateway.Annotations != nil {
			delete(k8sGateway.Annotations, networkingv1alpha1.GatewayListenersAnnotation)
		}
		if err := r.Update(r.ctx, k8sGateway); err != nil {
			log.Error(err, "Failed to clean k8s Gateway",
				"namespace", gateway.Spec.GatewayRef.Namespace, "name", gateway.Spec.GatewayRef.Name)
			return err
		}
	}

	if gateway.Spec.GatewayDef != nil {
		k8sGateway := &k8sgatewayapiv1alpha2.Gateway{
			ObjectMeta: metav1.ObjectMeta{Namespace: gateway.Spec.GatewayDef.Namespace, Name: gateway.Spec.GatewayDef.Name},
		}
		if err := r.Delete(r.ctx, k8sGateway); err != nil {
			if !util.IsNotFound(err) {
				log.Error(err, "Failed to clean k8s Gateway",
					"namespace", gateway.Spec.GatewayDef.Namespace, "name", gateway.Spec.GatewayDef.Name)
			}
			return util.IgnoreNotFound(err)
		}
	}
	return nil
}

func (r *GatewayReconciler) needReconcileK8sGateway(gateway *networkingv1alpha1.Gateway) bool {
	gatewayListeners := ofngateway.ConvertListenersListToMapping(gateway.Spec.GatewaySpec.Listeners)
	k8sGatewayListeners := ofngateway.ConvertListenersListToMapping(r.k8sGateway.Spec.Listeners)
	for name, gatewayListener := range gatewayListeners {
		if k8sGatewayListener, ok := k8sGatewayListeners[name]; !ok || !equality.Semantic.DeepEqual(gatewayListener, k8sGatewayListener) {
			return true
		}
	}
	var oldGateway networkingv1alpha1.Gateway
	gatewayConfigAnnotation := []byte(gateway.Annotations[networkingv1alpha1.GatewayConfigAnnotation])
	if err := json.Unmarshal(gatewayConfigAnnotation, &oldGateway); err != nil {
		return true
	}

	if !equality.Semantic.DeepEqual(oldGateway.Spec.GatewaySpec.Listeners, gateway.Spec.GatewaySpec.Listeners) {
		return true
	}
	return false
}

func (r *GatewayReconciler) updateGatewayAnnotations(gateway *networkingv1alpha1.Gateway) error {
	log := r.Log.WithName("updateGatewayAnnotations")
	var oldGateway networkingv1alpha1.Gateway

	oldGatewayConfigAnnotation := []byte(gateway.Annotations[networkingv1alpha1.GatewayConfigAnnotation])
	if err := json.Unmarshal(oldGatewayConfigAnnotation, &oldGateway); err != nil {
		log.Error(err, "Failed to Unmarshal GatewayConfigAnnotation")
	}
	if equality.Semantic.DeepEqual(oldGateway.Spec, gateway.Spec) {
		return nil
	}
	gatewayConfigAnnotation, _ := json.Marshal(networkingv1alpha1.Gateway{Spec: gateway.Spec})
	gateway.Annotations[networkingv1alpha1.GatewayConfigAnnotation] = string(gatewayConfigAnnotation)
	if err := r.Update(r.ctx, gateway); err != nil {
		log.Error(err, "Failed to update annotations on Gateway", "namespace", gateway.Namespace, "name", gateway.Name)
		return err
	}
	log.Info("Updated annotations on Gateway", "namespace", gateway.Namespace, "name", gateway.Name)
	return nil
}

func convertConditions(conditions []metav1.Condition) []networkingv1alpha1.Condition {
	var dest []networkingv1alpha1.Condition
	for _, condition := range conditions {
		dest = append(dest, networkingv1alpha1.Condition{
			Type:    condition.Type,
			Status:  condition.Status,
			Reason:  condition.Reason,
			Message: condition.Message,
		})
	}

	return dest
}

func (r *GatewayReconciler) syncStatusFromK8sGateway(gateway *networkingv1alpha1.Gateway) {
	gateway.Status.Conditions = convertConditions(r.k8sGateway.Status.Conditions)
	gatewayListeners := ofngateway.ConvertListenersListToMapping(gateway.Spec.GatewaySpec.Listeners)
	var refreshedGatewayListeners []networkingv1alpha1.ListenerStatus
	for _, gatewayListener := range r.k8sGateway.Status.Listeners {
		if _, ok := gatewayListeners[gatewayListener.Name]; ok {
			refreshedGatewayListeners = append(refreshedGatewayListeners, networkingv1alpha1.ListenerStatus{
				Name:           gatewayListener.Name,
				SupportedKinds: gatewayListener.SupportedKinds,
				AttachedRoutes: gatewayListener.AttachedRoutes,
				Conditions:     convertConditions(gatewayListener.Conditions),
			})
		}
	}
	gateway.Status.Listeners = refreshedGatewayListeners
	gateway.Status.Addresses = r.k8sGateway.Status.Addresses
}

func (r *GatewayReconciler) createOrUpdateService(gateway *networkingv1alpha1.Gateway) error {
	log := r.Log.WithName("createOrUpdateService")
	var externalName string

	// For the k8s gateway controller implements the addresses field, such as istio.
	if len(r.k8sGateway.Status.Addresses) > 0 {
		address := r.k8sGateway.Status.Addresses[0]
		if *address.Type == k8sgatewayapiv1alpha2.HostnameAddressType {
			externalName = strings.Split(address.Value, ":")[0]
		}
	}

	// For the gateway controller does not implement the addresses field, such as contour.
	if externalName == "" {
		targetServices := []string{
			fmt.Sprintf("%s-%s", r.k8sGateway.Name, networkingv1alpha1.DefaultK8sGatewayServiceName),
			fmt.Sprintf("%s-%s", networkingv1alpha1.DefaultK8sGatewayServiceName, r.k8sGateway.Name),
			networkingv1alpha1.DefaultK8sGatewayServiceName,
		}
		for _, serviceName := range targetServices {
			gatewayService := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Namespace: r.k8sGateway.Namespace, Name: serviceName},
			}
			if err := r.Get(r.ctx, client.ObjectKeyFromObject(gatewayService), gatewayService); err == nil {
				externalName = fmt.Sprintf("%s.%s.svc.%s",
					serviceName, r.k8sGateway.Namespace, gateway.Spec.ClusterDomain)
				break
			} else if !util.IsNotFound(err) {
				log.Error(err, "Failed to CreateOrUpdate service")
				return err
			}
		}
		if externalName == "" {
			err := errors.New(string(metav1.StatusReasonNotFound))
			log.Error(err, "Failed to CreateOrUpdate service")
			return err
		}
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Namespace: gateway.Namespace, Name: networkingv1alpha1.DefaultGatewayServiceName},
	}
	op, err := controllerutil.CreateOrUpdate(r.ctx, r.Client, service, r.mutateService(gateway, externalName, service))
	if err != nil {
		log.Error(err, "Failed to CreateOrUpdate service")
		return err
	}
	log.V(1).Info(fmt.Sprintf("Service %s", op))
	return nil
}

func (r *GatewayReconciler) mutateService(
	gateway *networkingv1alpha1.Gateway,
	externalName string,
	service *corev1.Service) controllerutil.MutateFn {
	return func() error {
		if r.k8sGateway != nil {
			var servicePorts []corev1.ServicePort
			for _, listener := range gateway.Spec.GatewaySpec.Listeners {
				if !strings.HasSuffix(string(*listener.Hostname), gateway.Spec.ClusterDomain) {
					servicePort := corev1.ServicePort{
						Name:       string(listener.Name),
						Protocol:   corev1.ProtocolTCP,
						Port:       int32(listener.Port),
						TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(listener.Port)},
					}
					servicePorts = append(servicePorts, servicePort)
				}
			}
			service.Spec.Type = corev1.ServiceTypeExternalName
			service.Spec.Ports = servicePorts
			service.Spec.ExternalName = externalName
			return ctrl.SetControllerReference(gateway, service, r.Scheme)
		}
		return nil
	}
}

func (r *GatewayReconciler) updateGatewayStatus(oldStatus *networkingv1alpha1.GatewayStatus, gateway *networkingv1alpha1.Gateway) {
	log := r.Log.WithName("updateGatewayStatus")
	if !equality.Semantic.DeepEqual(oldStatus, gateway.Status.DeepCopy()) {
		if err := r.Status().Update(r.ctx, gateway); err != nil {
			log.Error(err, "Failed to update status on Gateway", "namespace", gateway.Namespace, "name", gateway.Name)
		} else {
			log.Info("Updated status on Gateway", "namespace", gateway.Namespace, "name", gateway.Name)
		}
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &networkingv1alpha1.Gateway{}, GatewayField, func(rawObj client.Object) []string {
		gateway := rawObj.(*networkingv1alpha1.Gateway)
		if gateway.Spec.GatewayRef != nil {
			return []string{fmt.Sprintf("%s,%s", gateway.Spec.GatewayRef.Namespace, gateway.Spec.GatewayRef.Name)}
		}
		if gateway.Spec.GatewayDef != nil {
			return []string{fmt.Sprintf("%s,%s", gateway.Spec.GatewayDef.Namespace, gateway.Spec.GatewayDef.Name)}
		}
		return nil
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1alpha1.Gateway{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&corev1.Service{}).
		Watches(
			&source.Kind{Type: &k8sgatewayapiv1alpha2.Gateway{}},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForK8sGateway),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}, predicate.Funcs{UpdateFunc: r.filterK8sGatewayUpdateEvent}),
		).
		Complete(r)
}

func (r *GatewayReconciler) filterK8sGatewayUpdateEvent(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	oldGateway := e.ObjectOld.(*k8sgatewayapiv1alpha2.Gateway).DeepCopy()
	newGateway := e.ObjectNew.(*k8sgatewayapiv1alpha2.Gateway).DeepCopy()

	if !reflect.DeepEqual(oldGateway.Spec, newGateway.Spec) {
		return true
	}

	oldGateway.ManagedFields = make([]metav1.ManagedFieldsEntry, 0)
	newGateway.ManagedFields = make([]metav1.ManagedFieldsEntry, 0)
	newGateway.ResourceVersion = ""
	oldGateway.ResourceVersion = ""
	if !reflect.DeepEqual(oldGateway.ObjectMeta, newGateway.ObjectMeta) {
		return true
	}

	return false
}

func (r *GatewayReconciler) findObjectsForK8sGateway(k8sGateway client.Object) []reconcile.Request {
	attachedGateways := &networkingv1alpha1.GatewayList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(GatewayField, fmt.Sprintf("%s,%s", k8sGateway.GetNamespace(), k8sGateway.GetName())),
	}
	err := r.List(context.TODO(), attachedGateways, listOps)
	if err != nil {
		return []reconcile.Request{}
	}
	requests := make([]reconcile.Request, len(attachedGateways.Items))
	for i, item := range attachedGateways.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}
