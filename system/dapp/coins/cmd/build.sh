#!/bin/sh

strpwd=$(pwd)
strcmd=${strpwd##*dapp/}
strapp=${strcmd%/cmd*}

OUT_DIR="${1}/system/$strapp"
#FLAG=$2

mkdir -p "${OUT_DIR}"
cp ./build/* "${OUT_DIR}"
