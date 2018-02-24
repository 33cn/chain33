#!/usr/bin/env bash
# https://hub.docker.com/r/suyanlong/golang-dev/
# https://github.com/suyanlong/golang-dev
# sudo docker pull suyanlong/golang-dev:latest

sudo docker run -it -p 0.0.0.0:8081:8081 -l linux-chain33-build -v $GOPATH/src/code.aliyun.com/chain33/chain33/build:/root/build -w /root suyanlong/golang-dev:latest
