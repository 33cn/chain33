#!/bin/sh

package="chain33_tendermint_config.tar.gz"
log_file=".autoDeploy_log"

InitLog()
{
    if [ -f ${log_file} ]; then
        rm ${log_file}
    fi

    touch ${log_file}
}

Log()
{
    if [ -e ${log_file} ]; then
        `touch ${log_file}`
    fi
    # get current time
    local curtime
    curtime=`date "+%Y-%m-%d %H:%M:%S"`

    echo "[$curtime] $* ..." >> $log_file
}

GetInputFile()
{
    echo "Please input the file: (such as \"chain33 chain33-cli genesis.json\" ...) "
    read file

    # todo: file detection
    Log "The input file is ${file}"
}

PackageFiles()
{
    Log "Begin to package the files: ${file}"
    `tar zcf ${package} $file` 
}

GetUserNamePasswdAndPath()
{
    echo "Please input the username, password, ip and path of the destination: (such as \"ubuntu 123456 192.168.3.143 /home/ubuntu/chain33\")"

    read destInfo
    username=`echo ${destInfo} | awk -F ' ' '{print $1}'`
    password=`echo ${destInfo} | awk -F ' ' '{print $2}'`
    ip=`echo ${destInfo} | awk -F ' ' '{print $3}'`
    remote_dir=`echo ${destInfo} | awk -F ' ' '{print $4}'`

    Log "The dest ip is ${ip} and path is ${remote_dir}"
    
}

SendFile()
{
    ExpectCmd "scp  ${package} ${username}@${ip}:${remote_dir}"
}

DecompressFile()
{
    ExpectCmd "ssh ${username}@${ip} tar zxf ${remote_dir}/${package} -C ${remote_dir}"
}

ExpectCmd()
{
    cmd=$*
    expect -c "
    spawn ${cmd}
    expect {
        "yes" { send "yes\\r"; exp_continue }
        "password" { send "$password\\r" }
    }
    expect eof"
}

help()
{
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

main()
{
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
    SendFile
    if [ $? -ne 0 ]; then
        Log "SendFile err, this tool will be stoped..."
        exit
    fi
    
    DecompressFile
    if [ $? -ne 0 ]; then
        Log "DecompressFile err, this tool will be stoped..."
    fi
}



main
