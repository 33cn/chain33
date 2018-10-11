#!/bin/sh

protoc --go_out=plugins=grpc:../types ./*.proto --proto_path=. --proto_path="$GOPATH/src/gitlab.33.cn/chain33/chain33/types/proto/"
