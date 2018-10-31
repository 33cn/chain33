#!/usr/bin/env bash
# shellcheck disable=SC2178
set +e

PWD=$(cd "$(dirname "$0")" && pwd)
export PATH="$PWD:$PATH"

NODE3="${1}_chain33_1"
CLI="docker exec ${NODE3} /root/chain33-cli"

NODE2="${1}_chain32_1"
CLI2="docker exec ${NODE2} /root/chain33-cli"

NODE1="${1}_chain31_1"
CLI3="docker exec ${NODE1} /root/chain33-cli"

NODE4="${1}_chain30_1"
CLI4="docker exec ${NODE4} /root/chain33-cli"

NODE5="${1}_chain29_1"
CLI5="docker exec ${NODE5} /root/chain33-cli"

NODE6="${1}_chain28_1"
CLI6="docker exec ${NODE6} /root/chain33-cli"

containers=("${NODE1}" "${NODE2}" "${NODE3}" "${NODE4}" "${NODE5}" "${NODE6}")
forkContainers=("${CLI3}" "${CLI2}" "${CLI}" "${CLI4}" "${CLI5}" "${CLI6}")

export COMPOSE_PROJECT_NAME="$1"

sedfix=""
if [ "$(uname)" == "Darwin" ]; then
    sedfix=".bak"
fi

DAPP=""
if [ -n "${2}" ]; then
    DAPP=$2
fi

DAPP_TEST_FILE=""

if [ -n "${DAPP}" ]; then
    testfile="fork-test.sh"
    if [ -e "$testfile" ]; then
        # shellcheck source=/dev/null
        source "${testfile}"
        DAPP_TEST_FILE="$testfile"
    fi

    DAPP_COMPOSE_FILE="docker-compose-${DAPP}.yml"
    if [ -e "$DAPP_COMPOSE_FILE" ]; then
        export COMPOSE_FILE="docker-compose.yml:${DAPP_COMPOSE_FILE}"

    fi

fi

system_coins_file="system/coins/fork-test.sh"
# shellcheck source=/dev/null
source "${system_coins_file}"

echo "=========== # env setting ============="
echo "DAPP=$DAPP"
echo "DAPP_TEST_FILE=$DAPP_TEST_FILE"
echo "COMPOSE_FILE=$COMPOSE_FILE"
echo "COMPOSE_PROJECT_NAME=$COMPOSE_PROJECT_NAME"
echo "CLI=$CLI"

function base_init() {
    # update test environment
    sed -i $sedfix 's/^Title.*/Title="local"/g' chain33.toml
    sed -i $sedfix 's/^TestNet=.*/TestNet=true/g' chain33.toml

    # p2p
    sed -i $sedfix 's/^seeds=.*/seeds=["chain33:13802","chain32:13802","chain31:13802","chain30:13802","chain29:13802","chain28:13802"]/g' chain33.toml
    sed -i $sedfix 's/^enable=.*/enable=true/g' chain33.toml
    sed -i $sedfix 's/^isSeed=.*/isSeed=true/g' chain33.toml
    sed -i $sedfix 's/^innerSeedEnable=.*/innerSeedEnable=false/g' chain33.toml
    sed -i $sedfix 's/^useGithub=.*/useGithub=false/g' chain33.toml

    # rpc
    sed -i $sedfix 's/^jrpcBindAddr=.*/jrpcBindAddr="0.0.0.0:8801"/g' chain33.toml
    sed -i $sedfix 's/^grpcBindAddr=.*/grpcBindAddr="0.0.0.0:8802"/g' chain33.toml
    sed -i $sedfix 's/^whitelist=.*/whitelist=["localhost","127.0.0.1","0.0.0.0"]/g' chain33.toml

    # wallet
    sed -i $sedfix 's/^minerdisable=.*/minerdisable=false/g' chain33.toml

}

