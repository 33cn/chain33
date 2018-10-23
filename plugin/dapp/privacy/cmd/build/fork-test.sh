#!/usr/bin/env bash
# shellcheck disable=SC2178
set +e

CLIfromAddr1="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
CLIprivKey1="0a9d212b2505aefaa8da370319088bbccfac097b007f52ed71d8133456c8185823c8eac43c5e937953d7b6c8e68b0db1f4f03df4946a29f524875118960a35fb"
CLIfromAddr2="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
CLIprivKey2="fcbb75f2b96b6d41f301f2d1abc853d697818427819f412f8e4b4e12cacc0814d2c3914b27bea9151b8968ed1732bd241c8788a332b295b731aee8d39a060388"
CLI4fromAddr1="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
CLI4privKey11="069fdcd7a2d7cf30dfc87df6f277ae451a78cae6720a6bb05514a4a43e0622d55c854169cc63b6353234c3e65db75e7b205878b1bd94e9f698c7043b27fa162b"
CLI4fromAddr2="1KcCVZLSQYRUwE5EXTsAoQs9LuJW6xwfQa"
CLI4privKey12="d5672eeafbcdf53c8fc27969a5d9797083bb64fb4848bd391cd9b3919c4a1d3cb8534f12e09de3cc541eaaf45acccacaf808a6804fd10a976804397e9ecaf96f"

privacyExecAddr="1FeyE6VDZ4FYgpK1n2okWMDAtPkwBuooQd"

priTxFee1=0
priTxFee2=0
priTxindex=0
priTxHashs1=("")
priTxHashs2=("")
priRepeatTx=1 #重复发送交易次数
priTotalAmount1="300.0000"
priTotalAmount2="300.0000"

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

function resetPrivacyGlobalData() {
    priTxFee1=0
    priTxFee2=0
    priTxindex=0
    priTxHashs1=("")
    priTxHashs2=("")
}

function initPriAccount() {
    name="${CLI}"
    enablePrivacy "${name}"
    sleep 1

    fromAdd=$CLIfromAddr1
    execAdd=$privacyExecAddr
    note="test"
    amount=$priTotalAmount1
    SendToPrivacyExec "${name}" $fromAdd $execAdd $note $amount

    sleep 1

    name="${CLI4}"
    enablePrivacy "${name}"
    sleep 1

    fromAdd=$CLI4fromAddr1
    execAdd=$privacyExecAddr
    note="test"
    amount=$priTotalAmount2
    SendToPrivacyExec "${name}" $fromAdd $execAdd $note $amount

    block_wait_timeout "${CLI}" 3 50

    name="${CLI}"
    fromAdd=$CLIfromAddr1
    showPrivacyExec "${name}" $fromAdd

    sleep 1

    name="${CLI4}"
    fromAdd=$CLI4fromAddr1
    showPrivacyExec "${name}" $fromAdd
}

function displayPrivateTotalAmount() {
    fromAdd=$CLIfromAddr1
    showPrivacyTotalAmount "${name}" $fromAdd

    fromAdd=$CLI4fromAddr1
    showPrivacyTotalAmount "${name}" $fromAdd
}

