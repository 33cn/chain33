#!/usr/bin/env bash

coinsTxFee1=0
coinsTxFee2=0
coinsTxHashs1=("")
coinsTxHashs2=("")
coinsgStr=""
coinsRepeatTx=1 #重复发送交易次数
coinsTotalAmount1="100.0000"
coinsTotalAmount2="100.0000"
defaultTxFee="0.001"
coinsAddr1=""
#coinsAddr2=""
coinsTxSign=("")

function initCoinsAccount() {

    name="${CLI}"
    label="coinstest1"
    createAccount "${name}" $label
    coinsAddr1=$coinsgStr

    sleep 1
    From="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
    to="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
    note="coinstx"
    amount=$coinsTotalAmount1
    sendcoinsTx "${name}" $From $to $note $amount

    sleep 1
    name="${CLI4}"
    label="coinstest2"
    createAccount "${name}" $label
    #coinsAddr2=$coinsgStr

    sleep 1
    From="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
    to="1KcCVZLSQYRUwE5EXTsAoQs9LuJW6xwfQa"
    note="coinstx"
    amount=$coinsTotalAmount2
    sendcoinsTx "${name}" $From $to $note $amount

    #########查询新增账户余额#########
    sleep 60
    name="${CLI}"
    fromAdd="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
    showCoinsBalance "${name}" $fromAdd
    coinsTotalAmount1=$coinsgStr

    sleep 1
    name="${CLI4}"
    fromAdd="1KcCVZLSQYRUwE5EXTsAoQs9LuJW6xwfQa"
    showCoinsBalance "${name}" $fromAdd
    coinsTotalAmount2=$coinsgStr

}

function genFirstChainCoinstx() {
    echo "====== 发送coins交易 ======"
    name=$CLI
    echo "当前链为：${name}"

    for ((i = 0; i < coinsRepeatTx; i++)); do
        From="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
        to=$coinsAddr1
        note="coinstx"
        amount=1
        genCoinsTx "${name}" $From $to $note $amount
        coinsTxSign[$i]="${coinsgStr}"

        block_wait_timeout "${CLI}" 1 20

        #发送交易
        sendCoinsTxHash "${name}" "${coinsgStr}"
        coinsTxHashs1[$i]="${coinsgStr}"

        sleep 1
        height=$(${name} block last_header | jq ".height")
        printf '发送第 %d 笔交易当前高度 %s \n' $i "${height}"
    done
}

