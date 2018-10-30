#!/bin/sh
protoc --go_out=plugins=grpc:../ptypes ./*.proto
