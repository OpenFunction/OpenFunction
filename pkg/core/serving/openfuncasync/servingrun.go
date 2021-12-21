package openfuncasync

import (
	"context"
	"fmt"
	"strings"

	openfunctioncontext "github.com/OpenFunction/functions-framework-go/openfunction-context"
	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"github.com/go-logr/logr"
	jsoniter "github.com/json-iterator/go"
	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	openfunction "github.com/openfunction/apis/core/v1alpha2"
	"github.com/openfunction/pkg/constants"
	"github.com/openfunction/pkg/core"
	"github.com/openfunction/pkg/util"
)

const (
	servingLabel        = "openfunction.io/serving"
	openfunctionManaged = "openfunction.io/managed"
	runtimeLabel        = "runtime"

	workloadName  = "OpenFuncAsync/workload"
	scalerName    = "OpenFuncAsync/scaler"
	componentName = "OpenFuncAsync/component"

	daprEnabled     = "dapr.io/enabled"
	daprAPPID       = "dapr.io/app-id"
	daprLogAsJSON   = "dapr.io/log-as-json"
	daprAPPProtocol = "dapr.io/app-protocol"
	daprAPPPort     = "dapr.io/app-port"

	FUNCCONTEXT = "FUNC_CONTEXT"
)

type servingRun struct {
	client.Client
	ctx    context.Context
	log    logr.Logger
	scheme *runtime.Scheme
}

func Registry() []client.Object {
	return []client.Object{&appsv1.Deployment{}, &appsv1.StatefulSet{}, &batchv1.Job{},
		&kedav1alpha1.ScaledObject{}, &kedav1alpha1.ScaledJob{},
		&componentsv1alpha1.Component{}}
}

func NewServingRun(ctx context.Context, c client.Client, scheme *runtime.Scheme, log logr.Logger) core.ServingRun {
	return &servingRun{
		c,
		ctx,
		log.WithName("OpenFuncAsync"),
		scheme,
	}
}

func (r *servingRun) Run(s *openfunction.Serving) error {

	log := r.log.WithName("Run").
		WithValues("Serving", fmt.Sprintf("%s/%s", s.Namespace, s.Name))

	if s.Spec.OpenFuncAsync == nil {
		return fmt.Errorf("OpenFuncAsync config must not be nil when using OpenFuncAsync runtime")
	}

	if err := r.Clean(s); err != nil {
		log.Error(err, "Failed to Clean")
		return err
	}

	if err := r.checkComponentSpecExist(s); err != nil {
		log.Error(err, "Some Components does not exist")
		return err
	}

	s.Status.ResourceRef = make(map[string]string)
	if err := r.createComponents(s); err != nil {
		log.Error(err, "Failed to create Dapr Components")
		return err
	}

	workload := r.generateWorkload(s)
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

	return nil
}

