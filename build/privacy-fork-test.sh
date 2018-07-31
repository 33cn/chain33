#!/usr/bin/env bash
# shellcheck disable=SC2178
set +e

# 引入通用的函数
# shellcheck disable=SC1091
source comm-test.sh

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

function initPriAccount() {
    name="${CLI}"
    fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
    execAdd="1FeyE6VDZ4FYgpK1n2okWMDAtPkwBuooQd"
    note="test"
    amount=$priTotalAmount1
    SendToPrivacyExec "${name}" $fromAdd $execAdd $note $amount

    sleep 1

    name="${CLI4}"
    fromAdd="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
    execAdd="1FeyE6VDZ4FYgpK1n2okWMDAtPkwBuooQd"
    note="test"
    amount=$priTotalAmount2
    SendToPrivacyExec "${name}" $fromAdd $execAdd $note $amount

    block_wait_timeout "${CLI}" 3 50

    name="${CLI}"
    fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
    showPrivacyExec "${name}" $fromAdd

    sleep 1

    name="${CLI4}"
    fromAdd="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
    showPrivacyExec "${name}" $fromAdd
}

function displayPrivateTotalAmount() {
    fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
    showPrivacyTotalAmount "${name}" $fromAdd

    fromAdd="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
    showPrivacyTotalAmount "${name}" $fromAdd
}

function genFirstChainPritx() {
    echo "====== 发送公对私交易 ======"
    name=$CLI
    echo "当前链为：${name}"
    for ((i = 0; i < priRepeatTx; i++)); do
        fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
        priAdd="0a9d212b2505aefaa8da370319088bbccfac097b007f52ed71d8133456c8185823c8eac43c5e937953d7b6c8e68b0db1f4f03df4946a29f524875118960a35fb"
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
        fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
        priAdd="fcbb75f2b96b6d41f301f2d1abc853d697818427819f412f8e4b4e12cacc0814d2c3914b27bea9151b8968ed1732bd241c8788a332b295b731aee8d39a060388"
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
        fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
        toAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
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

    fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
    showPrivacyBalance "${name}" $fromAdd

    fromAdd="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
    showPrivacyBalance "${name}" $fromAdd
}

function genSecondChainPritx() {
    echo "====== 发送公对私交易 ======"
    name=$CLI4
    echo "当前链为：${name}"
    for ((i = 0; i < priRepeatTx; i++)); do
        fromAdd="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
        priAdd="069fdcd7a2d7cf30dfc87df6f277ae451a78cae6720a6bb05514a4a43e0622d55c854169cc63b6353234c3e65db75e7b205878b1bd94e9f698c7043b27fa162b"
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
        fromAdd="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
        priAdd="d5672eeafbcdf53c8fc27969a5d9797083bb64fb4848bd391cd9b3919c4a1d3cb8534f12e09de3cc541eaaf45acccacaf808a6804fd10a976804397e9ecaf96f"
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
        fromAdd="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
        toAdd="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
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

    fromAdd="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
    showPrivacyBalance "${name}" $fromAdd

    fromAdd="1KcCVZLSQYRUwE5EXTsAoQs9LuJW6xwfQa"
    showPrivacyBalance "${name}" $fromAdd
}

