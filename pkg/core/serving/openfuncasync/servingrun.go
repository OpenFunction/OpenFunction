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

package openfuncasync

import (
	"context"
	"fmt"
	"strings"

	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"github.com/go-logr/logr"
	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	openfunction "github.com/openfunction/apis/core/v1beta1"
	"github.com/openfunction/pkg/constants"
	"github.com/openfunction/pkg/core"
	"github.com/openfunction/pkg/core/serving/common"
	"github.com/openfunction/pkg/util"
)

const (
	runtimeLabel = "runtime"

	workloadName  = "Async/workload"
	scalerName    = "Async/scaler"
	componentName = "Async/component"
)

type servingRun struct {
	client.Client
	ctx    context.Context
	log    logr.Logger
	scheme *runtime.Scheme
}

func Registry(rm meta.RESTMapper) []client.Object {
	var objs = []client.Object{&appsv1.Deployment{}, &appsv1.StatefulSet{}, &batchv1.Job{}}
	if _, err := rm.ResourcesFor(schema.GroupVersionResource{Group: "keda.sh", Version: "v1alpha1", Resource: "scaledobjects"}); err == nil {
		objs = append(objs, &kedav1alpha1.ScaledObject{})
	}

	if _, err := rm.ResourcesFor(schema.GroupVersionResource{Group: "keda.sh", Version: "v1alpha1", Resource: "scaledjobs"}); err == nil {
		objs = append(objs, &kedav1alpha1.ScaledJob{})
	}

	if _, err := rm.ResourcesFor(schema.GroupVersionResource{Group: "dapr.io", Version: "v1alpha1", Resource: "components"}); err == nil {
		objs = append(objs, &componentsv1alpha1.Component{})
	}

	return objs
}

func NewServingRun(ctx context.Context, c client.Client, scheme *runtime.Scheme, log logr.Logger) core.ServingRun {
	return &servingRun{
		c,
		ctx,
		log.WithName("OpenFuncAsync"),
		scheme,
	}
}

func (r *servingRun) Run(s *openfunction.Serving, cm map[string]string) error {

	log := r.log.WithName("Run").
		WithValues("Serving", fmt.Sprintf("%s/%s", s.Namespace, s.Name))

	if err := r.Clean(s); err != nil {
		log.Error(err, "Failed to Clean")
		return err
	}

	pendingComponents, err := common.GetPendingCreateComponents(s)
	if err != nil {
		log.Error(err, "Failed to get pending create components")
		return err
	}

	if err := common.CheckComponentSpecExist(s, pendingComponents); err != nil {
		log.Error(err, "Some Components does not exist")
		return err
	}

	s.Status.ResourceRef = make(map[string]string)
	if err := common.CreateComponents(r.ctx, r.log, r.Client, r.scheme, s, pendingComponents, componentName); err != nil {
		log.Error(err, "Failed to create Dapr Components")
		return err
	}

	workload := r.generateWorkload(s, cm, pendingComponents)
	if err := controllerutil.SetControllerReference(s, workload, r.scheme); err != nil {
		log.Error(err, "Failed to SetControllerReference for workload")
		return err
	}

	if err := r.Create(r.ctx, workload); err != nil {
		log.Error(err, "Failed to create workload")
		return err
	}

	log.V(1).Info("Workload created", "Workload", workload.GetName())

	s.Status.ResourceRef[workloadName] = workload.GetName()

	if err := r.createScaler(s, workload); err != nil {
		log.Error(err, "Failed to create Keda triggers")
		return err
	}

	if common.NeedCreateDaprProxy(s) {
		if err := r.createService(s); err != nil {
			return err
		}
		if err := common.CreateDaprProxy(r.ctx, r.log, r.Client, r.scheme, s, cm, pendingComponents, componentName); err != nil {
			return err
		}
	}

	return nil
}

