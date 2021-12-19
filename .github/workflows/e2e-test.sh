#! /bin/bash

cat config/samples/function-sample-serving-only.yaml | kubectl apply -f -

NODE_IP=$(kubectl get nodes -o jsonpath={.items[0].status.addresses[0].address})
kubectl patch svc -n kourier-system kourier -p "{\"spec\": {\"type\": \"LoadBalancer\", \"externalIPs\": [\"${NODE_IP}\"]}}"
kubectl patch configmap/config-domain -n knative-serving  --type merge --patch "{\"data\":{\"${NODE_IP}.sslip.io\":\"\"}}"

sleep 10

status=`kubectl get ksvc -o jsonpath={.items}`
  if [ "$status" = "[]" ]; then
    echo "Cannot find function serving, exit..."
    exit 1
  fi
server_url=`kubectl get ksvc -o jsonpath={.items[0].status.url}`
echo "Function now is serving on ${server_url}"
curl ${server_url}
res=$?
if test "$res" != "0"; then
  echo "the curl command failed with: $res"
  exit 1
fi

cat config/samples/function-pubsub-sample-serving-only.yaml | kubectl apply -f -
