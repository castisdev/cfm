#!/bin/bash -e
source ./scripts/version.sh

#${AppName}은 version.sh 에서 구함
#${GoVer}은 version.sh 에서 구한 것을 override함
GoVer=go1.9.2

sudo docker run --rm -v $(pwd):$(pwd) -w $(pwd) -e GOPATH=$GOPATH golang:1.9.2 ./scripts/detail/build_${GoVer}.sh
sudo chown -R $(whoami):$(whoami) build_${GoVer}
