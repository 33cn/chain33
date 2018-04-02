#!/usr/bin/env bash
#这是个用于分发部署chain33的脚本
SHELL_FOLDER=$(cd "$(dirname "$0")";pwd)
echo "当前目录是:"$SHELL_FOLDER
cd $SHELL_FOLDER/go-scp/
go build -o go_scp
cp go_scp  servers.toml ../
rm -rf go_scp
cd $SHELL_FOLDER
./go_scp
rm -rf go_scp
rm -rf servers.toml
rm -rf chain33.tgz


#变量定义
#ip_array=("raft15258.chinacloudapp.cn" )
#user="ubuntu"
#remote_cmd="ls /"
#
##本地通过ssh执行远程服务器的脚本
#for ip in ${ip_array[*]}
#do
#    set timeout 30
#    spawn ssh -l $user  $ip
#    expect "password:"
#    send "Fuzamei#123456\r"
#    interact
#    ls /
##    if [ $ip = "192.168.1.1" ]; then
##        port="7777"
##    else
##        port="22"
##    fi
##    ssh -t -p $port $user@$ip "remote_cmd"
#done