#!/bin/bash

TEMP=$(getopt -o a --long all,with-tekton,with-knative,with-openFuncAsync -- "$@")

all=false
with_tekton=false
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
  --with-tekton)
    with_tekton=true
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
  with_tekton=true
  with_knative=true
  with_openFuncAsync=true
fi

if [ "$with_tekton" = "true" ]; then
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

  kubectl delete ns knative-eventing
  kubectl delete clusterrolebinding.rbac.authorization.k8s.io eventing-controller
  kubectl delete clusterrolebinding.rbac.authorization.k8s.io eventing-controller-resolver
  kubectl delete clusterrolebinding.rbac.authorization.k8s.io eventing-controller-source-observer
  kubectl delete clusterrolebinding.rbac.authorization.k8s.io eventing-controller-sources-controller
  kubectl delete clusterrolebinding.rbac.authorization.k8s.io eventing-controller-manipulator
  kubectl delete clusterrolebinding.rbac.authorization.k8s.io knative-eventing-pingsource-mt-adapter
  kubectl delete clusterrolebinding.rbac.authorization.k8s.io eventing-webhook
  kubectl delete clusterrolebinding.rbac.authorization.k8s.io eventing-webhook-resolver
  kubectl delete clusterrolebinding.rbac.authorization.k8s.io eventing-webhook-podspecable-binding
  kubectl delete clusterrole.rbac.authorization.k8s.io service-addressable-resolver
  kubectl delete clusterrole.rbac.authorization.k8s.io serving-addressable-resolver
  kubectl delete clusterrole.rbac.authorization.k8s.io channel-addressable-resolver
  kubectl delete clusterrole.rbac.authorization.k8s.io broker-addressable-resolver
  kubectl delete clusterrole.rbac.authorization.k8s.io messaging-addressable-resolver
  kubectl delete clusterrole.rbac.authorization.k8s.io flows-addressable-resolver
  kubectl delete clusterrole.rbac.authorization.k8s.io eventing-broker-filter
  kubectl delete clusterrole.rbac.authorization.k8s.io eventing-broker-ingress
  kubectl delete clusterrole.rbac.authorization.k8s.io eventing-config-reader
  kubectl delete clusterrole.rbac.authorization.k8s.io channelable-manipulator
  kubectl delete clusterrole.rbac.authorization.k8s.io meta-channelable-manipulator
  kubectl delete clusterrole.rbac.authorization.k8s.io knative-eventing-namespaced-admin
  kubectl delete clusterrole.rbac.authorization.k8s.io knative-messaging-namespaced-admin
  kubectl delete clusterrole.rbac.authorization.k8s.io knative-flows-namespaced-admin
  kubectl delete clusterrole.rbac.authorization.k8s.io knative-sources-namespaced-admin
  kubectl delete clusterrole.rbac.authorization.k8s.io knative-bindings-namespaced-admin
  kubectl delete clusterrole.rbac.authorization.k8s.io knative-eventing-namespaced-edit
  kubectl delete clusterrole.rbac.authorization.k8s.io knative-eventing-namespaced-view
  kubectl delete clusterrole.rbac.authorization.k8s.io knative-eventing-controller
  kubectl delete clusterrole.rbac.authorization.k8s.io knative-eventing-pingsource-mt-adapter
  kubectl delete clusterrole.rbac.authorization.k8s.io builtin-podspecable-binding
  kubectl delete clusterrole.rbac.authorization.k8s.io eventing-sources-source-observer
  kubectl delete clusterrole.rbac.authorization.k8s.io knative-eventing-sources-controller
  kubectl delete clusterrole.rbac.authorization.k8s.io knative-eventing-webhook
  kubectl delete validatingwebhookconfiguration.admissionregistration.k8s.io config.webhook.eventing.knative.dev
  kubectl delete mutatingwebhookconfiguration.admissionregistration.k8s.io webhook.eventing.knative.dev
  kubectl delete validatingwebhookconfiguration.admissionregistration.k8s.io validation.webhook.eventing.knative.dev
  kubectl delete mutatingwebhookconfiguration.admissionregistration.k8s.io sinkbindings.webhook.sources.knative.dev
fi

if [ "$with_openFuncAsync" = "true" ]; then
  dapr uninstall -k
  kubectl delete --ignore-not-found=true -f https://github.com/kedacore/keda/releases/download/v2.2.0/keda-2.2.0.yaml
fi