func (r *servingRun) Clean(s *openfunction.Serving) error {
	log := r.log.WithName("Clean").
		WithValues("Serving", fmt.Sprintf("%s/%s", s.Namespace, s.Name))

	list := func(lists []client.ObjectList) error {
		for _, list := range lists {
			if err := r.List(r.ctx, list, client.InNamespace(s.Namespace), client.MatchingLabels{common.ServingLabel: s.Name}); err != nil {
				return err
			}
		}

		return nil
	}

	deleteObj := func(obj client.Object) error {
		if strings.HasPrefix(obj.GetName(), s.Name) {
			if err := r.Delete(context.Background(), obj); util.IgnoreNotFound(err) != nil {
				return err
			}
			log.V(1).Info("Delete", "name", obj.GetName())
		}

		return nil
	}

	jobList := &batchv1.JobList{}
	deploymentList := &appsv1.DeploymentList{}
	statefulSetList := &appsv1.StatefulSetList{}
	scalerJobList := &kedav1alpha1.ScaledJobList{}
	scaledObjectList := &kedav1alpha1.ScaledObjectList{}
	serviceList := &corev1.ServiceList{}
	componentList := &componentsv1alpha1.ComponentList{}

	if err := list([]client.ObjectList{jobList, deploymentList, statefulSetList, scalerJobList, scaledObjectList, serviceList, componentList}); err != nil {
		return err
	}

	for _, item := range jobList.Items {
		if err := deleteObj(&item); err != nil {
			return err
		}
	}

	for _, item := range deploymentList.Items {
		if err := deleteObj(&item); err != nil {
			return err
		}
	}

	for _, item := range statefulSetList.Items {
		if err := deleteObj(&item); err != nil {
			return err
		}
	}

	for _, item := range scalerJobList.Items {
		if err := deleteObj(&item); err != nil {
			return err
		}
	}

	for _, item := range scaledObjectList.Items {
		if err := deleteObj(&item); err != nil {
			return err
		}
	}

	for _, item := range scaledObjectList.Items {
		if err := deleteObj(&item); err != nil {
			return err
		}
	}

	for _, item := range componentList.Items {
		if err := deleteObj(&item); err != nil {
			return err
		}
	}

	if err := common.CleanDaprProxy(r.ctx, log, r.Client, s); err != nil {
		return err
	}

	return nil
}

func (r *servingRun) Result(s *openfunction.Serving) (string, error) {

	// Currently, it only supports updating the status of serving through the status of deployment.
	if s.Spec.ScaleOptions != nil {
		if s.Spec.ScaleOptions.Keda == nil || s.Spec.ScaleOptions.Keda.ScaledObject == nil ||
			s.Spec.ScaleOptions.Keda.ScaledObject.WorkloadType == "StatefulSet" {
			return openfunction.Running, nil
		}
	}

	deploy := &appsv1.Deployment{}
	if err := r.Get(r.ctx, client.ObjectKey{Name: getWorkloadName(s), Namespace: s.Namespace}, deploy); err != nil {
		return "", err
	}

	for _, cond := range deploy.Status.Conditions {
		switch cond.Type {
		case appsv1.DeploymentProgressing:
			switch cond.Status {
			case corev1.ConditionUnknown, corev1.ConditionFalse:
				return "", nil
			}
		case appsv1.DeploymentReplicaFailure:
			switch cond.Status {
			case corev1.ConditionUnknown, corev1.ConditionTrue:
				return "", nil
			}
		}
	}

	if common.NeedCreateDaprProxy(s) {
		proxy := &appsv1.Deployment{}
		if err := r.Get(r.ctx, client.ObjectKey{Name: common.GetProxyName(s), Namespace: s.Namespace}, proxy); err != nil {
			return "", err
		}

		for _, cond := range proxy.Status.Conditions {
			switch cond.Type {
			case appsv1.DeploymentProgressing:
				switch cond.Status {
				case corev1.ConditionUnknown, corev1.ConditionFalse:
					return "", nil
				}
			case appsv1.DeploymentReplicaFailure:
				switch cond.Status {
				case corev1.ConditionUnknown, corev1.ConditionTrue:
					return "", nil
				}
			}
		}
	}

	return openfunction.Running, nil
}

