#!/usr/bin/env bash


OUT_DIR="${1}/paracross"

PARACLI="${OUT_DIR}/chain33-para-cli"
PARANAME=para
SRC_CLI=gitlab.33.cn/chain33/chain33/cmd/cli

go build -v -o "${PARACLI}" -ldflags "-X gitlab.33.cn/chain33/chain33/common/config.ParaName=user.p.${PARANAME}. -X gitlab.33.cn/chain33/chain33/common/config.RPCAddr=http://localhost:8901" "${SRC_CLI}"
# shellcheck disable=SC2086
cp ./build/* "${OUT_DIR}"

