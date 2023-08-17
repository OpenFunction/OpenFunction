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

package knative

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	kservingv1 "knative.dev/serving/pkg/apis/serving/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	openfunction "github.com/openfunction/apis/core/v1beta2"
	"github.com/openfunction/pkg/constants"
	"github.com/openfunction/pkg/core"
	"github.com/openfunction/pkg/core/serving/common"
	"github.com/openfunction/pkg/util"
)

const (
	knativeServiceKey = "serving.knative.dev/service"
)

type servingRun struct {
	client.Client
	ctx    context.Context
	log    logr.Logger
	scheme *runtime.Scheme
}

func Registry(rm meta.RESTMapper) []client.Object {
	if _, err := rm.ResourcesFor(schema.GroupVersionResource{Group: "serving.knative.dev", Version: "v1", Resource: "services"}); err != nil {
		return nil
	}
	return []client.Object{&kservingv1.Service{}}
}

func NewServingRun(ctx context.Context, c client.Client, scheme *runtime.Scheme, log logr.Logger) core.ServingRun {
	return &servingRun{
		c,
		ctx,
		log.WithName("Knative"),
		scheme,
	}
}

func (r *servingRun) Run(s *openfunction.Serving, cm map[string]string) error {
	log := r.log.WithName("Run").
		WithValues("Serving", fmt.Sprintf("%s/%s", s.Namespace, s.Name))

	if err := r.Clean(s); err != nil {
		log.Error(err, "Clean failed")
		return err
	}

	s.Status.ResourceRef = make(map[string]string)
	if err := common.CreateComponents(r.ctx, r.log, r.Client, r.scheme, s); err != nil {
		return err
	}

	service, err := r.createService(s, cm)
	if err != nil {
		log.Error(err, "Failed to create knative Service")
		return err
	}

	service.SetOwnerReferences(nil)
	if err := ctrl.SetControllerReference(s, service, r.scheme); err != nil {
		log.Error(err, "Failed to SetControllerReference for Service", "Service", service.Name)
		return err
	}

	if err := r.Create(r.ctx, service); err != nil {
		log.Error(err, "Failed to Create Service", "Service", service.Name)
		return err
	}

	log.V(1).Info("Service created", "Service", service.Name)

	if s.Status.ResourceRef == nil {
		s.Status.ResourceRef = make(map[string]string)
	}

	s.Status.ResourceRef[knativeServiceKey] = service.Name
	s.Status.Service = service.Name

	if common.NeedCreateDaprProxy(s) {
		if err := common.CreateDaprProxy(r.ctx, r.log, r.Client, r.scheme, s, cm); err != nil {
			log.Error(err, "Failed to Create dapr proxy", "Service", service.Name)
			return err
		}
	}

	return nil
}

func (r *servingRun) Clean(s *openfunction.Serving) error {
	log := r.log.WithName("Clean").
		WithValues("Serving", fmt.Sprintf("%s/%s", s.Namespace, s.Name))

	services := &kservingv1.ServiceList{}
	if err := r.List(r.ctx, services, client.InNamespace(s.Namespace), client.MatchingLabels{common.ServingLabel: s.Name}); err != nil {
		return err
	}

	for _, item := range services.Items {
		if strings.HasPrefix(item.Name, s.Name) {
			if err := r.Delete(context.Background(), &item); util.IgnoreNotFound(err) != nil {
				return err
			}
			log.V(1).Info("Delete Service", "Service", item.Name)
		}
	}

	if err := common.CleanDaprProxy(r.ctx, log, r.Client, s); err != nil {
		return err
	}

	return nil
}

func (r *servingRun) Result(s *openfunction.Serving) (string, string, string, error) {
	log := r.log.WithName("Result").
		WithValues("Serving", fmt.Sprintf("%s/%s", s.Namespace, s.Name))

	service := &kservingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getName(s, knativeServiceKey),
			Namespace: s.Namespace,
		},
	}

	if err := r.Get(r.ctx, client.ObjectKeyFromObject(service), service); err != nil {
		log.Error(err, "Failed to get Service", "Service", service.Name)
		return "", "", "", err
	}

	if service.IsFailed() {
		condition := service.Status.GetCondition(kservingv1.ServiceConditionReady)
		if condition == nil {
			return openfunction.Failed, "", "", nil
		} else {
			return openfunction.Failed, condition.Reason, condition.Message, nil
		}
	} else if !service.IsReady() {
		return "", "", "", nil
	}

	if common.NeedCreateDaprProxy(s) {
		proxy := &appsv1.Deployment{}
		if err := r.Get(r.ctx, client.ObjectKey{Name: common.GetProxyName(s), Namespace: s.Namespace}, proxy); err != nil {
			return "", "", "", err
		}

		for _, cond := range proxy.Status.Conditions {
			switch cond.Type {
			case appsv1.DeploymentProgressing:
				switch cond.Status {
				case corev1.ConditionUnknown, corev1.ConditionFalse:
					return "", "", "", nil
				}
			case appsv1.DeploymentReplicaFailure:
				switch cond.Status {
				case corev1.ConditionUnknown, corev1.ConditionTrue:
					return "", "", "", nil
				}
			}
		}
	}

	return openfunction.Running, openfunction.Running, openfunction.Running, nil
}

