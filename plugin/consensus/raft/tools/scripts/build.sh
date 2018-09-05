#!/usr/bin/env bash
#这是一个build 构建脚本，用于编译打包chain33
echo "-----start build chain33-----"
SHELL_FOLDER=$(
    cd "$(dirname "$0")" || exit 1
    pwd
)
echo "cur dir:$SHELL_FOLDER"
cd "$SHELL_FOLDER"/../../../../../cmd/chain33/ || exit 1
echo "---go build -o chain33---"
go build -o chain33
mv chain33 "$SHELL_FOLDER"
curDir=$(pwd)
echo "cur dir:$curDir"
cd "$SHELL_FOLDER" || exit 1
#dos2unix *.sh
tar cvf chain33.tgz chain33 chain33.toml raft_conf.sh run.sh
rm -rf chain33
echo "---- chain33 build success!----- "