function start() {
    # docker-compose ps
    docker-compose ps

    # remove exsit container
    docker-compose down

    # create and run docker-compose container
    #    docker-compose -f docker-compose.yml -f docker-compose-para.yml up --build -d
    docker-compose up --build -d

    local SLEEP=30
    echo "=========== sleep ${SLEEP}s ============="
    sleep ${SLEEP}

    docker-compose ps

    # query node run status

    ${CLI} block last_header
    ${CLI} net info

    ${CLI} net peer_info
    peersCount=$(${CLI} net peer_info | jq '.[] | length')
    echo "${peersCount}"
    if [ "${peersCount}" -lt 2 ]; then
        sleep 20
        peersCount=$(${CLI} net peer_info | jq '.[] | length')
        echo "${peersCount}"
        if [ "${peersCount}" -lt 2 ]; then
            echo "peers error"
            exit 1
        fi
    fi

    #echo "=========== # create seed for wallet ============="
    #seed=$(${CLI} seed generate -l 0 | jq ".seed")
    #if [ -z "${seed}" ]; then
    #    echo "create seed error"
    #    exit 1
    #fi

    echo "=========== # save seed to wallet ============="
    result=$(${CLI} seed save -p 1314 -s "tortoise main civil member grace happy century convince father cage beach hip maid merry rib" | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "save seed to wallet error seed, result: ${result}"
        exit 1
    fi

    sleep 1

    echo "=========== # unlock wallet ============="
    result=$(${CLI} wallet unlock -p 1314 -t 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        exit 1
    fi

    sleep 1

    echo "=========== # import private key returnAddr ============="
    result=$(${CLI} account import_key -k CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944 -l returnAddr | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi

    sleep 1

    echo "=========== # import private key mining ============="
    result=$(${CLI} account import_key -k 4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01 -l minerAddr | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi

    sleep 1
    echo "=========== # close auto mining ============="
    result=$(${CLI} wallet auto_mine -f 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        exit 1
    fi

    ## 2nd mining
    echo "=========== # save seed to wallet ============="
    result=$(${CLI4} seed save -p 1314 -s "tortoise main civil member grace happy century convince father cage beach hip maid merry rib" | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "save seed to wallet error seed, result: ${result}"
        exit 1
    fi

    sleep 1

    echo "=========== # unlock wallet ============="
    result=$(${CLI4} wallet unlock -p 1314 -t 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        exit 1
    fi

    sleep 1

    echo "=========== # import private key returnAddr ============="
    result=$(${CLI4} account import_key -k 2AFF1981291355322C7A6308D46A9C9BA311AA21D94F36B43FC6A6021A1334CF -l returnAddr | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi

    sleep 1

    echo "=========== # import private key mining ============="
    result=$(${CLI4} account import_key -k 2116459C0EC8ED01AA0EEAE35CAC5C96F94473F7816F114873291217303F6989 -l minerAddr | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi

    sleep 1
    echo "=========== # close auto mining ============="
    result=$(${CLI4} wallet auto_mine -f 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        exit 1
    fi

    block_wait "${CLI}" 1

    echo "=========== check genesis hash ========== "
    ${CLI} block hash -t 0
    res=$(${CLI} block hash -t 0 | jq ".hash")
    count=$(echo "$res" | grep -c "0x67c58d6ba9175313f0468ae4e0ddec946549af7748037c2fdd5d54298afd20b6")
    if [ "${count}" != 1 ]; then
        echo "genesis hash error!"
        exit 1
    fi

    echo "=========== query height ========== "
    ${CLI} block last_header
    result=$(${CLI} block last_header | jq ".height")
    if [ "${result}" -lt 1 ]; then
        exit 1
    fi

    ${CLI} wallet status
    ${CLI} account list
    ${CLI} mempool list
}

function dapp_run() {
    if [ -e "$DAPP_TEST_FILE" ]; then
        ${DAPP} "${CLI}" "${1}"
    fi

}

function optDockerfun() {
    #############################################
    #1 第一种分叉构造：首先两条链进行共同挖矿，然后再分
    # 别进行挖矿，即两条链上发生分叉时候的交易是不同的。
    #############################################
    forkType1
    #############################################
    #2 第二种分叉构造：包括第一组docker,第二组docker，
    # 以及公共节点docker，首先共同挖矿，然后停掉第二组
    # docker，备份公共节点docker数据库，在公共节点docker
    # 上创建交易，签名交易，记录签名，发送，然后关掉第一组
    # docker，然后将公共节点docker数据库恢复到备份状态，
    # 然后启动第二组docker,然后发送刚刚记录签名的交易。
    # 最后启动全部节点共同挖矿
    #############################################
    forkType2
}

function forkType1() {
    echo "=========== 开始进行类型1分叉测试 ========== "
    base_init
    dapp_run forkInit

    start
    optDockerPart1
    #############################################
    #此处根据具体需求加入；如从钱包中转入某个具体合约账户
    #1 初始化交易余额
    dapp_run forkConfig

    #############################################
    optDockerPart2
    #############################################
    #此处根据具体需求加入在一条测试链中发送测试数据
    #2 构造第一条链中交易
    dapp_run forkAGroupRun

    #############################################
    optDockerPart3
    #############################################
    #此处根据具体需求加入在第二条测试链中发送测试数据
    #3 构造第二条链中交易
    dapp_run forkBGroupRun

    #############################################
    optDockerPart4
    loopCount=30 #循环次数，每次循环休眠时间100s
    checkBlockHashfun $loopCount

    #############################################
    #此处根据具体需求加入结果检查
    #4 检查交易结果
    dapp_run forkCheckRst

    #############################################
    echo "=========== 类型1分叉测试结束 ========== "
}

function forkType2() {
    echo "=========== 开始进行类型2分叉测试 ========== "
    base_init
    dapp_run fork2Init

    start

    optDockerPart1
    #############################################
    #此处根据具体需求加入；如从钱包中转入某个具体合约账户
    #1 初始化交易余额
    initCoinsAccount
    dapp_run fork2Config

    #############################################
    type2_optDockerPart2
    #############################################
    #此处根据具体需求加入在一条测试链中发送测试数据
    #2 构造第一条链中交易
    genFirstChainCoinstx
    dapp_run fork2AGroupRun
    #############################################
    type2_optDockerPart3
    #############################################
    #此处根据具体需求加入在第二条测试链中发送测试数据
    #3 构造第二条链中交易
    genSecondChainCoinstx
    dapp_run fork2BGroupRun

    #############################################
    type2_optDockerPart4
    loopCount=30 #循环次数，每次循环休眠时间100s
    checkBlockHashfun $loopCount
    #############################################
    #此处根据具体需求加入结果检查
    #4 检查交易结果
    checkCoinsResult
    dapp_run fork2CheckRst
    #############################################
    echo "=========== 类型2分叉测试结束 ========== "
}

function optDockerPart1() {
    echo "====== 区块生成中 ======"
    #sleep 100
    block_wait_timeout "${CLI}" 10 100

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
        if [ $i -ge $((loopCount - 1)) ]; then
            echo "====== peers not enough ======"
            exit 1
        fi
    done

    return 1

}

function optDockerPart2() {
    checkMineHeight
    status=$?
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
    docker pause "${NODE4}" "${NODE5}" "${NODE6}"

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
    if [ $peerStatus -eq 1 ]; then
        echo "====== peers not enough ======"
        exit 1
    fi

}

function optDockerPart3() {
    echo "======第一组docker节点挖矿中======"
    block_wait_timeout "${CLI}" 5 100
    echo "======停止第一组docker节点挖矿======"
    result=$($CLI wallet auto_mine -f 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "stop wallet2 mine fail"
        exit 1
    fi

    echo "====== 第一组内部同步中 ======"
    names[0]="${NODE3}"
    names[1]="${NODE2}"
    names[2]="${NODE1}"
    syn_block_timeout "${CLI}" 3 50 "${names[@]}"

    echo "======================================="
    echo "======== 第二步：第二组docker挖矿 ======="
    echo "======================================="

    echo "======停止第一组docker======"
    docker pause "${NODE1}" "${NODE2}" "${NODE3}"

    echo "======sleep 5s======"
    sleep 5

    echo "======启动第二组docker======"
    docker unpause "${NODE4}" "${NODE5}" "${NODE6}"

    echo "======sleep 20s======"
    sleep 5
    result=$($CLI4 wallet unlock -p 1314 -t 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "wallet1 unlock fail"
        exit 1
    fi

    name="${CLI4}"
    time=60
    needCount=3

    peersCount "${name}" $time $needCount
    peerStatus=$?
    if [ $peerStatus -eq 1 ]; then
        echo "====== peers not enough ======"
        exit 1
    fi

    echo "======开启第二组docker节点挖矿======"
    sleep 1
    result=$($CLI4 wallet auto_mine -f 1 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "start wallet1 mine fail"
        exit 1
    fi

    names[0]="${NODE4}"
    names[1]="${NODE5}"
    names[2]="${NODE6}"
    syn_block_timeout "${CLI4}" 2 100 "${names[@]}"

}

function optDockerPart4() {
    echo "======第二组docker节点挖矿中======"
    block_wait_timeout "${CLI4}" 3 50
    echo "====== 第二组内部同步中 ======"
    names[0]="${NODE4}"
    names[1]="${NODE5}"
    names[2]="${NODE6}"
    syn_block_timeout "${CLI4}" 3 50 "${names[@]}"

    echo "======================================="
    echo "====== 第三步：两组docker共同挖矿 ======="
    echo "======================================="

    echo "======启动第一组docker======"
    docker unpause "${NODE1}" "${NODE2}" "${NODE3}"

    echo "======sleep 20s======"
    sleep 5
    result=$($CLI wallet unlock -p 1314 -t 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "wallet2 unlock fail"
        exit 1
    fi
    echo "======开启第一组docker节点挖矿======"
    sleep 1
    result=$($CLI wallet auto_mine -f 1 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "start wallet2 mine fail"
        exit 1
    fi

    echo "======两组docker节点共同挖矿中======"
    block_wait_timeout "${CLI}" 5 100
}

function copyData() {
    name="${NODE3}"
    sleep 1
    docker exec "${name}" mkdir beifen
    sleep 1
    docker exec "${name}" cp -r datadir beifen
    sleep 1
}

function restoreData() {
    name="${NODE3}"
    sleep 1
    docker exec "${name}" rm -rf datadir
    sleep 1
    docker exec "${name}" cp -r beifen/datadir ./
    sleep 1
}

function type2_optDockerPart2() {
    checkMineHeight
    status=$?
    if [ $status -eq 0 ]; then
        echo "====== All peers is the same height ======"
    else
        echo "====== All peers is the different height, syn blockchain fail======"
        exit 1
    fi

    echo "=============== 备份公共节点数据 =============="
    copyData

    echo "==================================="
    echo "====== 第一步：第一组docker挖矿======"
    echo "==================================="

    echo "======停止第二组docker ======"
    docker pause "${NODE4}" "${NODE5}" "${NODE6}"

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
    if [ $peerStatus -eq 1 ]; then
        echo "====== peers not enough ======"
        exit 1
    fi

}

function type2_optDockerPart3() {
    echo "======第一组docker节点挖矿中======"
    block_wait_timeout "${CLI}" 5 100
    echo "======停止第一组docker节点挖矿======"
    result=$($CLI wallet auto_mine -f 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "stop wallet2 mine fail"
        exit 1
    fi

    echo "====== 第一组内部同步中 ======"
    names[0]="${NODE3}"
    names[1]="${NODE2}"
    names[2]="${NODE1}"
    syn_block_timeout "${CLI}" 3 50 "${names[@]}"

    echo "======================================="
    echo "======== 第二步：第二组docker挖矿 ======="
    echo "======================================="

    echo "======停止第一组中除公共节点的docker======"
    docker pause "${NODE1}" "${NODE2}"

    echo "=============== 恢复公共节点数据 =============="
    restoreData
    docker pause "${NODE3}"

    echo "======sleep 5s======"
    sleep 5

    echo "======启动第二组docker======"
    docker unpause "${NODE3}" "${NODE4}" "${NODE5}" "${NODE6}"

    name="${CLI}"
    time=60
    needCount=4

    peersCount "${name}" $time $needCount
    peerStatus=$?
    if [ $peerStatus -eq 1 ]; then
        echo "====== peers not enough ======"
        exit 1
    fi

    echo "======sleep 20s======"
    sleep 20
    echo "======开启第二组docker节点挖矿======"

    result=$($CLI wallet unlock -p 1314 -t 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "wallet1 unlock fail"
        exit 1
    fi

    sleep 1
    result=$($CLI wallet auto_mine -f 1 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "start wallet1 mine fail"
        exit 1
    fi

    sleep 1
    result=$($CLI4 wallet unlock -p 1314 -t 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "wallet2 unlock fail"
        exit 1
    fi

    sleep 1
    result=$($CLI4 wallet auto_mine -f 1 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "start wallet2 mine fail"
        exit 1
    fi

    names[0]="${NODE3}"
    names[1]="${NODE4}"
    names[2]="${NODE5}"
    names[3]="${NODE6}"
    syn_block_timeout "${CLI}" 2 100 "${names[@]}"

}

function type2_optDockerPart4() {
    echo "======第二组docker节点挖矿中======"
    block_wait_timeout "${CLI}" 3 50
    echo "====== 第二组内部同步中 ======"
    names[0]="${NODE4}"
    names[1]="${NODE5}"
    names[2]="${NODE6}"
    names[3]="${NODE3}"
    syn_block_timeout "${CLI}" 3 50 "${names[@]}"

    echo "======================================="
    echo "====== 第三步：两组docker共同挖矿 ======="
    echo "======================================="

    echo "======启动第一组docker======"
    docker unpause "${NODE1}" "${NODE2}"

    echo "======两组docker节点共同挖矿中======"
    block_wait_timeout "${CLI}" 5 100
}

function checkMineHeight() {
    result=$($CLI4 wallet auto_mine -f 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "stop wallet1 mine fail"
        return 1
    fi
    sleep 1
    result=$($CLI wallet auto_mine -f 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "stop wallet2 mine fail"
        return 1
    fi

    echo "====== stop all wallet mine success ======"

    echo "====== syn blockchain ======"
    syn_block_timeout "${CLI}" 5 50 "${containers[@]}"

    height=0
    height1=$($CLI4 block last_header | jq ".height")
    sleep 1
    height2=$($CLI block last_header | jq ".height")
    if [ "${height2}" -ge "${height1}" ]; then
        height=$height2
        printf '当前最大高度 %s \n' "${height}"
    else
        height=$height1
        printf '当前最大高度 %s \n' "${height}"
    fi

    if [ "${height}" -eq 0 ]; then
        echo "获取当前最大高度失败"
        return 1
    fi
    loopCount=20
    for ((k = 0; k < ${#forkContainers[*]}; k++)); do
        for ((j = 0; j < loopCount; j++)); do
            height1=$(${forkContainers[$k]} block last_header | jq ".height")
            if [ "${height1}" -gt "${height}" ]; then #如果大于说明区块还没有完全产生完，替换期望高度
                height=$height1
                printf '查询 %s 目前区块最高高度为第 %s \n' "${containers[$k]}" "${height}"
            elif [ "${height1}" -eq "${height}" ]; then #找到目标高度
                break
            else
                printf '查询 %s 第 %d 次，当前高度 %d, 需要高度%d, 同步中，sleep 60s 后查询\n' "${containers[$k]}" $j "${height1}" "${height}"
                sleep 60
            fi
            #检查是否超过了最大检测次数
            if [ $j -ge $((loopCount - 1)) ]; then
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
        printf '查询节点 %s ,所需节点数 %d ,当前节点数 %s \n' "${name}" "${needCount}" "${peersCount}"
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
    echo "====== syn blockchain ======"
    syn_block_timeout "${CLI}" 10 50 "${containers[@]}"

    height=0
    hash=""
    height1=$($CLI block last_header | jq ".height")
    sleep 1
    height2=$($CLI4 block last_header | jq ".height")
    if [ "${height2}" -ge "${height1}" ]; then
        height=$height2
        printf "主链为 $CLI 当前最大高度 %d \\n" "${height}"
        sleep 1
        hash=$($CLI block hash -t "${height}" | jq ".hash")
    else
        height=$height1
        printf "主链为 $CLI4 当前最大高度 %d \\n" "${height}"
        sleep 1
        hash=$($CLI4 block hash -t "${height}" | jq ".hash")
    fi

    for ((j = 0; j < $1; j++)); do
        for ((k = 0; k < ${#forkContainers[*]}; k++)); do
            sleep 1
            height0[$k]=$(${forkContainers[$k]} block last_header | jq ".height")
            if [ "${height0[$k]}" -ge "${height}" ]; then
                sleep 1
                hash0[$k]=$(${forkContainers[$k]} block hash -t "${height}" | jq ".hash")
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
        peersCount=0
        peersCount=$(${forkContainers[0]} net peer_info | jq '.[] | length')
        printf '第 %d 次，未查询到网络同步，当前节点数 %d 个，100s后查询\n' $j "${peersCount}"
        sleep 100
        #检查是否超过了最大检测次数
        var=$(($1 - 1))
        if [ $j -ge "${var}" ]; then
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
    result=$($name tx query -s "${txHash}" | jq -r ".receipt.tyname")
    if [ "${result}" = "ExecOk" ]; then
        return 0
    fi
    return 1
}

function block_wait_timeout() {
    if [ "$#" -lt 3 ]; then
        echo "wrong block_wait params"
        exit 1
    fi
    cur_height=$(${1} block last_header | jq ".height")
    expect=$((cur_height + ${2}))
    count=0
    while true; do
        new_height=$(${1} block last_header | jq ".height")
        if [ "${new_height}" -ge "${expect}" ]; then
            break
        fi
        count=$((count + 1))
        sleep 1
        if [ $count -ge "${3}" ]; then
            echo "====== block wait timeout ======"
            break
        fi
    done
    echo "wait new block $count s"
}

function syn_block_timeout() {
    #${1} name
    #${2} minHeight
    #${3} timeout
    #${4} names

    names=${4}

    if [ "$#" -lt 3 ]; then
        echo "wrong block_wait params"
        exit 1
    fi
    cur_height=$(${1} block last_header | jq ".height")
    expect=$((cur_height + ${2}))
    count=0
    while true; do
        new_height=$(${1} block last_header | jq ".height")
        if [ "${new_height}" -lt "${expect}" ]; then
            count=$((count + 1))
            sleep 1
        else
            isSyn="true"
            for ((k = 0; k < ${#names[@]}; k++)); do
                sync_status=$(docker exec "${names[$k]}" /root/chain33-cli net is_sync)
                if [ "${sync_status}" = "false" ]; then
                    isSyn="false"
                    break
                fi
                count=$((count + 1))
                sleep 1
            done
            if [ "${isSyn}" = "true" ]; then
                break
            fi
        fi

        if [ $count -ge $(($3 + 1)) ]; then
            echo "====== syn block wait timeout ======"
            break
        fi

    done
    echo "wait block $count s"
}

function block_wait() {
    if [ "$#" -lt 2 ]; then
        echo "wrong block_wait params"
        exit 1
    fi
    cur_height=$(${1} block last_header | jq ".height")
    expect=$((cur_height + ${2}))
    count=0
    while true; do
        new_height=$(${1} block last_header | jq ".height")
        if [ "${new_height}" -ge "${expect}" ]; then
            break
        fi
        count=$((count + 1))
        sleep 1
    done
    echo "wait new block $count s"
}

optDockerfun
