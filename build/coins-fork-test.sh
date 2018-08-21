#!/usr/bin/env bash

coinsTxHashs1=("")
#coinsTxHashs2=("")
coinsgStr=""
coinsRepeatTx=1 #重复发送交易次数
coinsTotalAmount1="0"
coinsTxAmount="5" #每次发送交易数
#defaultTxFee="0.001"
coinsAddr1=""
coinsTxSign=("")

function initCoinsAccount() {

    name="${CLI}"
    label="coinstest1"
    createAccount "${name}" $label
    coinsAddr1=$coinsgStr

    sleep 1

    name="${CLI}"
    fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
    showCoinsBalance "${name}" $fromAdd
    coinsTotalAmount1=$coinsgStr

    shouAccountList "${name}"
}

function genFirstChainCoinstx() {
    echo "====== 发送coins交易 ======"
    name=$CLI
    echo "当前链为：${name}"

    for ((i = 0; i < coinsRepeatTx; i++)); do
        From="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
        to=$coinsAddr1
        note="coinstx"
        amount=$coinsTxAmount

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
    name=$CLI
    echo "当前链为：${name}"

    #将第一条链产生的签名交易发送出去
    for ((i = 0; i < ${#coinsTxSign[*]}; i++)); do
        #发送交易
        sign="${coinsTxSign[$i]}"
        sendCoinsTxHash "${name}" "${sign}"

        sleep 1
        height=$(${name} block last_header | jq ".height")
        printf '发送第 %d 笔交易当前高度 %s \n' $i "${height}"
    done
}

function checkCoinsResult() {
    name=$CLI

    echo "====================检查第一组docker运行结果================="

    totalCoinsTxAmount="0"

    for ((i = 0; i < ${#coinsTxHashs1[*]}; i++)); do
        txHash=${coinsTxHashs1[$i]}
        echo $txHash
        txQuery "${name}" $txHash
        result=$?
        if [ $result -eq 0 ]; then
            #coinsTxFee1=$(echo "$coinsTxFee1 + $defaultTxFee" | bc)
            totalCoinsTxAmount=$(echo "$totalCoinsTxAmount + $coinsTxAmount" | bc)
        fi
        sleep 1
    done

    sleep 1

    fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
    showCoinsBalance "${name}" $fromAdd
    value1=$coinsgStr

    sleep 1

    fromAdd=$coinsAddr1
    showCoinsBalance "${name}" $fromAdd
    value2=$coinsgStr

    actTotal=$(echo "$value1 + $value2 " | bc)
    echo "${name} 实际金额：$actTotal"

    #if [ `echo "${actTotal} > ${coinsTotalAmount1}" | bc` -eq 1 ]; then
    if [ "$(echo "${actTotal} > ${coinsTotalAmount1}" | bc)" -eq 1 ]; then
        echo "${name} 出现双花"
        exit 1
    else
        echo "${name} 未出现双花"
    fi
}

function sendcoinsTx() {
    name=$1
    fromAdd=$2
    toAdd=$3
    note=$4
    amount=$5

    #组合形式交易
    #hash=$(sudo docker exec -it $1 ./chain33-cli coins transfer -t $2 -a $5 -n $4 | tr '\r' ' ')
    #echo $hash
    #sign=$(sudo docker exec -it $1 ./chain33-cli wallet sign -a $3 -d $hash | tr '\r' ' ')
    #echo $sign
    #sudo docker exec -it $1 ./chain33-cli wallet send -d $sign

    #单个命令形式交易
    #sudo docker exec -it $1 ./chain33-cli send coins transfer -a $5 -n $4 -t $2 -k $From
    result=$($name send coins transfer -a "${amount}" -n "${note}" -t "${toAdd}" -k "${fromAdd}" | tr '\r' ' ')
    echo "hash: $result"
    coinsgStr=$result
}

function genCoinsTx() {
    name=$1
    fromAdd=$2
    toAdd=$3
    note=$4
    amount=$5
    expire="600s"

    #组合形式交易
    hash=$(${name} coins transfer -t "${toAdd}" -a "${amount}" -n "${note}" | tr '\r' ' ')
    echo "${hash}"
    sign=$(${name} wallet sign -a "${fromAdd}" -d "${hash}" -e "${expire}" | tr '\r' ' ')
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

function shouAccountList() {
    name=$1
    printf '==========shouAccountList name=%s ==========\n' "${name}"
    $name account list
}
