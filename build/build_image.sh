#!/bin/bash

docker build -t gochassis/sidecar-injector:latest .
rm -rf sidecar-injector

docker push gochassis/sidecar-injector:latest