function genFirstChainPritx() {
    echo "====== 发送公对私交易 ======"
    name=$CLI
    echo "当前链为：${name}"
    for ((i = 0; i < priRepeatTx; i++)); do
        fromAdd=$CLIfromAddr1
        priAdd=$CLIprivKey1
        note="pub2priv_test"
        amount=10
        expire=0
        pub2priv "${name}" $fromAdd $priAdd $note $amount $expire

        sleep 1
        height=$(${name} block last_header | jq ".height")
        printf '发送公对私第 %d 笔交易当前高度 %s \n' $i "${height}"
    done

    block_wait_timeout "${CLI}" 5 80

    priTxindex=0
    echo "====== 发送私对私交易 ======"
    for ((i = 0; i < priRepeatTx; i++)); do
        fromAdd=$CLIfromAddr1
        priAdd=$CLIprivKey2
        note="priv2priv_test"
        amount=3
        mixcount=0
        expire=0
        priv2priv "${name}" $fromAdd $priAdd $note $amount $mixcount $expire
        priTxHashs1[$priTxindex]=$returnStr1
        priTxindex=$((priTxindex + 1))

        sleep 1
        height=$(${name} block last_header | jq ".height")
        printf '发送私对私第 %d 笔交易当前高度 %s \n' $i "${height}"
    done

    block_wait_timeout "${CLI}" 5 80

    echo "====== 发送私对公交易 ======"
    for ((i = 0; i < priRepeatTx; i++)); do
        fromAdd=$CLIfromAddr1
        toAdd=$CLIfromAddr1
        note="priv2pub_test"
        amount=2
        mixcount=0
        expire=0
        priv2pub "${name}" $fromAdd $toAdd $note $amount $mixcount $expire
        priTxHashs1[$priTxindex]=$returnStr1
        priTxindex=$((priTxindex + 1))

        sleep 1
        height=$(${name} block last_header | jq ".height")
        printf '发送私对公第 %d 笔交易当前高度 %s \n' $i "${height}"
    done

    echo "=============查询当前隐私余额============="

    fromAdd=$CLIfromAddr1
    showPrivacyBalance "${name}" $fromAdd

    fromAdd=$CLIfromAddr2
    showPrivacyBalance "${name}" $fromAdd
}

function genSecondChainPritx() {
    echo "====== 发送公对私交易 ======"
    name=$CLI4
    echo "当前链为：${name}"
    for ((i = 0; i < priRepeatTx; i++)); do
        fromAdd=$CLI4fromAddr1
        priAdd=$CLI4privKey11
        note="pub2priv_test"
        amount=10
        expire=0
        pub2priv "${name}" $fromAdd $priAdd $note $amount $expire

        sleep 1
        height=$(${name} block last_header | jq ".height")
        printf '发送公对私第 %d 笔交易当前高度 %s \n' $i "${height}"
    done

    block_wait_timeout "${name}" 5 80

    priTxHashs2=("")
    priTxindex=0
    echo "====== 发送私对私交易 ======"
    for ((i = 0; i < priRepeatTx; i++)); do
        fromAdd=$CLI4fromAddr1
        priAdd=$CLI4privKey12
        note="priv2priv_test"
        amount=2
        mixcount=0
        expire=0
        priv2priv "${name}" $fromAdd $priAdd $note $amount $mixcount $expire
        priTxHashs2[$priTxindex]=$returnStr1
        priTxindex=$((priTxindex + 1))

        sleep 1
        height=$(${name} block last_header | jq ".height")
        printf '发送私对私第 %d 笔交易当前高度 %s \n' $i "${height}"
    done

    block_wait_timeout "${name}" 5 80

    echo "====== 发送私对公交易 ======"
    for ((i = 0; i < priRepeatTx; i++)); do
        fromAdd=$CLI4fromAddr1
        toAdd=$CLI4fromAddr1
        note="priv2pub_test"
        amount=2
        mixcount=0
        expire=0
        priv2pub "${name}" $fromAdd $toAdd $note $amount $mixcount $expire
        priTxHashs2[$priTxindex]=$returnStr1
        priTxindex=$((priTxindex + 1))

        sleep 1
        height=$(${name} block last_header | jq ".height")
        printf '发送私对私第 %d 笔交易当前高度 %s \n' $i "${height}"
    done

    echo "=============查询当前隐私余额============="

    fromAdd=$CLI4fromAddr1
    showPrivacyBalance "${name}" $fromAdd

    fromAdd=$CLI4fromAddr2
    showPrivacyBalance "${name}" $fromAdd
}

