# sidecar-injector  

To Inject sidecar below are the annotation added to client and server yaml

## Required

```
sidecar.mesher.io/inject:

example:

sidecar.mesher.io/inject: "yes"

Allowed values are

"yes" or "y"
```

## Optional

```
sidecar.mesher.io/discoveryType:

eample:

sidecar.mesher.io/discoveryType: "sc"

The allowed values are
1. sc
2. pilot
```

## Annotation required only for server(provider)

```
sidecar.mesher.io/servicePorts:

example:

sidecar.mesher.io/servicePorts: rest:9999

Where

9999 ----> Port where server(provider) is running
rest ----> This is the protocol(it can be rest or grpc)
```
