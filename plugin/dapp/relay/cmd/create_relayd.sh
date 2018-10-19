#!/bin/sh

OUT_DIR=../../../../build/
SRC_RELAYD=./relayd

go build -race -i -v -o "${OUT_DIR}/relayd" "${SRC_RELAYD}"
cp ${SRC_RELAYD}/relayd.toml "${OUT_DIR}"