function checkPriResult() {

    block_wait_timeout "${CLI}" 10 170

    name1=$CLI
    name2=$CLI4

    echo "====================检查第一组docker运行结果================="

    for ((i = 0; i < ${#priTxHashs1[*]}; i++)); do
        txHash=${priTxHashs1[$i]}
        txQuery "${name1}" "$txHash"
        result=$?
        if [ $result -eq 0 ]; then
            priTxFee1=$((priTxFee1 + 1))
        fi
        sleep 1
    done

    fromAdd=$CLIfromAddr1
    showPrivacyExec "${name1}" $fromAdd
    value1=$returnStr1

    sleep 1

    fromAdd=$CLIfromAddr1
    showPrivacyBalance "${name1}" $fromAdd
    value2=$returnStr1

    sleep 1

    fromAdd=$CLIfromAddr2
    showPrivacyBalance "${name1}" $fromAdd
    value3=$returnStr1

    printf '中间交易费为%d \n' "${priTxFee1}"

    actTotal=$(echo "$value1 + $value2 + $value3 + $priTxFee1" | bc)
    echo "${name1} 实际金额：$actTotal"

    if [ "${actTotal}" = $priTotalAmount1 ]; then
        echo "${name1} 分叉后检查实际金额符合"
    else
        echo "${name1} 分叉后检查实际金额不符合"
    fi

    echo "====================检查第二组docker运行结果================="
    for ((i = 0; i < ${#priTxHashs2[*]}; i++)); do
        txHash=${priTxHashs2[$i]}
        txQuery "${name2}" "$txHash"
        result=$?
        if [ $result -eq 0 ]; then
            priTxFee2=$((priTxFee2 + 1))
        fi
        sleep 1
    done

    fromAdd=$CLI4fromAddr1
    showPrivacyExec "${name2}" $fromAdd
    value1=$returnStr1

    fromAdd=$CLI4fromAddr1
    showPrivacyBalance "${name2}" $fromAdd
    value2=$returnStr1

    sleep 1

    fromAdd=$CLI4fromAddr2
    showPrivacyBalance "${name2}" $fromAdd
    value3=$returnStr1

    printf '中间交易费为%d \n' "${priTxFee2}"

    actTotal=$(echo "$value1 + $value2 + $value3 + $priTxFee2" | bc)
    echo "${name2} 实际金额：$actTotal"

    if [ "${actTotal}" = $priTotalAmount2 ]; then
        echo "${name2} 分叉后检查实际金额符合"
    else
        echo "${name2} 分叉后检查实际金额不符合"
    fi

    sleep 1
}

function genPrivacy2PrivacyTx() {
    name=$1
    fromaddr=$2
    keypair=$3
    note=$4
    amount=$5
    mixcount=6
    expire=$7
    note="priv2priv_test"
    priv2priv "${name}" "$fromaddr" "$keypair" "$note" "$amount" "$mixcount" "$expire"
    if [ "$group" -eq 1 ]; then
        priTxHashs1[$priTxindex]="$returnStr1"
        priTxindex=$((priTxindex + 1))
    else
        priTxHashs2[$priTxindex]="$returnStr1"
        priTxindex=$((priTxindex + 1))
    fi
}

function genPrivacy2PublicTx() {
    name=$1
    from=$2
    to=$3
    note=$4
    amount=$5
    mixcount=$6
    expire=$7
    priv2pub "${name}" "$from" "$to" "$note" "$amount" "$mixcount" "$expire"
    if [ "$group" -eq 1 ]; then
        priTxHashs1[$priTxindex]="$returnStr1"
        priTxindex=$((priTxindex + 1))
    else
        priTxHashs2[$priTxindex]="$returnStr1"
        priTxindex=$((priTxindex + 1))
    fi
}

# 使用三步发送交易的模式
# $1 name
# $2 fromAddr1  发起者的地址,以及私对公的接收者
# $3 pk1        公对私的接收者
# $4 pk2        私对私的接收者
# $5 group      1表示Docker分组1,2表示Docker分组2
function genTransactionInType4() {
    name=$1
    fromAddr1=$2
    pk1=$3
    pk2=$4
    group=$5
    echo "当前操作的链节点为：${name}, 分组类型为${group}"
    expire=120

    height=$(${name} block last_header | jq ".height")
    amount=17
    printf '公对私交易 高度为:%s 转账金额为:%s \n' "${height}" "${amount}"
    pub2priv "${name}" "$fromAddr1" "$pk1" "公对私120秒超时" 12 120
    block_wait_timeout "${name}" 1 16
    pub2priv "${name}" "$fromAddr1" "$pk1" "公对私3600秒超时" 13 3600
    pub2priv "${name}" "$fromAddr1" "$pk1" "公对私1800秒超时" 17 1800
    block_wait_timeout "${name}" 5 80

    height=$(${name} block last_header | jq ".height")
    amount=7
    mixcount=0
    printf '私对私交易 高度为:%s 转账金额为:%s \n' "${height}" "${amount}"
    genPrivacy2PrivacyTx "${name}" "$fromAddr1" "$pk2" "私对私120秒超时" 7 $mixcount 120
    pub2priv "${name}" "$fromAddr1" "$pk1" "公对私120秒超时" 12 120
    pub2priv "${name}" "$fromAddr1" "$pk1" "公对私1800秒超时" 17 1800
    block_wait_timeout "${name}" 1 16
    genPrivacy2PrivacyTx "${name}" "$fromAddr1" "$pk2" "私对私3600秒超时" 9 $mixcount 3600
    genPrivacy2PrivacyTx "${name}" "$fromAddr1" "$pk2" "私对私3600秒超时" 19 $mixcount 3600
    block_wait_timeout "${name}" 1 16
    pub2priv "${name}" "$fromAddr1" "$pk1" "公对私120秒超时" 19 120
    block_wait_timeout "${name}" 5 80

    height=$(${name} block last_header | jq ".height")
    amount=7
    from=$fromAddr1
    to=$fromAddr1
    mixcount=0
    printf '私对公交易 高度为:%s 转账金额为:%s \n' "${height}" "${amount}"
    pub2priv "${name}" "$fromAddr1" "$pk1" "公对私120秒超时" 12 120
    genPrivacy2PublicTx "${name}" "$from" "$to" "私对公120秒超时" $amount $mixcount 120
    genPrivacy2PublicTx "${name}" "$from" "$to" "私对公3600秒超时" $amount $mixcount 3600
    block_wait_timeout "${name}" 1 16
    genPrivacy2PrivacyTx "${name}" "$fromAddr1" "$pk2" "私对私120秒超时" 7 $mixcount 120
    block_wait_timeout "${name}" 5 80
}

function genFirstChainPritxType4() {
    genTransactionInType4 "${CLI}" $CLIfromAddr1 $CLIprivKey1 $CLIprivKey2 1

    echo "=============查询当前隐私余额============="
    showPrivacyBalance "${name}" $CLIfromAddr1
    showPrivacyBalance "${name}" $CLIfromAddr2
}

function genSecondChainPritxType4() {
    genTransactionInType4 "${CLI4}" $CLI4fromAddr1 $CLI4privKey11 $CLI4privKey12 2

    echo "=============查询当前隐私余额============="
    showPrivacyBalance "${name}" $CLI4fromAddr1
    showPrivacyBalance "${name}" $CLI4fromAddr2
}

# $1 name
function enablePrivacy() {
    name=$1
    printf '==========enablePrivacy name=%s ==========\n' "${name}"
    $name privacy enable -a all
}

# $1 name
# $2 txHash
#function txQuery()
#{
#    name=$1
#    txHash=$2
#    echo "txQuery hash: $txHash"
#    result=$($name tx query -s $txHash | jq -r ".receipt.tyname")
#    if [ "${result}" = "ExecOk" ]; then
#        return 0
#    fi
#    return 1
#}

function privacy() {
    if [ "${2}" == "forkInit" ]; then
        resetPrivacyGlobalData
    elif [ "${2}" == "forkConfig" ]; then
        initPriAccount
    elif [ "${2}" == "forkAGroupRun" ]; then
        genFirstChainPritx
        genFirstChainPritxType4
    elif [ "${2}" == "forkBGroupRun" ]; then
        genSecondChainPritx
        genSecondChainPritxType4
    elif [ "${2}" == "forkCheckRst" ]; then
        checkPriResult
    fi

    if [ "${2}" == "fork2Init" ]; then
        resetPrivacyGlobalData
    elif [ "${2}" == "fork2Config" ]; then
        return
    elif [ "${2}" == "fork2AGroupRun" ]; then
        genFirstChainPritxType4
    elif [ "${2}" == "fork2BGroupRun" ]; then
        genSecondChainPritxType4
    elif [ "${2}" == "fork2CheckRst" ]; then
        return
    fi

}
