#!/usr/bin/env bash

OUT_DIR=$1

# shellcheck disable=SC2086
cp ./build/Dockerfile-app* "${OUT_DIR}"
cp ./build/*.yml "${OUT_DIR}"
cp ./build/chain33.para.toml "${OUT_DIR}"
cp ./build/entrypoint.sh "${OUT_DIR}"
