package knative

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	kservingv1 "knative.dev/serving/pkg/apis/serving/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openfunction/pkg/constants"

	openfunction "github.com/openfunction/apis/core/v1alpha2"
	"github.com/openfunction/pkg/core"
	"github.com/openfunction/pkg/util"
)

const (
	servingLabel = "openfunction.io/serving"

	knativeService = "serving.knative.dev/service"
)

type servingRun struct {
	client.Client
	ctx    context.Context
	log    logr.Logger
	scheme *runtime.Scheme
}

func Registry() []client.Object {
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

func (r *servingRun) Run(s *openfunction.Serving) error {
	log := r.log.WithName("Run").
		WithValues("Serving", fmt.Sprintf("%s/%s", s.Namespace, s.Name))

	if err := r.Clean(s); err != nil {
		log.Error(err, "Clean failed")
		return err
	}

	service := r.createService(s)
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
	if err := r.List(r.ctx, services, client.InNamespace(s.Namespace), client.MatchingLabels{servingLabel: s.Name}); err != nil {
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

func (r *servingRun) createService(s *openfunction.Serving) *kservingv1.Service {

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

	version := constants.DefaultFunctionVersion
	if s.Spec.Version != nil {
		version = *s.Spec.Version
	}
	labels := map[string]string{
		constants.CommonLabelVersion: version,
	}
	labels = util.AppendLabels(s.Spec.Labels, labels)

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
				servingLabel: s.Name,
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

func getName(s *openfunction.Serving, key string) string {
	if s.Status.ResourceRef == nil {
		return ""
	}

	return s.Status.ResourceRef[key]
}
