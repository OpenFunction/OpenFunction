package controllers

import (
	"fmt"

	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	subscriptionsv1alpha1 "github.com/dapr/dapr/pkg/apis/subscriptions/v1alpha1"
	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	openfunction "github.com/openfunction/pkg/apis/v1alpha1"
	"github.com/openfunction/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	UserContainer = "user-container"
	UserPort      = "user-port"
)

func (r *ServingReconciler) createOrUpdateDaprService(s *openfunction.Serving) error {
	log := r.Log.WithName("createOrUpdateDaprService")

	if s.Spec.Dapr == nil {
		return fmt.Errorf("dapr config must not be nil when using dapr serving")
	}

	workload, err := r.createOrUpdateWorkload(s)
	if err != nil {
		log.Error(err, "Failed to CreateOrUpdate dapr workload", "err", err.Error(), "name", s.Name, "namespace", s.Namespace)
		return err
	}

	if err := r.createOrUpdateScaler(s, workload); err != nil {
		log.Error(err, "Failed to CreateOrUpdate keda triggers", "error", err.Error(), "name", s.Name, "namespace", s.Namespace)
		return err
	}

	if err := r.createOrUpdateSvc(s, workload); err != nil {
		log.Error(err, "Failed to CreateOrUpdate dapr service", "name", s.Name, "namespace", s.Namespace)
		return err
	}

	if err := r.createOrUpdateComponents(s); err != nil {
		log.Error(err, "Failed to CreateOrUpdate dapr components", "error", err.Error(), "name", s.Name, "namespace", s.Namespace)
		return err
	}

	if err := r.createOrUpdateSubscriptions(s); err != nil {
		log.Error(err, "Failed to CreateOrUpdate dapr subscriptions", "error", err.Error(), "name", s.Name, "namespace", s.Namespace)
		return err
	}

	return nil
}

func (r *ServingReconciler) createOrUpdateWorkload(s *openfunction.Serving) (runtime.Object, error) {
	log := r.Log.WithName("createOrUpdateWorkload")
	dapr := s.Spec.Dapr

	var obj runtime.Object
	if dapr.ScaledJob != nil {
		obj = &batchv1.Job{}
	} else if dapr.ScaledObject != nil {
		if dapr.ScaledObject.WorkloadType == "StatefulSet" {
			obj = &appsv1.StatefulSet{}
		} else {
			obj = &appsv1.Deployment{}
		}
	} else {
		obj = &appsv1.Deployment{}
	}

	accessor, _ := meta.Accessor(obj)
	accessor.SetName(s.Name)
	accessor.SetNamespace(s.Namespace)

	if err := r.Delete(r.ctx, obj); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to delete dapr workload", "name", s.Name, "namespace", s.Namespace)
		return nil, err
	}

	if err := r.mutateWorkload(obj, s)(); err != nil {
		log.Error(err, "Failed to mutate dapr workload", "name", s.Name, "namespace", s.Namespace)
		return nil, err
	}

	if err := r.Create(r.ctx, obj); err != nil {
		log.Error(err, "Failed to create dapr workload", "name", s.Name, "namespace", s.Namespace)
		return nil, err
	}

	log.V(1).Info("Create workload", "name", s.Name, "namespace", s.Namespace)
	return obj, nil
}

func (r *ServingReconciler) mutateWorkload(obj runtime.Object, s *openfunction.Serving) controllerutil.MutateFn {

	return func() error {

		dapr := s.Spec.Dapr

		accessor, _ := meta.Accessor(obj)
		labels := map[string]string{
			"openfunction.io/managed": "true",
			"serving":                 s.Name,
			"runtime":                 string(openfunction.DAPR),
		}
		accessor.SetLabels(labels)

		selector := &metav1.LabelSelector{
			MatchLabels: labels,
		}

		var replicas int32 = 1
		if dapr.ScaledObject != nil && dapr.ScaledObject.MinReplicaCount != nil {
			replicas = *dapr.ScaledObject.MinReplicaCount
		}

		var port int32 = 8080
		if s.Spec.Port != nil {
			port = *s.Spec.Port
		}

		var env []corev1.EnvVar
		if s.Spec.Params != nil {
			for k, v := range s.Spec.Params {
				env = append(env, corev1.EnvVar{
					Name:  k,
					Value: v,
				})
			}
		}

		template := corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: dapr.Annotations,
				Labels:      labels,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  UserContainer,
						Image: s.Spec.Image,
						Ports: []corev1.ContainerPort{
							{
								Name:          UserPort,
								ContainerPort: port,
								Protocol:      corev1.ProtocolTCP,
							},
						},
						ImagePullPolicy: corev1.PullAlways,
						Env:             env,
					},
				},
			},
		}

		switch obj.(type) {
		case *batchv1.Job:
			job := obj.(*batchv1.Job)
			job.Spec.Selector = selector
			job.Spec.Template = template
		case *appsv1.Deployment:
			deploy := obj.(*appsv1.Deployment)
			deploy.Spec.Selector = selector
			deploy.Spec.Replicas = &replicas
			deploy.Spec.Template = template
		case *appsv1.StatefulSet:
			statefulSet := obj.(*appsv1.StatefulSet)
			statefulSet.Spec.Selector = selector
			statefulSet.Spec.Replicas = &replicas
			statefulSet.Spec.Template = template
		}

		return controllerutil.SetControllerReference(s, accessor, r.Scheme)
	}
}