func (r *servingRun) generateWorkload(s *openfunction.Serving, cm map[string]string, components map[string]*componentsv1alpha1.ComponentSpec) client.Object {

	version := constants.DefaultFunctionVersion
	if s.Spec.Version != nil {
		version = *s.Spec.Version
	}

	labels := map[string]string{
		common.OpenfunctionManaged:   "true",
		common.ServingLabel:          s.Name,
		runtimeLabel:                 string(openfunction.Async),
		constants.CommonLabelVersion: version,
	}

	labels = util.AppendLabels(s.Spec.Labels, labels)

	selector := &metav1.LabelSelector{
		MatchLabels: labels,
	}

	var keda *openfunction.KedaScaleOptions
	var replicas int32 = 1
	restartPolicy := corev1.RestartPolicyOnFailure
	if s.Spec.ScaleOptions != nil && s.Spec.ScaleOptions.Keda != nil {
		keda = s.Spec.ScaleOptions.Keda
		if keda.ScaledObject != nil {
			if s.Spec.ScaleOptions.MaxReplicas != nil && keda.ScaledObject.MaxReplicaCount == nil {
				keda.ScaledObject.MaxReplicaCount = s.Spec.ScaleOptions.MaxReplicas
			}
			if s.Spec.ScaleOptions.MinReplicas != nil && keda.ScaledObject.MinReplicaCount == nil {
				keda.ScaledObject.MinReplicaCount = s.Spec.ScaleOptions.MinReplicas
			}
			if keda.ScaledObject.MinReplicaCount != nil {
				replicas = *keda.ScaledObject.MinReplicaCount
			}
		}
		if keda.ScaledJob != nil && keda.ScaledJob.RestartPolicy != nil {
			restartPolicy = *keda.ScaledJob.RestartPolicy
		}
	}

	var port = int32(constants.DefaultFuncPort)
	if s.Spec.Port != nil {
		port = *s.Spec.Port
	}

	annotations := make(map[string]string)
	annotations[common.DaprAppID] = fmt.Sprintf("%s-%s", common.GetFunctionName(s), s.Namespace)
	annotations[common.DaprLogAsJSON] = "true"
	// The dapr protocol must equal to the protocol of function framework.
	annotations[common.DaprAppProtocol] = "grpc"
	// The dapr port must equal the function port.
	annotations[common.DaprAppPort] = fmt.Sprintf("%d", port)
	annotations = util.AppendLabels(s.Spec.Annotations, annotations)

	if common.NeedCreateDaprSidecar(s) == false {
		annotations[common.DaprEnabled] = "false"
	} else {
		annotations[common.DaprEnabled] = "true"
	}

	spec := s.Spec.Template
	if spec == nil {
		spec = &corev1.PodSpec{}
	}

	if s.Spec.ImageCredentials != nil {
		spec.ImagePullSecrets = append(spec.ImagePullSecrets, *s.Spec.ImageCredentials)
	}

	var container *corev1.Container
	for index := range spec.Containers {
		if spec.Containers[index].Name == core.FunctionContainer {
			container = &spec.Containers[index]
		}
	}

	appended := false
	if container == nil {
		container = &corev1.Container{
			Name:            core.FunctionContainer,
			ImagePullPolicy: corev1.PullIfNotPresent,
		}
		appended = true
	}

	container.Image = s.Spec.Image

	container.Ports = append(container.Ports, corev1.ContainerPort{
		Name:          core.FunctionPort,
		ContainerPort: port,
		Protocol:      corev1.ProtocolTCP,
	})

	container.Env = append(container.Env, []corev1.EnvVar{
		{
			Name:  common.FunctionContextEnvName,
			Value: common.GenOpenFunctionContext(r.ctx, r.log, s, cm, components, common.GetFunctionName(s), componentName),
		},
		{
			Name:  common.DaprProtocolEnvVar,
			Value: annotations[common.DaprAppProtocol],
		},
	}...)

	if common.NeedCreateDaprProxy(s) {
		funcName := common.GetFunctionName(s)
		daprServiceName := fmt.Sprintf("%s-%s-dapr", funcName, s.Namespace)
		container.Env = append(container.Env, []corev1.EnvVar{
			{
				Name:  common.DaprHostEnvVar,
				Value: fmt.Sprintf("%s.%s.svc.cluster.local", daprServiceName, s.Namespace),
			},
			{
				Name:  common.DaprSidecarIPEnvVar,
				Value: fmt.Sprintf("%s.%s.svc.cluster.local", daprServiceName, s.Namespace),
			},
		}...)
	}

	if s.Spec.Params != nil {
		for k, v := range s.Spec.Params {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  k,
				Value: v,
			})
		}
	}
	container.Env = append(container.Env, common.AddPodMetadataEnv(s.Namespace)...)

	if appended {
		spec.Containers = append(spec.Containers, *container)
	}

	template := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: *spec,
	}

	version = strings.ReplaceAll(version, ".", "")
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-deployment-%s-", s.Name, version),
			Namespace:    s.Namespace,
			Labels:       labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: selector,
			Template: template,
		},
	}

	statefulset := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-statefulset-%s-", s.Name, version),
			Namespace:    s.Namespace,
			Labels:       labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: selector,
			Template: template,
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-job-%s-", s.Name, version),
			Namespace:    s.Namespace,
			Labels:       labels,
		},
		Spec: batchv1.JobSpec{
			Template: template,
		},
	}

	job.Spec.Template.Spec.RestartPolicy = restartPolicy

	// By default, use deployment to running the function.
	if keda == nil || (keda.ScaledJob == nil && keda.ScaledObject == nil) {
		return deploy
	} else {
		if keda.ScaledJob != nil {
			return job
		} else {
			if keda.ScaledObject.WorkloadType == "StatefulSet" {
				return statefulset
			} else {
				return deploy
			}
		}
	}
}

