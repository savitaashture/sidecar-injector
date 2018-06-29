#!/bin/bash

kubectl delete deployment sidecar-injector-webhook-mesher-deployment -n mutation
kubectl delete svc sidecar-injector-webhook-mesher-svc -n mutation
kubectl delete configmap mesher-configmap sidecar-injector-webhook-mesher-configmap -n mutation
kubectl delete pod client -n mutation
kubectl delete MutatingWebhookConfiguration sidecar-injector-webhook-mesher-cfg
kubectl delete secrets sidecar-injector-webhook-mesher-certs -n mutation

kubectl delete ns mutation