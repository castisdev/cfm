#!/bin/bash -e
source ./scripts/version.sh

#${AppName}은 version.sh 에서 구함
#${GoVer}은 version.sh 에서 구함

rm -rf build_$GoVer
mkdir -p build_$GoVer

go env
go get
go build -x -o build_$GoVer/${AppName}
GOARCH=386 go build -x -o build_$GoVer/${AppName}-i386
