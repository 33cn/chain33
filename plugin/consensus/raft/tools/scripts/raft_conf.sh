#!/usr/bin/env bash
#这是一个修改配置文件的脚本
nodeId=$1
function echo_green() {
    echo -e "\\033[32m$1\\033[0m"
}
function main() {
    sed -i "s/singleMode=true/singleMode=true/g" chain33.toml
    sed -i "s/nodeId=1/nodeId=$nodeId/g" chain33.toml
}

main
echo_green "修改完成"
