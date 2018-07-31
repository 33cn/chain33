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

returnStr1=""

sedfix=""
if [ "$(uname)" == "Darwin" ]; then
    sedfix=".bak"
fi

function setConfig() {
    echo "=========== 配置chain33.toml ============="
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

function startDockerCompose() {
    echo "=========== 启动docker-compose ============="
    # docker-compose ps
    docker-compose ps
    # remove exsit container
    docker-compose down
    # create and run docker-compose container
    docker-compose up --build -d

    echo "=========== sleep 30s wait for docker started ============="
    sleep 30
    docker-compose ps
}

# 设置好测试Docker组的环境,并启动Docker-compose
function startDockers() {
    setConfig
    startDockerCompose
}

# 将钱包进行解锁的操作
# $1 name
# $2 pswd
function unlockWallet() {
    name=$1
    pswd=$2
    result=$(${name} wallet unlock -p "${pswd}" -t 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        exit 1
    fi
    sleep 1
}

# 保存随机种子到钱包数据库
# $1 name
# $2 pswd
# $3 seed
function saveSeed() {
    name=$1
    pswd=$2
    seed=$3
    result=$(${name} seed save -p "${pswd}" -s "${seed}" | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "save seed to wallet error seed, result: ${result}"
        exit 1
    fi
    sleep 1
}

# 导入账号私钥到钱包
# $1 name
# $2 key
# $3 label
function importKey() {
    name=$1
    key=$2
    label=$3
    result=$(${name} account import_key -k "${key}" -l "${label}" | jq ".label")
    if [ -z "${result}" ]; then
        exit 1
    fi
    sleep 1
}

# 开启或停止钱包的挖矿功能
# $1 name
# $2 flag 0:close 1:open
function setAutomine() {
    name=$1
    flag=$2
    result=$(${name} wallet auto_mine -f "${flag}" | jq ".isok")
    if [ "${result}" = "false" ]; then
        exit 1
    fi
}

# $1 name
# $2 needCount
function waitAllPeerReady() {
    name=$1
    needCount=$2
    peersCount "${name}" "$needCount"
    peerStatus=$?
    if [ $peerStatus -eq 1 ]; then
        echo "====== peers not enough ======"
        exit 1
    fi
}

# $1 name
# $2 needCount
function peersCount() {
    name=$1
    needCount=$2
    retryTime=10
    sleepTime=15

    for ((i = 0; i < retryTime; i++)); do
        peersCount=$($name net peer_info | jq '.[] | length')
        printf '查询节点 %s ,所需节点数 %d ,当前节点数 %s \n' "${name}" "${needCount}" "${peersCount}"
        if [ "${peersCount}" = "$needCount" ]; then
            echo "============= 符合节点数要求 ============="
            return 0
        else
            echo "============= 休眠 ${sleepTime}s 继续查询 ============="
            sleep ${sleepTime}
        fi
    done

    return 1
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

#${1} name
#${2} minHeight
#${3} timeout
#${4} names
function syn_block_timeout() {
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

# $1 name
# $2 fromAdd
# $3 execAdd
# $4 note
# $5 amount
function SendToPrivacyExec() {
    name=$1
    fromAdd=$2
    execAdd=$3
    note=$4
    amount=$5
    #sudo docker exec -it $name ./chain33-cli send coins transfer -k $fromAdd -t $execAdd -n $note -a $amount
    result=$($name send coins transfer -k "${fromAdd}" -t "${execAdd}" -n "${note}" -a "${amount}")
    echo "hash : $result"
    returnStr1=$result
}

# $1 name
# $2 fromAdd
# $3 priAdd
# $4 note
# $5 amount
# $6 expire
function pub2priv() {
    name=$1
    fromAdd=$2
    priAdd=$3
    note=$4
    amount=$5
    expire=$6
    #sudo docker exec -it $name ./chain33-cli privacy pub2priv -f $fromAdd -p $priAdd -a $amount -n $note --expire $expire
    result=$($name privacy pub2priv -f "${fromAdd}" -p "${priAdd}" -a "${amount}" -n "${note}" --expire "${expire}" | jq -r ".hash")
    echo "hash : $result"
    returnStr1=$result
}

# $1 name
# $2 fromAdd
# $3 priAdd
# $4 note
# $5 amount
# $6 mixcount
# $7 expire
function priv2priv() {
    name=$1
    fromAdd=$2
    priAdd=$3
    note=$4
    amount=$5
    mixcount=$6
    expire=$7
    #sudo docker exec -it $name ./chain33-cli privacy priv2priv -f $fromAdd -p $priAdd -a $amount -n $note --expire $expire
    result=$($name privacy priv2priv -f "${fromAdd}" -p "${priAdd}" -a "${amount}" -n "${note}" --expire "${expire}" | jq -r ".hash")
    echo "hash : $result"
    returnStr1=$result
}

# $1 name
# $2 fromAdd
# $3 toAdd
# $4 note
# $5 amount
# $6 mixcount
# $7 expire
function priv2pub() {
    name=$1
    fromAdd=$2
    toAdd=$3
    note=$4
    amount=$5
    mixcount=$6
    expire=$7
    #sudo docker exec -it $name ./chain33-cli privacy priv2pub -f $fromAdd -t $toAdd -a $amount -n $note -m $mixcount --expire $expire
    result=$($name privacy priv2pub -f "${fromAdd}" -t "${toAdd}" -a "${amount}" -n "${note}" -m "${mixcount}" --expire "${expire}" | jq -r ".hash")
    echo "hash : $result"
    returnStr1=$result
}

# $1 name
# $2 fromAdd
function showPrivacyExec() {
    name=$1
    fromAdd=$2
    printf '==========showPrivacyExec name=%s addr=%s==========\n' "${name}" "${fromAdd}"
    result=$($name account balance -e privacy -a "${fromAdd}" | jq -r ".balance")
    printf 'balance %s \n' "${result}"
    returnStr1=$result
}

# $1 name
# $2 fromAdd
function showPrivacyBalance() {
    name=$1
    fromAdd=$2
    echo "$name" privacy showpai -a "${fromAdd}" -d 0
    result=$($name privacy showpai -a "${fromAdd}" -d 0 | jq -r ".AvailableAmount")
    printf 'AvailableAmount %s \n' "${result}"
    returnStr1=$result
}

# $1 name
# $2 fromAdd
function showPrivacyFrozenAmount() {
    name=$1
    fromAdd=$2
    echo "$name" privacy showpai -a "${fromAdd}" -d 0
    result=$($name privacy showpai -a "${fromAdd}" -d 0 | jq -r ".FrozenAmount")
    printf 'AvailableAmount %s \n' "${result}"
    returnStr1=$result
}

# $1 name
# $2 fromAdd
function showPrivacyTotalAmount() {
    name=$1
    fromAdd=$2
    echo "$name" privacy showpai -a "${fromAdd}" -d 0
    result=$($name privacy showpai -a "${fromAdd}" -d 0 | jq -r ".TotalAmount")
    printf 'AvailableAmount %s \n' "${result}"
    returnStr1=$result
}

# $1 name
# $2 keypair
# $3 amount
# $4 expire
# 返回 returnStr1 交易的编码字符串
function createPrivacyPub2PrivTx() {
    name=$1
    keypair=$2
    amount=$3
    expire=$4
    note="public_2_privacy_transaction"
    echo "$name" bty pub2priv -p "${keypair}" -a "${amount}" -n "${note}" --expire "${expire}"
    result=$($name bty pub2priv -p "${keypair}" -a "${amount}" -n "${note}" --expire "${expire}")
    returnStr1=$result
}

# $1 name
# $2 keypair
# $3 amount
# $4 sender
# $5 expire
# 返回 returnStr1 交易的编码字符串
function createPrivacyPriv2PrivTx() {
    name=$1
    keypair=$2
    amount=$3
    sender=$4
    expire=$5
    note="private_2_privacy_transaction"
    echo "$name" bty priv2priv -p "${keypair}" -a "${amount}" -s "${sender}" -n "${note}" --expire "${expire}"
    result=$($name bty priv2priv -p "${keypair}" -a "${amount}" -s "${sender}" -n "${note}" --expire "${expire}")
    returnStr1=$result
}

# 创建一个隐私到公开的交易,返回编码过的交易字符串
# $1 name
# $2 from
# $3 to
# $4 amount
# $5 expire
# 返回 returnStr1 交易的编码字符串
function createPrivacyPriv2PubTx() {
    name=$1
    from=$2
    to=$3
    amount=$4
    expire=$5
    note="private_2_public_transaction"
    echo "$name" bty priv2pub -f "${from}" -o "${to}" -a "${amount}" -n "${note}" --expire "${expire}"
    result=$($name bty priv2pub -f "${from}" -o "${to}" -a "${amount}" -n "${note}" --expire "${expire}")
    returnStr1=$result
}

# 对创建的交易进行签名
# $1 name
# $2 addr
# $3 data
# 返回 returnStr1 交易的编码字符串
function signRawTx() {
    name=$1
    addr=$2
    data=$3
    result=$($name wallet sign -a "${addr}" -d "${data}")
    returnStr1=$result
}

# 将签名后的交易发送到内存池并广播
# $1 name
# $2 data
function sendRawTx() {
    name=$1
    data=$2
    result=$($name wallet send -d "${data}")
    returnStr1=$result
}

# $1 name
function displayWalletStatus() {
    name=$1
    ${name} wallet status
}

# $1 name
function listAccount() {
    name=$1
    ${name} account list
}

# $1 name
function listMempoolTxs() {
    name=$1
    ${name} mempool list
}