func (r *ServingReconciler) createOrUpdateScaler(s *openfunction.Serving, workload runtime.Object) error {
	log := r.Log.WithName("createOrUpdateScaler")

	dapr := s.Spec.Dapr
	if dapr.ScaledJob == nil && dapr.ScaledObject == nil {
		return nil
	}

	var obj runtime.Object
	if dapr.ScaledJob != nil {
		obj = &kedav1alpha1.ScaledJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", s.Name, "scaler"),
				Namespace: s.Namespace,
			},
		}
	} else {
		obj = &kedav1alpha1.ScaledObject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", s.Name, "scaler"),
				Namespace: s.Namespace,
			},
		}
	}

	if err := r.Delete(r.ctx, obj); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to delete keda scaler", "name", s.Name, "namespace", s.Namespace)
		return err
	}

	if err := r.mutateScaler(obj, workload, s)(); err != nil {
		log.Error(err, "Failed to mutate keda scaler", "name", s.Name, "namespace", s.Namespace)
		return err
	}

	if err := r.Create(r.ctx, obj); err != nil {
		log.Error(err, "Failed to create keda scaler", "name", s.Name, "namespace", s.Namespace)
		return err
	}

	log.V(1).Info("Create scaler", "serving", s.Name, "namespace", s.Namespace)
	return nil
}

func (r *ServingReconciler) mutateScaler(obj runtime.Object, workload runtime.Object, s *openfunction.Serving) controllerutil.MutateFn {
	return func() error {

		dapr := s.Spec.Dapr
		switch obj.(type) {
		case *kedav1alpha1.ScaledJob:
			if dapr.ScaledJob == nil {
				return fmt.Errorf("ScaledJob is nil")
			}

			ref, err := r.getJobTargetRef(workload)
			if err != nil {
				return err
			}

			scaler := obj.(*kedav1alpha1.ScaledJob)
			scaledJob := dapr.ScaledJob
			scaler.Spec = kedav1alpha1.ScaledJobSpec{
				JobTargetRef:               ref,
				PollingInterval:            scaledJob.PollingInterval,
				SuccessfulJobsHistoryLimit: scaledJob.SuccessfulJobsHistoryLimit,
				FailedJobsHistoryLimit:     scaledJob.FailedJobsHistoryLimit,
				EnvSourceContainerName:     UserContainer,
				MaxReplicaCount:            scaledJob.MaxReplicaCount,
				ScalingStrategy:            scaledJob.ScalingStrategy,
				Triggers:                   scaledJob.Triggers,
			}
		case *kedav1alpha1.ScaledObject:
			if dapr.ScaledObject == nil {
				return fmt.Errorf("ScaledObject is nil")
			}

			ref, err := r.getObjectTargetRef(workload)
			if err != nil {
				return err
			}

			scaledObject := dapr.ScaledObject
			scaler := obj.(*kedav1alpha1.ScaledObject)
			scaler.Spec = kedav1alpha1.ScaledObjectSpec{
				ScaleTargetRef:  ref,
				PollingInterval: scaledObject.PollingInterval,
				CooldownPeriod:  scaledObject.CooldownPeriod,
				MinReplicaCount: scaledObject.MinReplicaCount,
				MaxReplicaCount: scaledObject.MaxReplicaCount,
				Advanced:        scaledObject.Advanced,
				Triggers:        scaledObject.Triggers,
			}

		default:
			return fmt.Errorf("neithor ScaledJob nor scaledObject")
		}

		accessor, _ := meta.Accessor(obj)
		return controllerutil.SetControllerReference(s, accessor, r.Scheme)
	}
}

func (r *ServingReconciler) getJobTargetRef(workload runtime.Object) (*batchv1.JobSpec, error) {

	job, ok := workload.(*batchv1.Job)
	if !ok {
		return nil, fmt.Errorf("workload is not job")
	}

	ref := job.DeepCopy().Spec
	return &ref, nil
}

