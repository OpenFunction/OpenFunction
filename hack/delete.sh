#!/bin/bash

TEMP=$(getopt -o a --long all,with-shipwright,with-knative,with-openFuncAsync -- "$@")

all=false
with_shipwright=false
with_knative=false
with_openFuncAsync=false

if [ $? != 0 ]; then
  echo "Terminating..." >&2
  exit 1
fi

# Note the quotes around `$TEMP': they are essential!
eval set -- "$TEMP"

while true; do
  case "$1" in
  --all)
    all=true
    shift
    ;;
  --with-shipwright)
    with_shipwright=true
    shift
    ;;
  --with-knative)
    with_knative=true
    shift
    ;;
  --with-openFuncAsync)
    with_openFuncAsync=true
    shift
    ;;
  --)
    shift
    break
    ;;
  *)
    echo "Internal error!"
    exit 1
    ;;
  esac
done

if [ "$all" = "true" ]; then
  with_shipwright=true
  with_knative=true
  with_openFuncAsync=true
fi

if [ "$with_shipwright" = "true" ]; then
  kubectl delete ns tekton-pipelines
  kubectl delete podsecuritypolicies tekton-pipelines
  kubectl delete ClusterRole tekton-pipelines-controller-cluster-access
  kubectl delete ClusterRole tekton-pipelines-controller-tenant-access
  kubectl delete ClusterRole tekton-pipelines-webhook-cluster-access
  kubectl delete ClusterRoleBinding tekton-pipelines-controller-cluster-access
  kubectl delete ClusterRoleBinding tekton-pipelines-controller-tenant-access
  kubectl delete ClusterRoleBinding tekton-pipelines-webhook-cluster-access
  kubectl delete crd clustertasks.tekton.dev
  kubectl delete crd conditions.tekton.dev
  kubectl delete crd pipelines.tekton.dev
  kubectl delete crd pipelineruns.tekton.dev
  kubectl delete crd pipelineresources.tekton.dev
  kubectl delete crd runs.tekton.dev
  kubectl delete crd tasks.tekton.dev
  kubectl delete crd taskruns.tekton.dev

  kubectl delete validatingwebhookconfiguration validation.webhook.pipeline.tekton.dev
  kubectl delete mutatingwebhookconfiguration webhook.pipeline.tekton.dev
  kubectl delete ValidatingWebhookConfiguration config.webhook.pipeline.tekton.dev
  kubectl delete ClusterRole tekton-aggregate-edit
  kubectl delete ClusterRole tekton-aggregate-view

  kubectl delete PodSecurityPolicy tekton-triggers
  kubectl delete ClusterRole tekton-triggers-admin
  kubectl delete ClusterRole tekton-triggers-core-interceptors
  kubectl delete ClusterRoleBinding tekton-triggers-controller-admin
  kubectl delete ClusterRoleBinding tekton-triggers-webhook-admin
  kubectl delete ClusterRoleBinding tekton-triggers-core-interceptors

  kubectl delete crd clusterinterceptors.triggers.tekton.dev
  kubectl delete crd clustertriggerbindings.triggers.tekton.dev
  kubectl delete crd eventlisteners.triggers.tekton.dev
  kubectl delete crd triggers.triggers.tekton.dev
  kubectl delete crd triggerbindings.triggers.tekton.dev
  kubectl delete crd triggertemplates.triggers.tekton.dev
  kubectl delete ValidatingWebhookConfiguration validation.webhook.triggers.tekton.dev
  kubectl delete MutatingWebhookConfiguration webhook.triggers.tekton.dev
  kubectl delete ValidatingWebhookConfiguration config.webhook.triggers.tekton.dev
  kubectl delete ClusterRole tekton-triggers-aggregate-edit
  kubectl delete ClusterRole tekton-triggers-aggregate-view

  kubectl delete crd extensions.dashboard.tekton.dev
  kubectl delete ClusterRole tekton-dashboard-backend
  kubectl delete ClusterRole tekton-dashboard-dashboard
  kubectl delete ClusterRole tekton-dashboard-extensions
  kubectl delete ClusterRole tekton-dashboard-pipelines
  kubectl delete ClusterRole tekton-dashboard-tenant
  kubectl delete ClusterRole tekton-dashboard-triggers
  kubectl delete ClusterRoleBinding tekton-dashboard-backend
  kubectl delete ClusterRoleBinding tekton-dashboard-tenant
  kubectl delete ClusterRoleBinding tekton-dashboard-extensions

  kubectl delete Namespace shipwright-build
  kubectl delete ClusterRole shipwright-build-controller
  kubectl delete ClusterRoleBinding shipwright-build-controller
  kubectl delete CustomResourceDefinition BuildRun
  kubectl delete CustomResourceDefinition Build
  kubectl delete CustomResourceDefinition BuildStrategy
  kubectl delete CustomResourceDefinition ClusterBuildStrategy
fi

if [ "$with_knative" = "true" ]; then
  kubectl delete crd images.caching.internal.knative.dev
  kubectl delete crd certificates.networking.internal.knative.dev
  kubectl delete crd configurations.serving.knative.dev
  kubectl delete crd ingresses.networking.internal.knative.dev
  kubectl delete crd metrics.autoscaling.internal.knative.dev
  kubectl delete crd podautoscalers.autoscaling.internal.knative.dev
  kubectl delete crd revisions.serving.knative.dev
  kubectl delete crd routes.serving.knative.dev
  kubectl delete crd serverlessservices.networking.internal.knative.dev
  kubectl delete crd services.serving.knative.dev

  kubectl delete Namespace knative-serving
  kubectl delete ClusterRole knative-serving-addressable-resolver
  kubectl delete ClusterRole knative-serving-namespaced-admin
  kubectl delete ClusterRole knative-serving-namespaced-edit
  kubectl delete ClusterRole knative-serving-namespaced-view
  kubectl delete ClusterRole knative-serving-core
  kubectl delete ClusterRole knative-serving-podspecable-binding
  kubectl delete ClusterRole knative-serving-admin
  kubectl delete ClusterRoleBinding knative-serving-controller-admin
  kubectl delete ValidatingWebhookConfiguration config.webhook.serving.knative.dev
  kubectl delete MutatingWebhookConfiguration webhook.serving.knative.dev
  kubectl delete ValidatingWebhookConfiguration validation.webhook.serving.knative.dev

  kubectl delete Namespace kourier-system
  kubectl delete clusterrole.rbac.authorization.k8s.io 3scale-kourier
  kubectl delete clusterrolebinding.rbac.authorization.k8s.io 3scale-kourier

  kubectl delete crd apiserversources.sources.knative.dev
  kubectl delete crd brokers.eventing.knative.dev
  kubectl delete crd channels.messaging.knative.dev
  kubectl delete crd containersources.sources.knative.dev
  kubectl delete crd eventtypes.eventing.knative.dev
  kubectl delete crd parallels.flows.knative.dev
  kubectl delete crd pingsources.sources.knative.dev
  kubectl delete crd sequences.flows.knative.dev
  kubectl delete crd sinkbindings.sources.knative.dev
  kubectl delete crd subscriptions.messaging.knative.dev
  kubectl delete crd triggers.eventing.knative.dev
fi

if [ "$with_openFuncAsync" = "true" ]; then
  dapr uninstall -k
  kubectl delete --ignore-not-found=true -f https://github.com/kedacore/keda/releases/download/v2.2.0/keda-2.2.0.yaml
fi
