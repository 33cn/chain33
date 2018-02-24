#!/usr/bin/env bash
# first you must build docker image, you can use make docker command

sudo docker run -it -p 0.0.0.0:8081:8081 -l linux-chain33-run -v $GOPATH/src/code.aliyun.com/chain33/chain33/build:/root -w /root  chain33:latest