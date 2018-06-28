FROM ubuntu:16.04 

RUN mkdir -p log
ADD sidecar-injector /sidecar-injector
ENTRYPOINT ["./sidecar-injector"]
