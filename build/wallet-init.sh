#!/usr/bin/env bash

function SendToPrivacyExec() {
    name=$1
    fromAdd=$2
    execAdd=$3
    note=$4
    amount=$5
    #sudo docker exec -it $name ./chain33-cli send bty transfer -k $fromAdd -t $execAdd -n $note -a $amount
    result=$($name send bty transfer -k "${fromAdd}" -t "${execAdd}" -n "${note}" -a "${amount}")
    echo "hash : $result"
}

function pub2priv() {
    name=$1
    fromAdd=$2
    priAdd=$3
    note=$4
    amount=$5
    #sudo docker exec -it $name ./chain33-cli privacy pub2priv -f $fromAdd -p $priAdd -a $amount -n $note
    result=$($name privacy pub2priv -f "${fromAdd}" -p "${priAdd}" -a "${amount}" -n "${note}" | jq -r ".hash")
    echo "hash : $result"
}

function showPrivacyExec() {
    name=$1
    fromAdd=$2
    printf '==========showPrivacyExec name=%s addr=%s==========\n' "${name}" "${fromAdd}"
    result=$($name account balance -e privacy -a "${fromAdd}" | jq -r ".balance")
    printf 'balance %s \n' "${result}"
}

function showPrivacyBalance() {
    name=$1
    fromAdd=$2
    printf '==========showPrivacyBalance name=%s addr=%s==========\n' "${name}" "${fromAdd}"
    result=$($name privacy showpai -a "${fromAdd}" -d 0 | jq -r ".AvailableAmount")
    printf 'AvailableAmount %s \n' "${result}"
}

function init() {
    echo "=========== # start set wallet 1 ============="
    echo "=========== # save seed to wallet ============="
    result=$(./chain33-cli seed save -p 1314 -s "tortoise main civil member grace happy century convince father cage beach hip maid merry rib" | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "save seed to wallet error seed, result: ${result}"
        exit 1
    fi

    sleep 2

    echo "=========== # unlock wallet ============="
    result=$(./chain33-cli wallet unlock -p 1314 -t 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        exit 1
    fi

    sleep 2

    echo "=========== # import private key transfer ============="
    result=$(./chain33-cli account import_key -k CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944 -l transfer | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi

    sleep 2

    echo "=========== # import private key mining ============="
    result=$(./chain33-cli account import_key -k 4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01 -l mining | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi

    sleep 2
    echo "=========== # set auto mining ============="
    result=$(./chain33-cli wallet auto_mine -f 1 | jq ".isok")
    if [ "${result}" = "false" ]; then
        exit 1
    fi

    echo "=========== # end set wallet 1 ============="

}

init

#构建隐私交易
#    sleep 20
#    name="./chain33-cli"
#    fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
#    execAdd="1FeyE6VDZ4FYgpK1n2okWMDAtPkwBuooQd"
#    note="test"
#    amount=100
#    SendToPrivacyExec "${name}" $fromAdd $execAdd $note $amount
#
#    sleep 30
#
#    fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
#    priAdd="0a9d212b2505aefaa8da370319088bbccfac097b007f52ed71d8133456c8185823c8eac43c5e937953d7b6c8e68b0db1f4f03df4946a29f524875118960a35fb"
#    note="pub2priv_test"
#    amount=10
#    pub2priv "${name}" $fromAdd $priAdd $note $amount
#
#
#    fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
#    showPrivacyExec "${name}" $fromAdd
#
#    fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
#    showPrivacyBalance "${name}" $fromAdd
#
#    fromAdd="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
#    showPrivacyBalance "${name}" $fromAdd