func (r *servingRun) Clean(s *openfunction.Serving) error {
	log := r.log.WithName("Clean").
		WithValues("Serving", fmt.Sprintf("%s/%s", s.Namespace, s.Name))

	list := func(lists []client.ObjectList) error {
		for _, list := range lists {
			if err := r.List(r.ctx, list, client.InNamespace(s.Namespace), client.MatchingLabels{servingLabel: s.Name}); err != nil {
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

	return nil
}

func (r *servingRun) Result(s *openfunction.Serving) (string, error) {

	// Currently, it only supports updating the status of serving through the status of deployment.
	if s.Spec.OpenFuncAsync == nil || s.Spec.OpenFuncAsync.Keda == nil ||
		s.Spec.OpenFuncAsync.Keda.ScaledObject == nil ||
		s.Spec.OpenFuncAsync.Keda.ScaledObject.WorkloadType == "StatefulSet" {
		return openfunction.Running, nil
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

	return openfunction.Running, nil
}

func (r *servingRun) generateWorkload(s *openfunction.Serving) client.Object {

	version := constants.DefaultFunctionVersion
	if s.Spec.Version != nil {
		version = *s.Spec.Version
	}

	labels := map[string]string{
		openfunctionManaged:          "true",
		servingLabel:                 s.Name,
		runtimeLabel:                 string(openfunction.OpenFuncAsync),
		constants.CommonLabelVersion: version,
	}
	labels = util.AppendLabels(s.Spec.Labels, labels)

	selector := &metav1.LabelSelector{
		MatchLabels: labels,
	}

	var replicas int32 = 1
	if s.Spec.OpenFuncAsync.Keda != nil &&
		s.Spec.OpenFuncAsync.Keda.ScaledObject != nil &&
		s.Spec.OpenFuncAsync.Keda.ScaledObject.MinReplicaCount != nil {
		replicas = *s.Spec.OpenFuncAsync.Keda.ScaledObject.MinReplicaCount
	}

	var port int32 = 8080
	if s.Spec.Port != nil {
		port = *s.Spec.Port
	}

	annotations := make(map[string]string)
	annotations[daprEnabled] = "true"
	annotations[daprAPPID] = fmt.Sprintf("%s-%s", getFunctionName(s), s.Namespace)
	annotations[daprLogAsJSON] = "true"
	if s.Spec.OpenFuncAsync.Dapr != nil {
		for k, v := range s.Spec.OpenFuncAsync.Dapr.Annotations {
			annotations[k] = v
		}
	}

	// The dapr protocol must equal to the protocol of function framework.
	annotations[daprAPPProtocol] = "grpc"
	// The dapr port must equal the function port.
	annotations[daprAPPPort] = fmt.Sprintf("%d", port)

	annotations = util.AppendLabels(s.Spec.Annotations, annotations)

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
		Name:  FUNCCONTEXT,
		Value: createFunctionContext(s),
	})

	if s.Spec.Params != nil {
		for k, v := range s.Spec.Params {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  k,
				Value: v,
			})
		}
	}

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

	job.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
	if s.Spec.OpenFuncAsync.Keda != nil &&
		s.Spec.OpenFuncAsync.Keda.ScaledJob != nil &&
		s.Spec.OpenFuncAsync.Keda.ScaledJob.RestartPolicy != nil {
		job.Spec.Template.Spec.RestartPolicy = *s.Spec.OpenFuncAsync.Keda.ScaledJob.RestartPolicy
	}

	keda := s.Spec.OpenFuncAsync.Keda
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

func (r *servingRun) createScaler(s *openfunction.Serving, workload runtime.Object) error {
	log := r.log.WithName("CreateKedaScaler").
		WithValues("Serving", fmt.Sprintf("%s/%s", s.Namespace, s.Name))

	keda := s.Spec.OpenFuncAsync.Keda
	if keda == nil || (keda.ScaledJob == nil && keda.ScaledObject == nil) {
		return nil
	}

	var obj client.Object
	if keda.ScaledJob != nil {
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
					openfunctionManaged: "true",
					servingLabel:        s.Name,
					runtimeLabel:        string(openfunction.OpenFuncAsync),
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
				Triggers:                   scaledJob.Triggers,
			},
		}
	} else {
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
					openfunctionManaged: "true",
					servingLabel:        s.Name,
					runtimeLabel:        string(openfunction.OpenFuncAsync),
				},
			},
			Spec: kedav1alpha1.ScaledObjectSpec{
				ScaleTargetRef:  ref,
				PollingInterval: scaledObject.PollingInterval,
				CooldownPeriod:  scaledObject.CooldownPeriod,
				MinReplicaCount: scaledObject.MinReplicaCount,
				MaxReplicaCount: scaledObject.MaxReplicaCount,
				Advanced:        scaledObject.Advanced,
				Triggers:        scaledObject.Triggers,
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

func (r *servingRun) createComponents(s *openfunction.Serving) error {
	log := r.log.WithName("CreateDaprComponents").
		WithValues("Serving", fmt.Sprintf("%s/%s", s.Namespace, s.Name))

	dapr := s.Spec.OpenFuncAsync.Dapr
	if dapr == nil {
		return nil
	}

	value := ""
	for name, dc := range dapr.Components {
		component := &componentsv1alpha1.Component{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-component-%s-", s.Name, name),
				Namespace:    s.Namespace,
				Labels: map[string]string{
					openfunctionManaged: "true",
					servingLabel:        s.Name,
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

func (r *servingRun) checkComponentSpecExist(s *openfunction.Serving) error {

	var cs []string
	if s.Spec.OpenFuncAsync != nil && s.Spec.OpenFuncAsync.Dapr != nil {
		dapr := s.Spec.OpenFuncAsync.Dapr

		if dapr.Inputs != nil && len(dapr.Inputs) > 0 {
			for _, i := range dapr.Inputs {
				if _, ok := dapr.Components[i.Component]; !ok {
					cs = append(cs, i.Component)
				}
			}
		}

		if dapr.Outputs != nil && len(dapr.Outputs) > 0 {
			for _, o := range dapr.Outputs {
				if _, ok := dapr.Components[o.Component]; !ok {
					cs = append(cs, o.Component)
				}
			}
		}
	}

	if cs != nil && len(cs) > 0 {
		return fmt.Errorf("component %s does not exist", strings.Join(cs, ","))
	}
	return nil
}

func createFunctionContext(s *openfunction.Serving) string {

	rt := openfunctioncontext.Knative
	if s.Spec.Runtime != nil {
		rt = openfunctioncontext.Runtime(*s.Spec.Runtime)
	}

	var port int32 = 8080
	if s.Spec.Port != nil {
		port = *s.Spec.Port
	}

	version := ""
	if s.Spec.Version != nil {
		version = *s.Spec.Version
	}

	fc := openfunctioncontext.OpenFunctionContext{
		Name:    getFunctionName(s),
		Version: version,
		Runtime: rt,
		Port:    fmt.Sprintf("%d", port),
	}

	if s.Spec.OpenFuncAsync != nil && s.Spec.OpenFuncAsync.Dapr != nil {
		dapr := s.Spec.OpenFuncAsync.Dapr

		if dapr.Inputs != nil && len(dapr.Inputs) > 0 {
			fc.Inputs = make(map[string]*openfunctioncontext.Input)

			for _, i := range dapr.Inputs {
				c, _ := dapr.Components[i.Component]
				componentType := strings.Split(c.Type, ".")[0]
				uri := i.Topic
				if componentType == string(openfunctioncontext.OpenFuncBinding) {
					uri = i.Component
				}
				input := openfunctioncontext.Input{
					Uri:       uri,
					Component: getComponentName(s, i.Component),
					Type:      openfunctioncontext.ResourceType(componentType),
					Metadata:  i.Params,
				}
				fc.Inputs[i.Name] = &input
			}
		}

		if dapr.Outputs != nil && len(dapr.Outputs) > 0 {
			fc.Outputs = make(map[string]*openfunctioncontext.Output)

			for _, o := range dapr.Outputs {
				c, _ := dapr.Components[o.Component]
				componentType := strings.Split(c.Type, ".")[0]
				uri := o.Topic
				if componentType == string(openfunctioncontext.OpenFuncBinding) {
					uri = o.Component
				}
				output := openfunctioncontext.Output{
					Uri:       uri,
					Component: getComponentName(s, o.Component),
					Type:      openfunctioncontext.ResourceType(componentType),
					Metadata:  o.Params,
					Operation: o.Operation,
				}
				fc.Outputs[o.Name] = &output
			}
		}
	}

	bs, _ := jsoniter.Marshal(fc)
	return string(bs)
}

func getFunctionName(s *openfunction.Serving) string {

	return s.Labels[constants.FunctionLabel]
}

func getComponentName(s *openfunction.Serving, name string) string {

	names := strings.Split(s.Status.ResourceRef[componentName], ",")
	for _, n := range names {
		tmp := strings.TrimPrefix(n, fmt.Sprintf("%s-component-", s.Name))
		if index := strings.LastIndex(tmp, "-"); index != -1 {
			if tmp[:index] == name {
				return n
			}
		}
	}

	return name
}

func getWorkloadName(s *openfunction.Serving) string {
	if s.Status.ResourceRef == nil {
		return ""
	}

	return s.Status.ResourceRef[workloadName]
}
