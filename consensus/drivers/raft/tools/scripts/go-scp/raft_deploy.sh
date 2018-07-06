#!/usr/bin/env bash
#这是个用于分发部署chain33的脚本
#Program:
# This is a chain33 deploy scripts!
SHELL_FOLDER=$(
    cd "$(dirname "$0")" || exit 1
    pwd
)
echo "curl dir:$SHELL_FOLDER"
if [ "$1" == "start" ]; then
    cd "$SHELL_FOLDER"/go-scp/ || exit 1
    go build -o go_scp
    cp go_scp servers.toml ../
    rm -rf go_scp
    cd "$SHELL_FOLDER" || exit 1
    ./go_scp start all
    #rm -rf go_scp
    #rm -rf servers.toml
    rm -rf chain33.tgz
elif [ "$1" == "stop" ]; then
    ./go_scp stop all
elif [ "$1" == "clear" ]; then
    ./go_scp clear all
else
    echo "Usage: ./raft_deploy.sh [start,stop,clear]"
fi