func (r *servingRun) createService(s *openfunction.Serving, cm map[string]string) (*kservingv1.Service, error) {
	version := constants.DefaultFunctionVersion
	if s.Spec.Version != nil {
		version = *s.Spec.Version
	}
	labels := map[string]string{
		common.ServingLabel:          s.Name,
		constants.CommonLabelVersion: version,
	}
	labels = util.AppendLabels(s.Spec.Labels, labels)

	// Handle the scale options, which have the following priority relationship:
	// ScaleOptions.Knative["autoscaling.knative.dev/max-scale" ("autoscaling.knative.dev/min-scale")] >
	// ScaleOptions.maxScale (minScale) >
	// Annotations["autoscaling.knative.dev/max-scale" ("autoscaling.knative.dev/min-scale")] >
	// And in Knative Serving v1.1, the scale bounds' name were changed from "maxScale" ("minScale") to "max-scale" ("min-scale"),
	// we need to support both of these layouts.
	if s.Spec.ScaleOptions != nil {
		maxScale := ""
		minScale := ""

		if s.Spec.Annotations == nil {
			s.Spec.Annotations = map[string]string{}
		}

		if s.Spec.ScaleOptions.MaxReplicas != nil {
			maxScale = strconv.Itoa(int(*s.Spec.ScaleOptions.MaxReplicas))
		}
		if s.Spec.ScaleOptions.MinReplicas != nil {
			minScale = strconv.Itoa(int(*s.Spec.ScaleOptions.MinReplicas))
		}

		if s.Spec.ScaleOptions.Knative != nil {
			s.Spec.Annotations = util.AppendLabels(s.Spec.Annotations, *s.Spec.ScaleOptions.Knative)
		}

		if maxScale != "" {
			s.Spec.Annotations["autoscaling.knative.dev/maxScale"] = maxScale
			s.Spec.Annotations["autoscaling.knative.dev/max-scale"] = maxScale
		}

		if minScale != "" {
			s.Spec.Annotations["autoscaling.knative.dev/minScale"] = minScale
			s.Spec.Annotations["autoscaling.knative.dev/min-scale"] = minScale
		}
	}

	var appPort = int32(constants.DefaultFuncPort)
	port := corev1.ContainerPort{}
	if s.Spec.Triggers.Http.Port != nil {
		appPort = *s.Spec.Triggers.Http.Port
		port.ContainerPort = *s.Spec.Triggers.Http.Port
	}

	annotations := make(map[string]string)
	annotations[common.DaprAppID] = fmt.Sprintf("%s-%s", common.GetFunctionName(s), s.Namespace)
	annotations[common.DaprLogAsJSON] = "true"
	// The dapr protocol must equal to the protocol of function framework.
	annotations[common.DaprAppProtocol] = "grpc"
	// The dapr port must equal the function port.
	annotations[common.DaprAppPort] = fmt.Sprintf("%d", appPort)
	annotations[common.DaprMetricsPort] = "19090"
	annotations = util.AppendLabels(s.Spec.Annotations, annotations)

	if common.NeedCreateDaprSidecar(s) == false {
		annotations[common.DaprEnabled] = "false"
	} else {
		annotations[common.DaprEnabled] = "true"
	}

	template := s.Spec.Template
	if template == nil {
		template = &corev1.PodSpec{}
	}

	if s.Spec.ImageCredentials != nil {
		template.ImagePullSecrets = append(template.ImagePullSecrets, *s.Spec.ImageCredentials)
	}

	var container *corev1.Container
	for index := range template.Containers {
		if template.Containers[index].Name == core.FunctionContainer {
			container = &template.Containers[index]
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

	container.Ports = append(container.Ports, port)

	container.Env = append(container.Env, corev1.EnvVar{
		Name:  common.DaprProtocolEnvVar,
		Value: annotations[common.DaprAppProtocol],
	})
	container.Env = append(container.Env, common.GetSkywalkingEnv(r.log, s, cm)...)

	if env, err := common.CreateFunctionContextENV(r.ctx, r.log, r.Client, s, cm); err != nil {
		return nil, err
	} else {
		container.Env = append(container.Env, env...)
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

	if common.NeedCreateDaprProxy(s) {
		daprServiceName := fmt.Sprintf("%s-dapr", annotations[common.DaprAppID])
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

	if appended {
		template.Containers = append(template.Containers, *container)
	}

	if _, ok := s.Annotations[constants.WasmVariantAnnotation]; ok && template.RuntimeClassName == nil {
		runtimeClassName := constants.WasmEdgeRuntimeClassName
		template.RuntimeClassName = &runtimeClassName
	}

	rand.Seed(time.Now().UnixNano())
	serviceName := fmt.Sprintf("%s-ksvc-%s", s.Name, rand.String(5))
	workloadName := serviceName
	workloadName = fmt.Sprintf("%s-%s", workloadName, strings.ReplaceAll(version, ".", ""))

	// Handle hard limit, this setting is not an annotation.
	// The hard limit is specified per Revision using the containerConcurrency field on the Revision spec.
	var containerConcurrency *int64
	if val, ok := s.Spec.Annotations["autoscaling.knative.dev/container-concurrency"]; ok {
		c, err := strconv.ParseInt(val, 10, 64)
		if err == nil {
			containerConcurrency = &c
		}
	}
	service := kservingv1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "serving.knative.dev/v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: s.Namespace,
			Labels: map[string]string{
				common.ServingLabel: s.Name,
			},
		},
		Spec: kservingv1.ServiceSpec{
			ConfigurationSpec: kservingv1.ConfigurationSpec{
				Template: kservingv1.RevisionTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:        workloadName,
						Namespace:   s.Namespace,
						Labels:      labels,
						Annotations: annotations,
					},
					Spec: kservingv1.RevisionSpec{
						ContainerConcurrency: containerConcurrency,
						PodSpec:              *template,
					},
				},
			},
		},
	}

	return &service, nil
}

func getName(s *openfunction.Serving, key string) string {
	if s.Status.ResourceRef == nil {
		return ""
	}

	return s.Status.ResourceRef[key]
}
