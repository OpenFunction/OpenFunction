#! /bin/bash

echo "---Status of all related pods---"
kubectl get po -n cert-manager
kubectl get po -n dapr-system
kubectl get po -n ingress-nginx
kubectl get po -n keda
kubectl get po -n knative-serving
kubectl get po -n kourier-system
kubectl get po -n openfunction
kubectl get po -n shipwright-build
kubectl get po -n tekton-pipelines

echo "---Functions---"
kubectl get fn -oyaml

echo "---Builder---"
kubectl get builder -oyaml
kubectl get build -oyaml
kubectl get buildrun -oyaml

echo "---Serving---"
kubectl get serving -oyaml

echo "---OpenFunction controller pod logs---"
kubectl logs -n openfunction "$(kubectl get pod -n openfunction -o jsonpath='{.items[0].metadata.name}')" openfunction

echo "---OpenFunction controller pod status---"
kubectl describe po -n openfunction "$(kubectl get pod -n openfunction -o jsonpath='{.items[0].metadata.name}')"

echo "---Shipwright controller pod logs---"
kubectl logs -n shipwright-build "$(kubectl get pod -n shipwright-build -o jsonpath='{.items[0].metadata.name}')"

echo "---Knative controller pod logs---"
kubectl logs -n knative-serving "$(kubectl get pod -n knative-serving -l app=controller -o jsonpath='{.items[0].metadata.name}')" controller