function checkPriResult() {

    block_wait_timeout "${CLI}" 10 170

    name1=$CLI
    name2=$CLI4

    echo "====================检查第一组docker运行结果================="

    for ((i = 0; i < ${#priTxHashs1[*]}; i++)); do
        txHash=${priTxHashs1[$i]}
        txQuery "${name1}" $txHash
        result=$?
        if [ $result -eq 0 ]; then
            priTxFee1=$((priTxFee1 + 1))
        fi
        sleep 1
    done

    fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
    showPrivacyExec "${name1}" $fromAdd
    value1=$returnStr1

    sleep 1

    fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
    showPrivacyBalance "${name1}" $fromAdd
    value2=$returnStr1

    sleep 1

    fromAdd="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
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
        txQuery "${name2}" $txHash
        result=$?
        if [ $result -eq 0 ]; then
            priTxFee2=$((priTxFee2 + 1))
        fi
        sleep 1
    done

    fromAdd="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
    showPrivacyExec "${name2}" $fromAdd
    value1=$returnStr1

    fromAdd="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
    showPrivacyBalance "${name2}" $fromAdd
    value2=$returnStr1

    sleep 1

    fromAdd="1KcCVZLSQYRUwE5EXTsAoQs9LuJW6xwfQa"
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

# 使用三步发送交易的模式
# $1 name
# $2 fromAddr1  发起者的地址,以及私对公的接收者
# $3 pk1        公对私的接收者
# $4 pk2        私对私的接收者
# $5 group      1表示Docker分组1,2表示Docker分组2
function makeTransactionIn3Step() {
    name=$1
    fromAddr1=$2
    pk1=$3
    pk2=$4
    group=$5
    echo "当前操作的链节点为：${name}, 分组类型为${group}"
    priTxindex=0

    height=$(${name} block last_header | jq ".height")
    amount=10
    expire=$((height + 2))
    printf '公对私交易 高度为:%s 转账金额为:%s \n' "${height}" "${amount}"
    createPrivacyPub2PrivTx "${name}" "$pk1" $amount $expire
    signRawTx "${name}" "$fromAddr1" "$returnStr1"
    sendRawTx "${name}" "$returnStr1"
    echo "$returnStr1"
    block_wait_timeout "${name}" 1 15

    height=$(${name} block last_header | jq ".height")
    printf '发送私对私交易当前高度 %s \n' "${height}"
    amount=4
    expire=$((height + 2))
    printf '私对私交易 高度为:%s 转账金额为:%s \n' "${height}" "${amount}"
    createPrivacyPriv2PrivTx "${name}" "$pk2" $amount "$fromAddr1" $expire
    signRawTx "${name}" "$fromAddr1" "$returnStr1"
    sendRawTx "${name}" "$returnStr1"
    echo "$returnStr1"
    if [ "$group" -eq 1 ]; then
        priTxHashs1[$priTxindex]="$returnStr1"
        priTxindex=$((priTxindex + 1))
    else
        priTxHashs2[$priTxindex]="$returnStr1"
        priTxindex=$((priTxindex + 1))
    fi
    block_wait_timeout "${name}" 1 15

    height=$(${name} block last_header | jq ".height")
    printf '发送私对公交易当前高度 %s \n' "${height}"
    amount=4
    from=$fromAddr1
    to=$fromAddr1
    # 4个区块高度以后过期
    expire=$((height + 2))
    createPrivacyPriv2PubTx "${name}" "$from" "$to" $amount $expire
    signRawTx "${name}" "$from" "$returnStr1"
    sendRawTx "${name}" "$returnStr1"
    echo "$returnStr1"
    if [ $group -eq 1 ]; then
        priTxHashs1[$priTxindex]="$returnStr1"
        priTxindex=$((priTxindex + 1))
    else
        priTxHashs2[$priTxindex]="$returnStr1"
        priTxindex=$((priTxindex + 1))
    fi
    block_wait_timeout "${name}" 1 15
}

function genFirstChainPritxType3() {
    makeTransactionIn3Step "${CLI}" $CLIfromAddr1 $CLIprivKey1 $CLIprivKey2 1

    echo "=============查询当前隐私余额============="
    showPrivacyBalance "${name}" $CLIfromAddr1
    showPrivacyBalance "${name}" $CLIfromAddr2
}

function genSecondChainPritxType3() {
    makeTransactionIn3Step "${CLI4}" $CLI4fromAddr1 $CLI4privKey11 $CLI4privKey12 2

    echo "=============查询当前隐私余额============="
    showPrivacyBalance "${name}" $CLI4fromAddr1
    showPrivacyBalance "${name}" $CLI4fromAddr2
}
