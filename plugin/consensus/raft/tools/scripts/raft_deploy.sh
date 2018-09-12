#!/usr/bin/env bash
# shellcheck disable=SC2029
########################################################################################################################
##########################chain33自动部署脚本###########################################################################
########################################################################################################################
##############################解析配置文件#######################################################
pemFile=$1
cmd=$(sed -n '/^[# ]*\[.*\][ ]*/p' servers.conf)
fileName="servers.conf"
serverStr="servers."

getSections() {
    sections=$cmd
}

getInfoByIndex() {
    index=$1
    nextIndex=$((index + 1))
    info=$(cat <"$fileName" | sed -n "/^[# ]*\\[servers.${index}/,/^[# ]*\\[servers.${nextIndex}/p")
}

getInfoByIndexAndKey() {
    index=$1
    key=$2
    info=$(cat <"$fileName" | sed -n "/^[# ]*\\[servers.${index}/,/^[# ]*\\[servers.${nextIndex}/p" | grep -i "$key" | awk -F '=' '{print $2}')
}

main() {
    getSections
    for line in $sections; do
        if [[ $line =~ $serverStr ]]; then
            index=$(echo "$line" | awk -F '.' '{print $2}' | awk -F ']' '{print$1}')
            getInfoByIndexAndKey "$index" "userName"
            echo "servers.$index: userName->$info"
            getInfoByIndexAndKey "$index" "hostIp"
            echo "servers.$index: hostIp->$info"
            getInfoByIndexAndKey "$index" "port"
            echo "servers.$index: port->$info"
        fi
    done
}

############################从本地copy文件到远程主机上#####################################################################
scpFileFromLocal() {
    hostIP=$1
    echo "hostIp:$hostIP"
    port=$2
    echo "port:$port"
    userName=$3
    echo "userName:$userName"
    pemFile=$4
    echo "pemFile:$pemFile"
    scpFile=$5
    deployDir=$6
    ssh -i "$pemFile" -p "$port" "$userName"@"$hostIP" "mkdir -p $deployDir"
    echo "scp -i $pemFile -P $port $scpFile $userName@$hostIP:$deployDir"
    scp -i "$pemFile" -P "$port" "$scpFile" "$userName"@"$hostIP":"$deployDir"

}
####################################解压和启动chain33#################################################################
startChain33() {
    hostIP=$1
    port=$2
    userName=$3
    pemFile=$4
    deployDir=$5
    nodeId=$6
    ssh -i "$pemFile" -p "$port" "$userName"@"$hostIP" "cd $deployDir;tar -xvf chain33.tgz;bash raft_conf.sh $nodeId;bash run.sh start"
    echo done!
}
stopChain33() {
    hostIP=$1
    port=$2
    userName=$3
    pemFile=$4
    deployDir=$5
    nodeId=$6
    ssh -i "$pemFile" -p "$port" "$userName"@"$hostIP" "cd $deployDir;bash run.sh stop"
    echo done!
}
clearChain33() {
    hostIP=$1
    port=$2
    userName=$3
    pemFile=$4
    deployDir=$5
    ssh -i "$pemFile" -p "$port" "$userName"@"$hostIP" "cd $deployDir;bash run.sh clear"
    echo done!
}

