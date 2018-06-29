#!/bin/bash
set -e
set -x

#gvt restore

appname="sidecar-injector"

rm -rf $appname

BUILD_PATH=$(cd $(dirname $0);pwd)

echo $BUILD_PATH
cd $BUILD_PATH

CGO_ENABLED=0 GO_EXTLINK_ENABLED=0 go build --ldflags '-s -w -extldflags "-static"' -a -o $appname

cp $appname build/; cd $BUILD_PATH/build

bash -x build_image.sh
