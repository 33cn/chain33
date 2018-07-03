#!/usr/bin/env bash

set +e

#引入隐私交易分叉测试
. ./privacy-fork-test.sh

function optDockerfun() {
    init

    optDockerPart1
    #############################################
    #此处根据具体需求加入；如从钱包中转入某个具体合约账户
    #1 初始化隐私交易余额
    #initPriAccount

    #############################################
    optDockerPart2
    #############################################
    #此处根据具体需求加入在一条测试链中发送测试数据
    #2 构造第一条链中隐私交易
    #genFirstChainPritx

    #############################################
    optDockerPart3
    #############################################
    #此处根据具体需求加入在第二条测试链中发送测试数据
    #3 构造第二条链中隐私交易
    #genSecondChainPritx

    #############################################
    optDockerPart4
    loopCount=30 #循环次数，每次循环休眠时间100s
    checkBlockHashfun $loopCount
    #############################################
    #此处根据具体需求加入结果检查
    #4 检查隐私交易结果
    #checkPriResult

    #############################################

}

function optDockerPart1() {
    echo "====== sleep 100s 区块生成中 ======"
    sleep 100

    loopCount=20
    for ((i = 0; i < loopCount; i++)); do
        name="${CLI}"
        time=2
        needCount=6
        peersCount "${name}" $time $needCount
        peerStatus=$?
        if [ $peerStatus -eq 1 ]; then
            name="${CLI4}"
            peersCount "${name}" $time $needCount
            peerStatus=$?
            if [ $peerStatus -eq 0 ]; then
                break
            fi
        else
            break
        fi
        #检查是否超过了最大检测次数
        if [ $i -ge $(expr $loopCount - 1) ]; then
            echo "====== peers not enough ======"
            exit 1
        fi
    done

    return 1

}

