#!/bin/bash

kubectl create ns mutation

deployment/signed-cert.sh --service sidecar-injector-webhook-mesher-svc --secret sidecar-injector-webhook-mesher-certs --namespace mutation

export CA_BUNDLE=$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\n')

sed 's/${CA_BUNDLE}/'"$CA_BUNDLE"'/g' deployment/mutatingwebhook.yaml > deployment/webhook_cabundle.yaml

kubectl create -f deployment/mesherconfigmap.yaml -n mutation
kubectl create -f deployment/configmap.yaml -n mutation
kubectl create -f deployment/deployment.yaml -n mutation
kubectl create -f deployment/service.yaml -n mutation
kubectl create -f deployment/webhook_cabundle.yaml -n mutation

kubectl label namespace mutation sidecar-injector=enabled
kubectl get namespace -L sidecar-injector
