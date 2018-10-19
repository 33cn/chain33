#!/bin/sh

OUT_DIR=$1
SRC_RELAYD=./relayd
go build -i $2 -v -o "${OUT_DIR}/relayd" "${SRC_RELAYD}"
cp ${SRC_RELAYD}/relayd.toml "${OUT_DIR}"
