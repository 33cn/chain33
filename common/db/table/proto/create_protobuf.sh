#!/bin/sh
protoc --go_out=plugins=grpc:. ./*.proto --proto_path=.
