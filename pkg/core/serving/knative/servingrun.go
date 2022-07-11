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

	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	kservingv1 "knative.dev/serving/pkg/apis/serving/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	openfunction "github.com/openfunction/apis/core/v1beta1"
	"github.com/openfunction/pkg/constants"
	"github.com/openfunction/pkg/core"
	"github.com/openfunction/pkg/core/serving/common"
	"github.com/openfunction/pkg/util"
)

const (
	knativeService = "serving.knative.dev/service"
	componentName  = "Knative/component"
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

	service := r.createService(s, cm, pendingComponents)
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

	s.Status.ResourceRef[knativeService] = service.Name
	s.Status.Service = service.Name

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

	return nil
}

func (r *servingRun) Result(s *openfunction.Serving) (string, error) {
	log := r.log.WithName("Result").
		WithValues("Serving", fmt.Sprintf("%s/%s", s.Namespace, s.Name))

	service := &kservingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getName(s, knativeService),
			Namespace: s.Namespace,
		},
	}

	if err := r.Get(r.ctx, client.ObjectKeyFromObject(service), service); err != nil {
		log.Error(err, "Failed to get Service", "Service", service.Name)
		return "", err
	}

	if service.IsReady() {
		return openfunction.Running, nil
	} else if service.IsFailed() {
		return openfunction.Failed, nil
	} else {
		return "", nil
	}
}

func (r *servingRun) createService(s *openfunction.Serving, cm map[string]string, components map[string]*componentsv1alpha1.ComponentSpec) *kservingv1.Service {

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

	var appPort int32 = 8080
	port := corev1.ContainerPort{}
	if s.Spec.Port != nil {
		appPort = *s.Spec.Port
		port.ContainerPort = *s.Spec.Port
		container.Ports = append(container.Ports, port)
	}

	container.Env = append(container.Env, corev1.EnvVar{
		Name:  common.FunctionContextEnvName,
		Value: common.GenOpenFunctionContext(r.ctx, r.log, s, cm, components, getFunctionName(s), componentName),
	})

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
		template.Containers = append(template.Containers, *container)
	}

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
			for k, v := range *s.Spec.ScaleOptions.Knative {
				switch k {
				case "autoscaling.knative.dev/max-scale", "autoscaling.knative.dev/maxScale":
					maxScale = v
				case "autoscaling.knative.dev/min-scale", "autoscaling.knative.dev/minScale":
					minScale = v
				}
			}
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

	if s.Spec.Outputs != nil {
		if s.Spec.Annotations == nil {
			s.Spec.Annotations = map[string]string{}
		}
		s.Spec.Annotations[common.DaprEnabled] = "true"
		s.Spec.Annotations[common.DaprAppID] = fmt.Sprintf("%s-%s", getFunctionName(s), s.Namespace)
		s.Spec.Annotations[common.DaprLogAsJSON] = "true"

		// The dapr protocol must equal to the protocol of function framework.
		s.Spec.Annotations[common.DaprAppProtocol] = "grpc"
		// The dapr port must equal the function port.
		s.Spec.Annotations[common.DaprAppPort] = fmt.Sprintf("%d", appPort)
		s.Spec.Annotations[common.DaprMetricsPort] = "19090"
	}

	rand.Seed(time.Now().UnixNano())
	serviceName := fmt.Sprintf("%s-ksvc-%s", s.Name, rand.String(5))
	workloadName := serviceName
	workloadName = fmt.Sprintf("%s-%s", workloadName, strings.ReplaceAll(version, ".", ""))

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
						Annotations: s.Spec.Annotations,
					},
					Spec: kservingv1.RevisionSpec{
						PodSpec: *template,
					},
				},
			},
		},
	}

	return &service
}

func getFunctionName(s *openfunction.Serving) string {

	return s.Labels[constants.FunctionLabel]
}

func getName(s *openfunction.Serving, key string) string {
	if s.Status.ResourceRef == nil {
		return ""
	}

	return s.Status.ResourceRef[key]
}
