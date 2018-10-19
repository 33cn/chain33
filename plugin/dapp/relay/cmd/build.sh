#!/bin/sh

OUT_DIR=$1
SRC_RELAYD=./relayd
FLAG=$2

# shellcheck disable=SC2086
go build -i ${FLAG} -v -o "${OUT_DIR}/relayd" "${SRC_RELAYD}"
cp "${SRC_RELAYD}/relayd.toml" "${OUT_DIR}"
