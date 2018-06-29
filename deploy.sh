#!/bin/bash

kubectl create ns mutation

deploy/signed-cert.sh --service sidecar-injector-webhook-mesher-svc --secret sidecar-injector-webhook-mesher-certs --namespace mutation

export CA_BUNDLE=$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\n')

sed 's/${CA_BUNDLE}/'"$CA_BUNDLE"'/g' deploy/mutatingwebhook.yaml > deploy/webhook_cabundle.yaml

kubectl create -f deploy/mesherconfigmap.yaml -n mutation
kubectl create -f deploy/configmap.yaml -n mutation
kubectl create -f deploy/deployment.yaml -n mutation
kubectl create -f deploy/service.yaml -n mutation
kubectl create -f deploy/webhook_cabundle.yaml -n mutation