func (r *servingRun) createService(s *openfunction.Serving) error {
	funcName := common.GetFunctionName(s)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Namespace: s.Namespace, Name: funcName},
	}
	op, err := controllerutil.CreateOrUpdate(r.ctx, r.Client, service, r.mutateService(s, service))
	if err != nil {
		r.log.Error(err, "Failed to CreateOrUpdate service")
		return err
	}
	r.log.V(1).Info(fmt.Sprintf("Service %s", op))
	return nil
}

func (r *servingRun) mutateService(s *openfunction.Serving, service *corev1.Service) controllerutil.MutateFn {
	var port = int32(constants.DefaultFuncPort)
	if s.Spec.Port != nil {
		port = *s.Spec.Port
	}
	return func() error {
		funcPort := corev1.ServicePort{
			Name:       "http",
			Protocol:   corev1.ProtocolTCP,
			Port:       port,
			TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: port},
		}
		if s.Annotations[common.DaprAppProtocol] != "http" {
			service.Spec.ClusterIP = corev1.ClusterIPNone
		}
		selector := map[string]string{common.ServingLabel: s.Name}
		service.Spec.Ports = []corev1.ServicePort{funcPort}
		service.Spec.Selector = selector
		service.SetOwnerReferences(nil)
		return ctrl.SetControllerReference(s, service, r.scheme)
	}
}

func (r *servingRun) createScaler(s *openfunction.Serving, workload runtime.Object) error {
	log := r.log.WithName("CreateKedaScaler").
		WithValues("Serving", fmt.Sprintf("%s/%s", s.Namespace, s.Name))

	// When no Triggers are configured, it means that no scaler needs to be created for the function.
	if s.Spec.Triggers == nil {
		log.Info("No triggers found, no need to create scaler.")
		return nil
	}

	var scaledJobTriggers []kedav1alpha1.ScaleTriggers
	var scaledObjectTriggers []kedav1alpha1.ScaleTriggers
	for _, triggers := range s.Spec.Triggers {
		t := triggers.DeepCopy()
		if t.TargetKind != nil && *t.TargetKind == openfunction.ScaledJob {
			scaledJobTriggers = append(scaledJobTriggers, t.ScaleTriggers)
		} else {
			scaledObjectTriggers = append(scaledObjectTriggers, t.ScaleTriggers)
		}
	}

	var keda *openfunction.KedaScaleOptions

	if s.Spec.ScaleOptions != nil && s.Spec.ScaleOptions.Keda != nil {
		keda = s.Spec.ScaleOptions.Keda
	}

	if keda == nil || (keda.ScaledJob == nil && keda.ScaledObject == nil) {
		return nil
	}

	var obj client.Object
	if keda.ScaledJob != nil && scaledJobTriggers != nil {
		ref, err := r.getJobTargetRef(workload)
		if err != nil {
			return err
		}

		scaledJob := keda.ScaledJob
		obj = &kedav1alpha1.ScaledJob{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-scaler-", s.Name),
				Namespace:    s.Namespace,
				Labels: map[string]string{
					common.OpenfunctionManaged: "true",
					common.ServingLabel:        s.Name,
					runtimeLabel:               string(openfunction.Async),
				},
			},
			Spec: kedav1alpha1.ScaledJobSpec{
				JobTargetRef:               ref,
				PollingInterval:            scaledJob.PollingInterval,
				SuccessfulJobsHistoryLimit: scaledJob.SuccessfulJobsHistoryLimit,
				FailedJobsHistoryLimit:     scaledJob.FailedJobsHistoryLimit,
				EnvSourceContainerName:     core.FunctionContainer,
				MaxReplicaCount:            scaledJob.MaxReplicaCount,
				ScalingStrategy:            scaledJob.ScalingStrategy,
				Triggers:                   scaledJobTriggers,
			},
		}
	} else if keda.ScaledObject != nil && scaledObjectTriggers != nil {
		ref, err := r.getObjectTargetRef(workload)
		if err != nil {
			return err
		}

		scaledObject := keda.ScaledObject
		obj = &kedav1alpha1.ScaledObject{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-scaler-", s.Name),
				Namespace:    s.Namespace,
				Labels: map[string]string{
					common.OpenfunctionManaged: "true",
					common.ServingLabel:        s.Name,
					runtimeLabel:               string(openfunction.Async),
				},
			},
			Spec: kedav1alpha1.ScaledObjectSpec{
				ScaleTargetRef:  ref,
				PollingInterval: scaledObject.PollingInterval,
				CooldownPeriod:  scaledObject.CooldownPeriod,
				MinReplicaCount: scaledObject.MinReplicaCount,
				MaxReplicaCount: scaledObject.MaxReplicaCount,
				Advanced:        scaledObject.Advanced,
				Triggers:        scaledObjectTriggers,
			},
		}
	}

	if err := controllerutil.SetControllerReference(s, obj, r.scheme); err != nil {
		log.Error(err, "Failed to SetControllerReference")
		return err
	}

	if err := r.Create(r.ctx, obj); err != nil {
		log.Error(err, "Failed to create Keda scaler")
		return err
	}

	s.Status.ResourceRef[scalerName] = obj.GetName()

	log.V(1).Info("Keda scaler Created", "Scaler", obj.GetName())
	return nil
}