func (r *ServingReconciler) getObjectTargetRef(workload runtime.Object) (*kedav1alpha1.ScaleTarget, error) {

	accessor, _ := meta.Accessor(workload)
	ref := &kedav1alpha1.ScaleTarget{
		Name:                   accessor.GetName(),
		EnvSourceContainerName: UserContainer,
	}

	switch workload.(type) {
	case *appsv1.Deployment:
		ref.Kind = "Deployment"
	case *appsv1.StatefulSet:
		ref.Kind = "StatefulSet"
	default:
		return nil, fmt.Errorf("workload is neithor deployment nor statefulSet")
	}

	return ref, nil
}

func (r *ServingReconciler) createOrUpdateSvc(s *openfunction.Serving, workload runtime.Object) error {

	log := r.Log.WithName("createOrUpdateSvc")

	svc := &corev1.Service{}
	svc.Name = fmt.Sprintf("%s-%s", s.Name, "svc")
	svc.Namespace = s.Namespace

	if err := r.Delete(r.ctx, svc); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to delete dapr service", "name", s.Name, "namespace", s.Namespace)
		return err
	}

	if err := r.mutateSvc(svc, s, workload)(); err != nil {
		log.Error(err, "Failed to mutate dapr service", "name", s.Name, "namespace", s.Namespace)
		return err
	}

	if err := r.Create(r.ctx, svc); err != nil {
		log.Error(err, "Failed to create dapr service", "name", s.Name, "namespace", s.Namespace)
		return err
	}

	log.V(1).Info("Create service", "name", svc.Name, "namespace", svc.Namespace)
	return nil
}

func (r *ServingReconciler) mutateSvc(svc *corev1.Service, s *openfunction.Serving, workload runtime.Object) controllerutil.MutateFn {

	return func() error {
		svc.Labels = map[string]string{
			"openfunction.io/managed": "",
			"serving":                 s.Name,
			"runtime":                 string(openfunction.DAPR),
		}

		accessor, _ := meta.Accessor(workload)
		svc.Spec.Selector = accessor.GetLabels()

		var port int32 = 8080
		if s.Spec.Port != nil {
			port = *s.Spec.Port
		}

		svc.Spec.Ports = []corev1.ServicePort{
			{
				Name: "serving",
				Port: port,
				TargetPort: intstr.IntOrString{
					IntVal: port,
				},
				Protocol: corev1.ProtocolTCP,
			},
		}
		svc.Spec.Type = corev1.ServiceTypeClusterIP
		return controllerutil.SetControllerReference(s, svc, r.Scheme)
	}
}

func (r *ServingReconciler) createOrUpdateComponents(s *openfunction.Serving) error {
	log := r.Log.WithName("createOrUpdateDaprComponents")

	dapr := s.Spec.Dapr
	if dapr == nil {
		return nil
	}

	for _, dc := range dapr.Components {
		component := &componentsv1alpha1.Component{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dc.Name,
				Namespace: s.Namespace,
			},
		}

		if err := r.Delete(r.ctx, component); util.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to delete dapr component", "serving", s.Name, "component", dc.Name)
			return err
		}

		component.Spec = dc.ComponentSpec

		if err := controllerutil.SetControllerReference(s, component, r.Scheme); err != nil {
			log.Error(err, "Failed to SetControllerReference", "serving", s.Name, "component", dc.Name)
			return err
		}

		if err := r.Create(r.ctx, component); err != nil {
			log.Error(err, "Failed to create dapr component", "serving", s.Name, "component", dc.Name)
			return err
		}
	}

	log.V(1).Info("Create components", "serving", s.Name, "namespace", s.Namespace)
	return nil
}

func (r *ServingReconciler) createOrUpdateSubscriptions(s *openfunction.Serving) error {
	log := r.Log.WithName("createOrUpdateDaprSubscriptions")

	dapr := s.Spec.Dapr
	if dapr == nil {
		return nil
	}

	for _, ds := range dapr.Subscriptions {
		subscription := &subscriptionsv1alpha1.Subscription{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ds.Name,
				Namespace: s.Namespace,
			},
		}

		if err := r.Delete(r.ctx, subscription); util.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to delete dapr subscription", "serving", s.Name, "subscription", ds.Name)
			return err
		}

		subscription.Spec = ds.SubscriptionSpec
		subscription.Scopes = ds.Scopes

		if err := controllerutil.SetControllerReference(s, subscription, r.Scheme); err != nil {
			log.Error(err, "Failed to SetControllerReference", "serving", s.Name, "subscription", ds.Name)
			return err
		}

		if err := r.Create(r.ctx, subscription); err != nil {
			log.Error(err, "Failed to create dapr subscription", "serving", s.Name, "subscription", ds.Name)
			return err
		}
	}

	log.V(1).Info("Createsubscriptions", "serving", s.Name, "namespace", s.Namespace)
	return nil
}
