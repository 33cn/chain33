#!/bin/sh
cd proto
protoc --go_out=plugins=grpc:../types ./*.proto
