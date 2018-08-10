#!/bin/bash

set -x
set -e
kubectl create ns chassis

bash -x deploy/signed-cert.sh --service sidecar-injector-webhook-mesher-svc --secret sidecar-injector-webhook-mesher-certs --namespace chassis

kubectl create configmap config --from-file=$HOME/.kube/config -n chassis
kubectl create -f deploy/mesherconfigmap.yaml -n chassis
kubectl create -f deploy/configmap.yaml -n chassis
kubectl create -f deploy/deployment.yaml -n chassis
kubectl create -f deploy/service.yaml -n chassis
kubectl create -f deploy/mutatingwebhook.yaml -n chassis