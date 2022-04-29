#!/bin/bash

# Copyright 2022 The OpenFunction Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

all=false
with_shipwright=false
with_knative=false
with_openFuncAsync=false
region_cn=false

# add help information
if [[ ${1} = "--help" ]] || [[ ${1} = "-h" ]]
then
  clear
  echo "-----------------------------------------------------------------------------------------------"
  echo "This shell script used to delete OpenFunction components from your Cluster"
  echo "Delete version: v0.26.0"
  echo "--all Will delete cert_manager shipwright knative openFuncAsync all components from your cluster"
  echo "--with-cert-manager Will delete cert-manager component"
  echo "--with-shipwright Will delete shipwright component"
  echo "--with-knative Will delete knative component"
  echo "--with-openFuncAsync Will delete dapr and keda components"
  echo "-p If you can't access the github repo"
  echo "-----------------------------------------------------------------------------------------------"
  exit 0
fi


if [ $? != 0 ]; then
  echo "Terminating..." >&2
  exit 1
fi

while test $# -gt 0; do
  case "$1" in
    --all)
      all=true
      ;;
    --with-shipwright)
      with_shipwright=true
      ;;
    --with-knative)
      with_knative=true
      ;;
    --with-openFuncAsync)
      with_openFuncAsync=true
      ;;
    -p | --region-cn)
      region_cn=true
      ;;
    *)
      echo "Internal error!"
      exit 1
      ;;
  esac
  shift
done

if [ "${all}" = "true" ]; then
  with_shipwright=true
  with_knative=true
  with_openFuncAsync=true
  with_ingress=true
fi

if [ "${with_shipwright}" = "true" ]; then
  if [ "${region_cn}" = "false" ]; then
    kubectl delete --filename https://github.com/tektoncd/pipeline/releases/download/v0.28.1/release.yaml
    kubectl delete --filename https://github.com/shipwright-io/build/releases/download/v0.6.0/release.yaml
  else
    kubectl delete --filename https://openfunction.sh1a.qingstor.com/tekton/pipeline/v0.28.1/release.yaml
    kubectl delete --filename https://openfunction.sh1a.qingstor.com/shipwright/v0.6.0/release.yaml
  fi
fi

if [ "${with_knative}" = "true" ]; then
  if [ "${region_cn}" = "false" ]; then
    kubectl delete -f https://github.com/knative/serving/releases/download/v0.26.0/serving-crds.yaml
    kubectl delete -f https://github.com/knative/serving/releases/download/v0.26.0/serving-core.yaml
    kubectl delete -f https://github.com/knative/net-kourier/releases/download/v0.26.0/kourier.yaml
    kubectl delete -f https://github.com/knative/serving/releases/download/v0.26.0/serving-default-domain.yaml
  else
    kubectl delete -f https://openfunction.sh1a.qingstor.com/knative/serving/v0.26.0/serving-crds.yaml
    kubectl delete -f https://openfunction.sh1a.qingstor.com/knative/serving/v0.26.0/serving-core.yaml
    kubectl delete -f https://openfunction.sh1a.qingstor.com/knative/net-kourier/v0.26.0/kourier.yaml
    kubectl delete -f https://openfunction.sh1a.qingstor.com/knative/serving/v0.26.0/serving-default-domain.yaml
  fi
fi

if [ "${with_openFuncAsync}" = "true" ]; then
  if [ "${region_cn}" = "false" ]; then
    # Installs the latest Dapr CLI.
    wget -q https://raw.githubusercontent.com/dapr/cli/master/install/install.sh -O - | /bin/bash -s 1.4.0
    # Init dapr
    dapr uninstall -k --all
    kubectl delete ns dapr-system
    # Installs the latest release version
    kubectl delete -f https://github.com/kedacore/keda/releases/download/v2.4.0/keda-2.4.0.yaml
  else
    # Installs the latest Dapr CLI.
    wget -q https://openfunction.sh1a.qingstor.com/dapr/install.sh -O - | /bin/bash -s 1.4.0
    # Init dapr
    dapr uninstall -k --all
    kubectl delete ns dapr-system
    # Installs the latest release version
    kubectl delete -f https://openfunction.sh1a.qingstor.com/v2.4.0/keda-2.4.0.yaml
  fi
fi

if [ "${with_ingress}" = "true" ]; then
  if [ "${region_cn}" = "false" ]; then
    kubectl delete -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/cloud/deploy.yaml
  else
    kubectl delete -f https://openfunction.sh1a.qingstor.com/ingress-nginx/deploy/static/provider/cloud/deploy.yaml
  fi
fi
