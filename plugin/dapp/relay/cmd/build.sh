#!/usr/bin/env bash

OUT_DIR="${1}/relay"
SRC_RELAYD=gitlab.33.cn/chain33/chain33/plugin/dapp/relay/cmd/relayd
FLAG=$2

# shellcheck disable=SC2086
go build -i ${FLAG} -v -o "${OUT_DIR}/relayd" "${SRC_RELAYD}"
cp ./relayd/relayd.toml "${OUT_DIR}/relayd.toml"
cp ./build/* "${OUT_DIR}"
