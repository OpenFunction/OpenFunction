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

package kedahttp

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"github.com/go-logr/logr"
	httpv1alpha1 "github.com/kedacore/http-add-on/operator/apis/http/v1alpha1"

	openfunction "github.com/openfunction/apis/core/v1beta2"
	"github.com/openfunction/pkg/constants"
	"github.com/openfunction/pkg/core"
	"github.com/openfunction/pkg/core/serving/common"
	"github.com/openfunction/pkg/util"
)

const (
	workloadName = "http.keda.sh/deployment"
	scalerName   = "http.keda.sh/httpscaledobject"
)

type servingRun struct {
	client.Client
	ctx    context.Context
	log    logr.Logger
	scheme *runtime.Scheme
}

func Registry(rm meta.RESTMapper) []client.Object {
	var objs = []client.Object{&appsv1.Deployment{}}
	if _, err := rm.ResourcesFor(schema.GroupVersionResource{Group: "http.keda.sh", Version: "v1alpha1", Resource: "httpscaledobjects"}); err == nil {
		objs = append(objs, &httpv1alpha1.HTTPScaledObject{})
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
		log.WithName("KedaHttp"),
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

	s.Status.ResourceRef = make(map[string]string)
	if err := common.CreateComponents(r.ctx, r.log, r.Client, r.scheme, s); err != nil {
		log.Error(err, "Failed to create Dapr Components")
		return err
	}

	workload, err := r.generateWorkload(s, cm)
	if err != nil {
		log.Error(err, "Failed to generate workload")
		return err
	}

	if err := controllerutil.SetControllerReference(s, workload, r.scheme); err != nil {
		log.Error(err, "Failed to SetControllerReference for workload")
		return err
	}

	if err := r.Create(r.ctx, workload); err != nil {
		log.Error(err, "Failed to create workload", "workload", workload.GetName())
		return err
	}

	log.V(1).Info("Workload created", "Workload", workload.GetName())

	if s.Status.ResourceRef == nil {
		s.Status.ResourceRef = make(map[string]string)
	}

	s.Status.ResourceRef[workloadName] = workload.GetName()

	service, err := r.generateService(s)
	if err != nil {
		log.Error(err, "Failed to generate service")
		return err
	}

	service.SetOwnerReferences(nil)
	if err := controllerutil.SetControllerReference(s, service, r.scheme); err != nil {
		log.Error(err, "Failed to SetControllerReference for service", "service", service.Name)
		return err
	}

	if err := r.Create(r.ctx, service); err != nil {
		log.Error(err, "Failed to create service", "service", service.Name)
		return err
	}

	log.V(1).Info("Service created", "Service", service.Name)

	s.Status.Service = service.Name

	if err := r.createScaler(s, workload, service); err != nil {
		log.Error(err, "Failed to create Keda scaler")
		return err
	}

	if common.NeedCreateDaprProxy(s) {
		if err := common.CreateDaprProxy(r.ctx, r.log, r.Client, r.scheme, s, cm); err != nil {
			log.Error(err, "Failed to Create dapr proxy", "HttpScaledObject", workload.GetName())
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

	deploymentList := &appsv1.DeploymentList{}
	httpScaledObjectList := &httpv1alpha1.HTTPScaledObjectList{}
	serviceList := &corev1.ServiceList{}

	if err := list([]client.ObjectList{deploymentList, httpScaledObjectList, serviceList}); err != nil {
		return err
	}

	for _, item := range httpScaledObjectList.Items {
		if err := deleteObj(&item); err != nil {
			return err
		}
	}

	for _, item := range serviceList.Items {
		if err := deleteObj(&item); err != nil {
			return err
		}
	}

	for _, item := range deploymentList.Items {
		if err := deleteObj(&item); err != nil {
			return err
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

	deploy := &appsv1.Deployment{}
	if err := r.Get(r.ctx, client.ObjectKey{Name: getWorkloadName(s), Namespace: s.Namespace}, deploy); err != nil {
		log.Error(err, "Failed to get Deployment", "Deployment", deploy.Name)
		return "", "", "", err
	}

	for _, condition := range deploy.Status.Conditions {
		switch condition.Status {
		case corev1.ConditionUnknown:
			return "", "", "", nil
		case corev1.ConditionFalse:
			return "", condition.Reason, condition.Message, nil
		}
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

func (r *servingRun) generateWorkload(s *openfunction.Serving, cm map[string]string) (client.Object, error) {
	version := constants.DefaultFunctionVersion
	if s.Spec.Version != nil {
		version = *s.Spec.Version
	}
	version = strings.ReplaceAll(version, ".", "")

	labels := map[string]string{
		common.OpenfunctionManaged:   "true",
		common.ServingLabel:          s.Name,
		constants.CommonLabelVersion: version,
	}

	labels = util.AppendLabels(s.Spec.Labels, labels)

	selector := &metav1.LabelSelector{
		MatchLabels: labels,
	}

	var port = int32(constants.DefaultFuncPort)
	if s.Spec.Triggers.Http.Port != nil {
		port = *s.Spec.Triggers.Http.Port
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

	container.Env = append(container.Env, corev1.EnvVar{
		Name:  common.DaprProtocolEnvVar,
		Value: annotations[common.DaprAppProtocol],
	})

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
		spec.Containers = append(spec.Containers, *container)
	}

	if _, ok := s.Annotations[constants.WasmVariantAnnotation]; ok && spec.RuntimeClassName == nil {
		runtimeClassName := constants.WasmEdgeRuntimeClassName
		spec.RuntimeClassName = &runtimeClassName
	}

	template := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: *spec,
	}

	var replicas int32 = 1
	if s.Spec.ScaleOptions != nil && s.Spec.ScaleOptions.Keda != nil {
		if s.Spec.ScaleOptions.MinReplicas != nil {
			num := *s.Spec.ScaleOptions.MinReplicas
			if num > 0 {
				replicas = num
			}
		}
	}

	// In current version of keda http-addon(v0.5.0), Deployment is the only available workload
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

	return deploy, nil
}

func (r *servingRun) generateService(s *openfunction.Serving) (*corev1.Service, error) {
	version := constants.DefaultFunctionVersion
	if s.Spec.Version != nil {
		version = *s.Spec.Version
	}
	version = strings.ReplaceAll(version, ".", "")

	labels := map[string]string{
		common.OpenfunctionManaged:   "true",
		common.ServingLabel:          s.Name,
		constants.CommonLabelVersion: version,
	}

	labels = util.AppendLabels(s.Spec.Labels, labels)

	svcPort := corev1.ServicePort{
		Port: 80, // Default to 80(HTTP), no need to change
		TargetPort: intstr.IntOrString{
			IntVal: *s.Spec.Triggers.Http.Port,
		},
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-service-%s-", s.Name, version),
			Namespace:    s.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports:    []corev1.ServicePort{svcPort},
			Selector: labels,
		},
	}

	return service, nil
}

func (r *servingRun) createScaler(s *openfunction.Serving, workload runtime.Object, service *corev1.Service) error {
	log := r.log.WithName("CreateKedaScaler").
		WithValues("Serving", fmt.Sprintf("%s/%s", s.Namespace, s.Name))

	// When no Triggers are configured, it means that no scaler needs to be created for the function.
	if s.Spec.ScaleOptions == nil || s.Spec.ScaleOptions.Keda == nil {
		log.Info("No keda scaleOptions found, no need to create scaler.")
		return nil
	}

	version := constants.DefaultFunctionVersion
	if s.Spec.Version != nil {
		version = *s.Spec.Version
	}
	version = strings.ReplaceAll(version, ".", "")

	var hosts []string
	for _, hostname := range s.Spec.Triggers.Http.Route.Hostnames {
		hosts = append(hosts, string(hostname))
	}

	var pathPrefix []string
	for _, rule := range s.Spec.Triggers.Http.Route.Rules {
		for _, match := range rule.Matches {
			if *match.Path.Value == "" {
				pathPrefix = append(pathPrefix, "/")
			} else {
				pathPrefix = append(pathPrefix, *match.Path.Value)
			}
		}
	}

	keda := s.Spec.ScaleOptions.Keda

	var targetPendingRequests int32 = 100 // Default to 100
	if keda.HTTPScaledObject.TargetPendingRequests != nil {
		targetPendingRequests = *keda.HTTPScaledObject.TargetPendingRequests
	}
	var cooldownPeriod int32 = 300 // Default to 300
	if keda.HTTPScaledObject.CooldownPeriod != nil {
		cooldownPeriod = *keda.HTTPScaledObject.CooldownPeriod
	}

	accessor, _ := meta.Accessor(workload)

	httpScaledObject := &httpv1alpha1.HTTPScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-scaler-", s.Name),
			Namespace:    s.Namespace,
			Labels: map[string]string{
				common.OpenfunctionManaged: "true",
				common.ServingLabel:        s.Name,
			},
		},
		Spec: httpv1alpha1.HTTPScaledObjectSpec{
			Hosts:        hosts,
			PathPrefixes: pathPrefix,
			ScaleTargetRef: httpv1alpha1.ScaleTargetRef{
				Deployment: accessor.GetName(),
				Service:    service.GetName(),
				Port:       80,
			},
			Replicas: &httpv1alpha1.ReplicaStruct{
				Min: s.Spec.ScaleOptions.MinReplicas,
				Max: s.Spec.ScaleOptions.MaxReplicas,
			},
			TargetPendingRequests: &targetPendingRequests,
			CooldownPeriod:        &cooldownPeriod,
		},
	}

	if err := controllerutil.SetControllerReference(s, httpScaledObject, r.scheme); err != nil {
		log.Error(err, "Failed to SetControllerReference")
		return err
	}

	if err := r.Create(r.ctx, httpScaledObject); err != nil {
		log.Error(err, "Failed to create Keda scaler")
		return err
	}

	s.Status.ResourceRef[scalerName] = httpScaledObject.GetName()

	log.V(1).Info("Keda scaler Created", "Scaler", httpScaledObject.GetName())

	return nil
}

func getWorkloadName(s *openfunction.Serving) string {
	if s.Status.ResourceRef == nil {
		return ""
	}

	return s.Status.ResourceRef[workloadName]
}
