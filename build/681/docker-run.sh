#!/usr/bin/env bash
# first you must build docker image, you can use make docker command
# docker build . -f Dockerfile-run -t chain33-build:latest

sudo docker run -it -p 8801:8801 -p 8802:8802 -p 6060:6060 -p 50051:50051 -l linux-chain33-run \
    -v "$GOPATH"/src/gitlab.33.cn/chain33/chain33:/go/src/gitlab.33.cn/chain33/chain33 \
    -w /go/src/gitlab.33.cn/chain33/chain33 chain33:latest
