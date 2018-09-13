#!/bin/bash
# shellcheck disable=SC1078
# shellcheck disable=SC1079
# shellcheck disable=SC1117
# shellcheck disable=SC2002
# shellcheck disable=SC2003
# shellcheck disable=SC2086
# shellcheck disable=SC2091
# shellcheck disable=SC2116
# shellcheck disable=SC2129
# shellcheck disable=SC2140
# shellcheck disable=SC2162
# shellcheck disable=SC2181

package="chain33_tendermint_config.tar.gz"
log_file=".auto_deploy.log"
config_file="auto_deploy.config"
serverStr="servers"

InitLog() {
    if [ -f ${log_file} ]; then
        rm ${log_file}
    fi

    touch ${log_file}
}

Log() {
    if [ -e ${log_file} ]; then
        $(touch ${log_file})
    fi
    # get current time
    local curtime
    curtime=$(date "+%Y-%m-%d %H:%M:%S")

    echo "[$curtime] $* ..." >>$log_file
}

GetInputFile() {
    echo 'Please input the file: (such as "chain33 chain33-cli genesis.json" ...) '
    read file

    # todo: file detection
    Log "The input file is ${file}"
}

PackageFiles() {
    Log "Begin to package the files: ${file}"
    $(tar zcf ${package} $file)
}

GetUserNamePasswdAndPath() {
    echo "Which way to get environment? 1) Input 2) Config file"
    read choice
    if [ ${choice} -eq 1 ]; then
        echo 'Please input the username, password and path of the destination: (such as "ubuntu 123456 /home/ubuntu/chain33")'

        read destInfo
        username=$(echo ${destInfo} | awk -F ' ' '{print $1}')
        password=$(echo ${destInfo} | awk -F ' ' '{print $2}')
        remote_dir=$(echo ${destInfo} | awk -F ' ' '{print $3}')

        echo 'Please input ip list of your destination: (such as "192.168.3.143 192.168.3.144 192.168.3.145 192.168.3.146")'
        read iplist
        index=0
        CreateNewConfigFile
        for ip in $(echo ${iplist}); do
            index=$(expr $index + 1)
            echo "[servers.${index}]" >>${config_file}
            echo "userName:${username}" >>${config_file}
            echo "password:${password}" >>${config_file}
            echo "hostIp:${ip}" >>${config_file}
            echo "path:${remote_dir}" >>${config_file}
        done

        Log "The dest ip is ${ip} and path is ${remote_dir}"
    elif [ ${choice} -eq 2 ]; then
        ShowConfigInfo

        echo "Does the config of destination right?(yes/no)"
        read input
        if [ "X${input}" = "Xno" ]; then
            echo "The config file is wrong. You can config it manually."
            return 1
        fi
    elif [ ${choice} -eq 3 ]; then
        echo "Wrong input..."
        return 2
    fi

    ShowConfigInfo
}

SendFileAndDecompressFile() {
    getSections

    for line in $sections; do
        if [[ $line =~ $serverStr ]]; then
            index=$(echo $line | awk -F '.' '{print $2}' | awk -F ']' '{print$1}')
            getInfoByIndexAndKey $index "userName"
            username=${info}
            getInfoByIndexAndKey $index "password"
            password=${info}
            getInfoByIndexAndKey $index "hostIp"
            ip=${info}
            getInfoByIndexAndKey $index "path"
            remote_dir=${info}

            ExpectCmd "scp  ${package} ${username}@${ip}:${remote_dir}"
            if [ $? -ne 0 ]; then
                Log "Send file failed, this tool will stoped..."
                return 1
            fi
            ExpectCmd "ssh ${username}@${ip} tar zxf ${remote_dir}/${package} -C ${remote_dir}"
            if [ $? -ne 0 ]; then
                Log "Decompress file failed, this tool will stoped..."
                return 2
            fi
        fi
    done
}

ExpectCmd() {
    cmd=$*
    expect -c "
    spawn ${cmd}
    expect {
        "yes" { send "yes\\r"; exp_continue }
        "password" { send "$password\\r" }
    }
    expect eof"
}

CreateNewConfigFile() {
    if [ -f ${config_file} ]; then
        rm ${config_file}
    fi

    touch ${config_file}
}

ShowConfigInfo() {
    if [ ! -f ${config_file} ]; then
        Log "Config file is not existed."
        return 1
    fi

    getSections

    for line in $sections; do
        if [[ $line =~ $serverStr ]]; then
            index=$(echo $line | awk -F '.' '{print $2}' | awk -F ']' '{print$1}')
            getInfoByIndexAndKey $index "userName"
            echo "servers.$index: userName->$info"
            getInfoByIndexAndKey $index "password"
            echo "servers.$index: password->$info"
            getInfoByIndexAndKey $index "hostIp"
            echo "servers.$index: hostIp->$info"
            getInfoByIndexAndKey $index "path"
            echo "servers.$index: path->$info"
        fi
    done
}

getSections() {
    sections=$(sed -n '/^[# ]*\[.*\][ ]*/p' ${config_file})
}

getInfoByIndex() {
    index=$1
    nextIndex=$(expr ${index} + 1)
    info=$(cat ${config_file} | sed -n "/^[# ]*\[servers.${index}/,/^[# ]*\[servers.${nextIndex}/p")
}

getInfoByIndexAndKey() {
    index=$1
    nextIndex=$(expr ${index} + 1)
    key=$2
    info=$(cat ${config_file} | sed -n "/^[# ]*\[servers.${index}/,/^[# ]*\[servers.${nextIndex}/p" | grep -i $key | awk -F ':' '{print $2}')
}

help() {
    echo "***************************************************************************************************"
    echo "*"
    echo "* This tool can send file to specified path."
    echo "* And you should input the file first(It doesn't support get file auto-matically now)"
    echo "* Then it will pack those file into a package and send to the environment."
    echo "*"
    echo "* Note: You should move the file to the current directory, otherwise the packing process will be failed."
    echo "*"
    echo "***************************************************************************************************"
}

main() {
    # Help for this tool
    help
    # Init log file
    InitLog

    # Input the file want to be translate
    GetInputFile

    # Package the files
    PackageFiles
    if [ $? -ne 0 ]; then
        Log "Pachage file err, this tool will be stoped..."
        exit
    fi

    # Input the IP and path of the destination
    GetUserNamePasswdAndPath
    if [ $? -ne 0 ]; then
        Log "GetUserNamePasswdAndPath err, this tool will be stoped..."
        exit
    fi

    # Send and decompress the package
    SendFileAndDecompressFile
    if [ $? -eq 1 ]; then
        echo "Send file err and exit soon..."
        exit
    elif [ $? -eq 2 ]; then
        echo "Decompress file err and exit soon..."
    fi
}

main
