package knative

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/openfunction/pkg/util"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kservingv1 "knative.dev/serving/pkg/apis/serving/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	openfunction "github.com/openfunction/apis/core/v1alpha2"
	"github.com/openfunction/pkg/core"
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

func NewServingRun(ctx context.Context, c client.Client, scheme *runtime.Scheme, log logr.Logger) core.ServingRun {
	return &servingRun{
		c,
		ctx,
		log.WithName("Knative"),
		scheme,
	}
}

func (r *servingRun) Run(s *openfunction.Serving) error {
	log := r.log.WithName("Run")

	if err := r.clean(s); err != nil {
		log.Error(err, "Clean failed", "name", s.Name, "namespace", s.Namespace)
		return err
	}

	service := r.createService(s)
	service.SetOwnerReferences(nil)
	if err := ctrl.SetControllerReference(s, service, r.scheme); err != nil {
		log.Error(err, "Failed to SetControllerReference", "name", s.Name, "namespace", s.Namespace)
		return err
	}

	if err := r.Create(r.ctx, service); err != nil {
		log.Error(err, "Failed to Create knative service", "name", s.Name, "namespace", s.Namespace)
		return err
	}

	log.V(1).Info("Knative service created", "namespace", service.Namespace, "name", service.Name)

	if s.Status.ResourceRef == nil {
		s.Status.ResourceRef = make(map[string]string)
	}

	s.Status.ResourceRef[knativeService] = service.Name

	return nil
}

func (r *servingRun) clean(s *openfunction.Serving) error {
	log := r.log.WithName("Clean")

	services := &kservingv1.ServiceList{}
	if err := r.List(r.ctx, services, client.InNamespace(s.Namespace), client.MatchingLabels{servingLabel: s.Name}); err != nil {
		return err
	}

	for _, item := range services.Items {
		if strings.HasPrefix(item.Name, s.Name) {
			if err := r.Delete(context.Background(), &item); util.IgnoreNotFound(err) != nil {
				return err
			}
			log.V(1).Info("Delete knative service", "namespace", item.Namespace, "name", item.Name)
		}
	}

	return nil
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

	rand.Seed(time.Now().UnixNano())
	serviceName := fmt.Sprintf("%s-ksvc-%s", s.Name, rand.String(5))
	workloadName := serviceName
	if s.Spec.Version != nil {
		workloadName = fmt.Sprintf("%s-%s", workloadName, strings.ReplaceAll(*s.Spec.Version, ".", ""))
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
				servingLabel: s.Name,
			},
		},
		Spec: kservingv1.ServiceSpec{
			ConfigurationSpec: kservingv1.ConfigurationSpec{
				Template: kservingv1.RevisionTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:      workloadName,
						Namespace: s.Namespace,
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