##########################################批量copy本地文件到多个远程主机上面####################################################################
batchScpFileFromLocal() {
    getSections
    for line in $sections; do
        if [[ $line =~ $serverStr ]]; then
            index=$(echo "$line" | awk -F '.' '{print $2}' | awk -F ']' '{print$1}')
            getInfoByIndexAndKey "$index" "userName"
            echo "servers.$index: userName->$info"
            userName=$info
            getInfoByIndexAndKey "$index" "hostIp"
            echo "servers.$index: hostIp->$info"
            hostIP=$info
            getInfoByIndexAndKey "$index" "port"
            echo "servers.$index: port->$info"
            port=$info
            getInfoByIndexAndKey "$index" "localFilePath"
            echo "servers.$index: localFilePath->$info"
            localFilePath=$info
            getInfoByIndexAndKey "$index" "remoteDir"
            echo "servers.$index: remoteDir->$info"
            remoteDir=$info
            scpFileFromLocal "$hostIP" "$port" "$userName" "$pemFile" "$localFilePath" "$remoteDir"
            echo "the servers.$index:scp file successfully!"
        fi
    done
}
######################################批量执行解压和启动chain33#################################################################################
batchStartChain33() {
    getSections
    for line in $sections; do
        if [[ $line =~ $serverStr ]]; then
            index=$(echo "$line" | awk -F '.' '{print $2}' | awk -F ']' '{print$1}')
            getInfoByIndexAndKey "$index" "userName"
            echo "servers.$index: userName->$info"
            userName=$info
            getInfoByIndexAndKey "$index" "hostIp"
            echo "servers.$index: hostIp->$info"
            hostIP=$info
            getInfoByIndexAndKey "$index" "port"
            echo "servers.$index: port->$info"
            port=$info
            getInfoByIndexAndKey "$index" "localFilePath"
            echo "servers.$index: localFilePath->$info"
            localFilePath=$info
            getInfoByIndexAndKey "$index" "port"
            echo "servers.$index: remoteDir->$info"
            remoteDir=$info
            startChain33 "$hostIP" "$port" "$userName" "$pemFile" "$remoteDir" "$index"
            echo "the servers.$index:start chain33 successfully!"
        fi
    done
}
######################################批量停止chain33服务######################################################################################
batchStopChain33() {
    getSections
    for line in $sections; do
        if [[ $line =~ $serverStr ]]; then
            index=$(echo "$line" | awk -F '.' '{print $2}' | awk -F ']' '{print$1}')
            getInfoByIndexAndKey "$index" "userName"
            echo "servers.$index: userName->$info"
            userName=$info
            getInfoByIndexAndKey "$index" "hostIp"
            echo "servers.$index: hostIp->$info"
            hostIP=$info
            getInfoByIndexAndKey "$index" "port"
            echo "servers.$index: port->$info"
            port=$info
            getInfoByIndexAndKey "$index" "localFilePath"
            echo "servers.$index: localFilePath->$info"
            localFilePath=$info
            getInfoByIndexAndKey "$index" "remoteDir"
            echo "servers.$index: remoteDir->$info"
            remoteDir=$info
            stopChain33 "$hostIP" "$port" "$userName" "$pemFile" "$remoteDir"
            echo "the servers.$index:stop chain33 successfully!"
        fi
    done
}

######################################批量清理chain33拥有数据##################################################################################
batchClearChain33() {
    getSections
    for line in $sections; do
        if [[ $line =~ $serverStr ]]; then
            index=$(echo "$line" | awk -F '.' '{print $2}' | awk -F ']' '{print$1}')
            getInfoByIndexAndKey "$index" "userName"
            echo "servers.$index: userName->$info"
            userName=$info
            getInfoByIndexAndKey "$index" "hostIp"
            echo "servers.$index: hostIp->$info"
            hostIP=$info
            getInfoByIndexAndKey "$index" "port"
            echo "servers.$index: port->$info"
            port=$info
            getInfoByIndexAndKey "$index" "localFilePath"
            echo "servers.$index: localFilePath->$info"
            localFilePath=$info
            getInfoByIndexAndKey "$index" "remoteDir"
            echo "servers.$index: remoteDir->$info"
            remoteDir=$info
            clearChain33 "$hostIP" "$port" "$userName" "$pemFile" "$remoteDir"
            echo "the servers.$index:clear chain33 data successfully!"
        fi
    done
}
######################################本脚本使用指导##################################################################################
#Program:
# This is a chain33 deploy scripts!
if [ "$2" == "start" ]; then
    batchStartChain33
elif [ "$2" == "scp" ]; then
    batchScpFileFromLocal
elif [ "$2" == "stop" ]; then
    batchStopChain33
elif [ "$2" == "clear" ]; then
    batchClearChain33
else
    echo "Usage: ./raft_deploy.sh [pemFile:认证文件] [scp,start,stop,clear]"
fi
