#!/bin/sh

package="chain33_tendermint_config.tar.gz"
log_file="auto_deploy.log"

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
    echo "Please input the file "
    read file
    Log "The input file is ${file}"
}

PackageFiles()
{
    Log "Begin to package the files: ${file}"
    `tar zcf ${package} $file` 
}

GetIPAndPath()
{
    echo "Please input the username, password, ip and path of the destination:"

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

main()
{
    # Init log file
    InitLog

    # Input the file want to be translate
    GetInputFile

    # Package the files
    PackageFiles

    # Input the IP and path of the destination
    GetIPAndPath

    # Send and decompress the package
    SendFile
    DecompressFile
}



main