func (r *servingRun) getJobTargetRef(workload runtime.Object) (*batchv1.JobSpec, error) {

	job, ok := workload.(*batchv1.Job)
	if !ok {
		return nil, fmt.Errorf("%s", "Workload is not job")
	}

	ref := job.DeepCopy().Spec
	return &ref, nil
}

func (r *servingRun) getObjectTargetRef(workload runtime.Object) (*kedav1alpha1.ScaleTarget, error) {

	accessor, _ := meta.Accessor(workload)
	ref := &kedav1alpha1.ScaleTarget{
		Name:                   accessor.GetName(),
		EnvSourceContainerName: core.FunctionContainer,
	}

	switch workload.(type) {
	case *appsv1.Deployment:
		ref.Kind = "Deployment"
	case *appsv1.StatefulSet:
		ref.Kind = "StatefulSet"
	default:
		return nil, fmt.Errorf("%s", "Workload is neithor deployment nor statefulSet")
	}

	return ref, nil
}

func (r *servingRun) createComponents(s *openfunction.Serving, components map[string]*componentsv1alpha1.ComponentSpec) error {
	log := r.log.WithName("CreateDaprComponents").
		WithValues("Serving", fmt.Sprintf("%s/%s", s.Namespace, s.Name))

	if components == nil {
		return nil
	}

	value := ""
	for name, daprComponent := range components {
		dc := daprComponent.DeepCopy()
		component := &componentsv1alpha1.Component{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-component-%s-", s.Name, name),
				Namespace:    s.Namespace,
				Labels: map[string]string{
					common.OpenfunctionManaged: "true",
					common.ServingLabel:        s.Name,
				},
			},
		}

		if dc != nil {
			component.Spec = *dc
		}

		if err := controllerutil.SetControllerReference(s, component, r.scheme); err != nil {
			log.Error(err, "Failed to SetControllerReference", "Component", name)
			return err
		}

		if err := r.Create(r.ctx, component); err != nil {
			log.Error(err, "Failed to Create Dapr Component", "Component", name)
			return err
		}

		value = fmt.Sprintf("%s%s,", value, component.Name)
		log.V(1).Info("Component Created", "Component", component.Name)
	}

	if value != "" {
		s.Status.ResourceRef[componentName] = strings.TrimSuffix(value, ",")
	}

	return nil
}

func getWorkloadName(s *openfunction.Serving) string {
	if s.Status.ResourceRef == nil {
		return ""
	}

	return s.Status.ResourceRef[workloadName]
}
