#!/usr/bin/env bash
# https://hub.docker.com/r/suyanlong/golang-dev/
# https://github.com/suyanlong/golang-dev
# sudo docker pull suyanlong/golang-dev:latest

sudo docker run -it -p 8801:8801 -p 8802:8802 -p 6060:6060 -p 50051:50051 -l linux-chain33-build \
    -v "$GOPATH"/src/gitlab.33.cn/chain33/chain33:/go/src/gitlab.33.cn/chain33/chain33 \
    -w /go/src/gitlab.33.cn/chain33/chain33 suyanlong/golang-dev:latest
