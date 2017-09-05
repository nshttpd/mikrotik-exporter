#!/bin/bash
set -e
set -x

DIR=`pwd`
NAME=`basename ${DIR}`
SHA=`git rev-parse --short HEAD`
VERSION=${VERSION:-$SHA}

GOOS=linux GOARCH=arm go build .

docker build -t nshttpd/${NAME}:${VERSION}-armhf -f Dockerfile.armhf .
docker push nshttpd/${NAME}:${VERSION}-armhf

rm mikrotik-exporter