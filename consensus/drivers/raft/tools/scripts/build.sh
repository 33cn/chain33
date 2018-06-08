#!/usr/bin/env bash
#这是一个build 构建脚本，用于编译打包chain33
echo "-----start build chain33-----"
SHELL_FOLDER=$(cd "$(dirname "$0")";pwd)
echo "cur dir:"$SHELL_FOLDER
cd $SHELL_FOLDER/../../../../../
go build -o chain33
mv chain33 $SHELL_FOLDER
cp chain33.toml $SHELL_FOLDER
curDir=$(pwd)
echo "cur dir:":$curDir
cd $SHELL_FOLDER
dos2unix *.sh
tar cvf chain33.tgz chain33 chain33.toml raft_conf.sh run.sh
rm -rf chain33
rm -rf chain33.toml
echo "---- chain33 build success!----- "