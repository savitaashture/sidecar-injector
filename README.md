# sidecar-injector

## Need to update the license file

## Prerequisites
```
1. Ensure that the kubernetes cluster has at least 1.9 or above.
2. Ensure that MutatingAdmissionWebhook controllers are enabled
3. Ensure that the admissionregistration.k8s.io/v1beta1 API is enabled
```
Verification:
```
kubectl api-versions | grep admissionregistration.k8s.io/v1beta1
```
The output should be:
```
admissionregistration.k8s.io/v1beta1
```

##OR

```
ps -ef | grep kube-apiserver | grep admission-control
```
Output should be:
```
--admission-control=NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota
```

## Build

1. Setup dependency

   The repo uses [gvt](https://github.com/FiloSottile/gvt) as the dependency management tool for its Go codebase. Install `gvt` by the following command:
```
go get -u github.com/FiloSottile/gvt
```

2. Build binary, image and push to docker hub

```
1. setup a GOPATH

2. bash -x build.sh
```

## Deploy

```
bash -x deploy.sh
```

## Verify

1. The sidecar injector webhook should be running
```
[root@mstnode ~]# kubectl get pods -n mutation
NAME                                                          READY     STATUS    RESTARTS   AGE
sidecar-injector-webhook-mesher-deployment-8576646db8-x6f56   1/1       Running   0          20s

[root@mstnode ~]# kubectl get deployment -n mutation
NAME                                         DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
sidecar-injector-webhook-mesher-deployment   1         1         1            1           1m
```

2. Label the mutation namespace with `sidecar-injector=enabled`
```
kubectl label namespace mutation sidecar-injector=enabled
[root@mstnode ~]# kubectl get namespace -L sidecar-injector
NAME          STATUS    AGE       SIDECAR-INJECTOR
default       Active    18h
kube-public   Active    18h
kube-system   Active    18h
mutation      Active    3m        enabled
```

3. Deploy an app in Kubernetes cluster, take `client` app as an example

```
[root@mstnode ~]# cat <<EOF | kubectl create -f -
apiVersion: v1
kind: Pod
metadata:
  name: client
  namespace: mutation
  annotations:
    sidecar-injector-mesher.io/inject: "yes"
  labels:
    app: client
    version: 0.0.1
spec:
  containers:
    - name: client
      image: xiaoliang/client-go
      env:
        - name: TARGET
          value: http://server-mesher/
        - name: http_proxy
          value: http://127.0.0.1:30101/
      ports:
        - containerPort: 9000
EOF
```

4. Verify sidecar container injected
```
[root@mstnode ~]# kubectl get pods -n mutation
NAME                                                          READY     STATUS    RESTARTS   AGE
client                                                        2/2       Running   0          12s
```

## Clean
```
bash -x clean.sh
```
