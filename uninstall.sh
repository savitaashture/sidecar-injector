#!/bin/bash

kubectl delete deployment sidecar-injector-webhook-mesher-deployment -n chassis
kubectl delete svc sidecar-injector-webhook-mesher-svc -n chassis
kubectl delete configmap mesher-configmap sidecar-injector-webhook-mesher-configmap -n chassis
kubectl delete MutatingWebhookConfiguration sidecar-injector-webhook-mesher-cfg
kubectl delete secrets sidecar-injector-webhook-mesher-certs -n chassis

kubectl delete ns chassis