function genSecondChainCoinstx() {
    echo "====== 发送交易 ======"
    name=$CLI4
    echo "当前链为：${name}"
    #将第一条链产生的签名交易发送出去
    for ((i = 0; i < ${#coinsTxSign[*]}; i++)); do
        #发送交易
        sign="${coinsTxSign[$i]}"
        sendCoinsTxHash "${name}" "${sign}"

        coinsTxHashs2[$i]="${coinsgStr}"

        sleep 1
        height=$(${name} block last_header | jq ".height")
        printf '发送第 %d 笔交易当前高度 %s \n' $i "${height}"
    done
}

function checkCoinsResult() {
    name1=$CLI
    name2=$CLI4

    echo "====================检查第一组docker运行结果================="

    for ((i = 0; i < ${#coinsTxHashs1[*]}; i++)); do
        txHash=${coinsTxHashs1[$i]}
        echo $txHash
        txQuery "${name1}" $txHash
        result=$?
        if [ $result -eq 0 ]; then
            coinsTxFee1=$(echo "$coinsTxFee1 + $defaultTxFee" | bc)
        fi
        sleep 1
    done

    #    sleep 1
    #
    #    fromAdd="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
    #    showcoinsBalance "${name1}" $fromAdd
    #    value1=$coinsgStr
    #
    #    sleep 1
    #
    #    fromAdd=$coinsAddr1
    #    showcoinsBalance "${name1}" $fromAdd
    #    value2=$coinsgStr
    #
    #    printf "中间交易费为%d \n" $coinsTxFee1
    #
    #    actTotal=$(echo "$value1 + $value2 + $coinsTxFee1" | bc)
    #    echo "${name1} 实际金额：$actTotal"
    #
    #    if [ "${actTotal}" = $coinsTotalAmount1 ]; then
    #        echo "${name1} 分叉后检查实际金额符合"
    #    else
    #        echo "${name1} 分叉后检查实际金额不符合"
    #    fi

    echo "====================检查第二组docker运行结果================="
    for ((i = 0; i < ${#coinsTxHashs2[*]}; i++)); do
        txHash=${coinsTxHashs2[$i]}
        echo $txHash
        txQuery "${name2}" $txHash
        result=$?
        if [ $result -eq 0 ]; then
            coinsTxFee2=$(echo "$coinsTxFee2 + $defaultTxFee" | bc)
        fi
        sleep 1
    done

    #    fromAdd="1KcCVZLSQYRUwE5EXTsAoQs9LuJW6xwfQa"
    #    showcoinsBalance "${name2}" $fromAdd
    #    value1=$coinsgStr
    #
    #    sleep 1
    #
    #    fromAdd=$coinsAddr2
    #    showcoinsBalance "${name2}" $fromAdd
    #    value2=$coinsgStr
    #
    #    printf "中间交易费为%d \n" $coinsTxFee2
    #
    #    actTotal=$(echo "$value1 + $value2 +  $coinsTxFee2" | bc)
    #    echo "${name2} 实际金额：$actTotal"
    #
    #    if [ "${actTotal}" = $coinsTotalAmount2 ]; then
    #        echo "${name2} 分叉后检查实际金额符合"
    #    else
    #        echo "${name2} 分叉后检查实际金额不符合"
    #    fi
    #
    #    sleep 1
}

function sendcoinsTx() {
    name=$1
    fromAdd=$2
    toAdd=$3
    note=$4
    amount=$5

    #组合形式交易
    #hash=$(sudo docker exec -it $1 ./chain33-cli bty transfer -t $2 -a $5 -n $4 | tr '\r' ' ')
    #echo $hash
    #sign=$(sudo docker exec -it $1 ./chain33-cli wallet sign -a $3 -d $hash | tr '\r' ' ')
    #echo $sign
    #sudo docker exec -it $1 ./chain33-cli wallet send -d $sign

    #单个命令形式交易
    #sudo docker exec -it $1 ./chain33-cli send bty transfer -a $5 -n $4 -t $2 -k $From
    result=$($name send bty transfer -a "${amount}" -n "${note}" -t "${toAdd}" -k "${fromAdd}" | tr '\r' ' ')
    echo "hash: $result"
    coinsgStr=$result
}

function genCoinsTx() {
    name=$1
    fromAdd=$2
    toAdd=$3
    note=$4
    amount=$5

    #组合形式交易
    hash=$(${name} bty transfer -t "${toAdd}" -a "${amount}" -n "${note}" | tr '\r' ' ')
    echo "${hash}"
    sign=$(${name} wallet sign -a "${fromAdd}" -d "${hash}" | tr '\r' ' ')
    echo "sign: $sign"
    coinsgStr=$sign

}

function sendCoinsTxHash() {
    name=$1
    sign=$2
    result=$(${name} wallet send -d "${sign}" | tr '\r' ' ')
    echo "hash: $result"
    coinsgStr=$result
}

function showCoinsBalance() {
    name=$1
    fromAdd=$2
    printf '==========showCoinBalance name=%s addr=%s==========\n' "${name}" "${fromAdd}"
    result=$($name account balance -e coins -a "${fromAdd}" | jq -r ".balance")
    printf 'balance %s \n' "${result}"
    coinsgStr=$result
}

function createAccount() {
    name=$1
    label=$2
    printf '==========CreateAccount name=%s ==========\n' "${name}"
    result=$($name account create -l "${label}" | jq -r ".acc.addr")
    printf 'New account addr %s \n' "${result}"
    coinsgStr=$result
}