function optDockerPart2() {
    checkMineHeight
    status=$?
    echo $status
    if [ $status -eq 0 ]; then
        echo "====== All peers is the same height ======"
    else
        echo "====== All peers is the different height, syn blockchain fail======"
        exit 1
    fi

    echo "==================================="
    echo "====== 第一步：第一组docker挖矿======"
    echo "==================================="

    echo "======停止第二组docker ======"
    sudo docker stop $NODE4
    sleep 3
    sudo docker stop $NODE5
    sleep 3
    sudo docker stop $NODE6

    echo "======开启第一组docker节点挖矿======"
    sleep 3
    result=$($CLI wallet auto_mine -f 1 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "start wallet2 mine fail"
        exit 1
    fi

    name=$CLI
    time=60
    needCount=3

    peersCount "${name}" $time $needCount
    peerStatus=$?
    echo $peerStatus
    if [ $peerStatus -eq 1 ]; then
        echo "====== peers not enough ======"
        exit 1
    fi
}

function optDockerPart3() {
    echo "======停止第一组docker节点挖矿======"
    result=$($CLI wallet auto_mine -f 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "stop wallet2 mine fail"
        exit 1
    fi

    echo "====== sleep 100s 第一组内部同步 ======"
    sleep 100

    echo "======================================="
    echo "======== 第二步：第二组docker挖矿 ======="
    echo "======================================="

    echo "======停止第一组docker======"
    sleep 3
    sudo docker stop $NODE1
    sleep 3
    sudo docker stop $NODE2
    sleep 3
    sudo docker stop $NODE3

    echo "======sleep 20s======"
    sleep 20

    echo "======启动第二组docker======"
    sleep 3
    sudo docker start $NODE4
    sleep 3
    sudo docker start $NODE5
    sleep 3
    sudo docker start $NODE6

    echo "======sleep 20s======"
    sleep 20
    result=$($CLI4 wallet unlock -p 1314 -t 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "wallet1 unlock fail"
        exit 1
    fi

    name=$CLI4
    time=60
    needCount=3

    peersCount "${name}" $time $needCount
    peerStatus=$?
    echo $peerStatus
    if [ $peerStatus -eq 1 ]; then
        echo "====== peers not enough ======"
        exit 1
    fi

    echo "======开启第二组docker节点挖矿======"
    sleep 3
    result=$($CLI4 wallet auto_mine -f 1 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "start wallet1 mine fail"
        exit 1
    fi
}

function optDockerPart4() {
    echo "====== sleep 100s 第二组内部同步 ======"
    sleep 100

    echo "======================================="
    echo "====== 第三步：两组docker共同挖矿 ======="
    echo "======================================="

    echo "======启动第一组docker======"
    sleep 3
    sudo docker start $NODE1
    sleep 3
    sudo docker start $NODE2
    sleep 3
    sudo docker start $NODE3

    echo "======sleep 20s======"
    sleep 20
    result=$($CLI wallet unlock -p 1314 -t 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "wallet2 unlock fail"
        exit 1
    fi
    echo "======开启第一组docker节点挖矿======"
    sleep 3
    result=$($CLI wallet auto_mine -f 1 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "start wallet2 mine fail"
        exit 1
    fi
}

function checkMineHeight() {
    result=$($CLI4 wallet auto_mine -f 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "stop wallet1 mine fail"
        return 1
    fi
    sleep 3
    result=$($CLI wallet auto_mine -f 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "stop wallet2 mine fail"
        return 1
    fi

    echo "====== stop all wallet mine success ======"

    echo "====== sleep 100s syn blockchain ======"
    sleep 100

    height=0
    height1=$($CLI4 block last_header | jq ".height")
    sleep 3
    height2=$($CLI block last_header | jq ".height")
    if [ $height2 -ge $height1 ]; then
        height=$height2
        printf "当前最大高度 %d \n" $height
    else
        height=$height1
        printf "当前最大高度 %d \n" $height
    fi

    if [ $height -eq 0 ]; then
        echo "获取当前最大高度失败"
        return 1
    fi
    loopCount=20
    for ((k = 0; k < ${#forkContainers[*]}; k++)); do
        for ((j = 0; j < loopCount; j++)); do
            height1=$(${forkContainers[$k]} block last_header | jq ".height")
            if [ $height1 -gt $height ]; then #如果大于说明区块还没有完全产生完，替换期望高度
                height=$height1
                printf "查询 %s 目前区块最高高度为第 %d \n" "${containers[$k]}" $height
            elif [ $height1 -eq $height ]; then #找到目标高度
                break
            else
                printf "查询 %s 第 %d 次，当前高度 %d, 需要高度%d, 同步中，sleep 60s 后查询\n" "${containers[$k]}" $j $height1 $height
                sleep 60
            fi
            #检查是否超过了最大检测次数
            if [ $j -ge $(expr $loopCount - 1) ]; then
                echo "====== syn blockchain fail======"
                return 1
            fi
        done
    done

    return 0
}

function peersCount() {
    name=$1
    time=$2
    needCount=$3

    for ((i = 0; i < time; i++)); do
        peersCount=$($name net peer_info | jq '.[] | length')
        printf "查询节点 %s ,所需节点数 %d ,当前节点数 %s\n" "${name}" $needCount ${peersCount}
        if [ "${peersCount}" = "$needCount" ]; then
            echo "============= 符合节点数要求 ============="
            return 0
        else
            echo "============= 休眠 30s 继续查询 ============="
            sleep 30
        fi
    done

    return 1
}

function checkBlockHashfun() {
    echo "====== sleep 100s syn blockchain ======"
    sleep 100

    height=0
    hash=""
    sleep 3
    height1=$($CLI block last_header | jq ".height")
    sleep 3
    height2=$($CLI4 block last_header | jq ".height")
    if [ $height2 -ge $height1 ]; then
        height=$height2
        printf "主链为 $CLI 当前最大高度 %d \n" $height
        sleep 3
        hash=$($CLI block hash -t $height | jq ".hash")
    else
        height=$height1
        printf "主链为 $CLI4 当前最大高度 %d \n" $height
        sleep 3
        hash=$($CLI4 block hash -t $height | jq ".hash")
    fi

    echo $hash

    for ((j = 0; j < $1; j++)); do
        for ((k = 0; k < ${#forkContainers[*]}; k++)); do
            sleep 3
            height0[$k]=$(${forkContainers[$k]} block last_header | jq ".height")
            if [ ${height0[$k]} -ge $height ]; then
                sleep 3
                hash0[$k]=$(${forkContainers[$k]} block hash -t $height | jq ".hash")
            else
                hash0[$k]="${forkContainers[$k]}"
            fi
        done

        if [ "${hash0[0]}" = "${hash}" ] && [ "${hash0[1]}" = "${hash}" ] && [ "${hash0[2]}" = "${hash}" ] && [ "${hash0[3]}" = "${hash}" ] && [ "${hash0[4]}" = "${hash}" ] && [ "${hash0[5]}" = "${hash}" ]; then
            echo "syn blockchain success break"
            break
        else
            if [ "${hash0[1]}" = "${hash0[0]}" ] && [ "${hash0[2]}" = "${hash0[0]}" ] && [ "${hash0[3]}" = "${hash0[0]}" ] && [ "${hash0[4]}" = "${hash0[0]}" ] && [ "${hash0[5]}" = "${hash0[0]}" ]; then
                echo "syn blockchain success break"
                break
            fi
        fi
        printf "第 %d 次，未查询到网络同步，sleep 100s 后查询\n" $j
        sleep 100
        #检查是否超过了最大检测次数
        var=$(expr $1 - 1)
        if [ $j -ge $var ]; then
            echo "====== syn blockchain fail======"
            exit 1
        fi
    done
    echo "====== syn blockchain success======"
}

# $1 name
# $2 txHash
function txQuery() {
    name=$1
    txHash=$2
    result=$($name tx query -s $txHash | jq -r ".receipt.tyname")
    if [ "${result}" = "ExecOk" ]; then
        return 0
    fi
    return 1
}
