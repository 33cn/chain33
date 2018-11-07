#!/usr/bin/env bash

PARA_CLI="docker exec ${NODE3} /root/chain33-para-cli"

PARA_CLI2="docker exec ${NODE2} /root/chain33-para-cli"
PARA_CLI1="docker exec ${NODE1} /root/chain33-para-cli"
PARA_CLI4="docker exec ${NODE4} /root/chain33-para-cli"

PARANAME="para"

xsedfix=""
if [ "$(uname)" == "Darwin" ]; then
    xsedfix=".bak"
fi

function para_init() {
    para_set_toml chain33.para33.toml
    para_set_toml chain33.para32.toml
    para_set_toml chain33.para31.toml
    para_set_toml chain33.para30.toml

    sed -i $xsedfix 's/^authAccount=.*/authAccount="1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4"/g' chain33.para33.toml
    sed -i $xsedfix 's/^authAccount=.*/authAccount="1JRNjdEqp4LJ5fqycUBm9ayCKSeeskgMKR"/g' chain33.para32.toml
    sed -i $xsedfix 's/^authAccount=.*/authAccount="1NLHPEcbTWWxxU3dGUZBhayjrCHD3psX7k"/g' chain33.para31.toml
    sed -i $xsedfix 's/^authAccount=.*/authAccount="1MCftFynyvG2F4ED5mdHYgziDxx6vDrScs"/g' chain33.para30.toml
}

function para_set_toml() {
    cp chain33.para.toml "${1}"

    sed -i $xsedfix 's/^Title.*/Title="user.p.'''$PARANAME'''."/g' "${1}"
    sed -i $xsedfix 's/^# TestNet=.*/TestNet=true/g' "${1}"
    sed -i $xsedfix 's/^startHeight=.*/startHeight=20/g' "${1}"
    sed -i $xsedfix 's/^emptyBlockInterval=.*/emptyBlockInterval=4/g' "${1}"

    # rpc
    sed -i $xsedfix 's/^jrpcBindAddr=.*/jrpcBindAddr="0.0.0.0:8901"/g' "${1}"
    sed -i $xsedfix 's/^grpcBindAddr=.*/grpcBindAddr="0.0.0.0:8902"/g' "${1}"
    sed -i $xsedfix 's/^whitelist=.*/whitelist=["localhost","127.0.0.1","0.0.0.0"]/g' "${1}"
}

function para_set_wallet() {
    echo "=========== # para set wallet ============="
    para_import_key "${PARA_CLI}" "0x6da92a632ab7deb67d38c0f6560bcfed28167998f6496db64c258d5e8393a81b"
    para_import_key "${PARA_CLI2}" "0x19c069234f9d3e61135fefbeb7791b149cdf6af536f26bebb310d4cd22c3fee4"
    para_import_key "${PARA_CLI1}" "0x7a80a1f75d7360c6123c32a78ecf978c1ac55636f87892df38d8b85a9aeff115"
    para_import_key "${PARA_CLI4}" "0xcacb1f5d51700aea07fca2246ab43b0917d70405c65edea9b5063d72eb5c6b71"
}

function para_import_key() {
    echo "=========== # save seed to wallet ============="
    result=$(${1} seed save -p 1314 -s "tortoise main civil member grace happy century convince father cage beach hip maid merry rib" | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "save seed to wallet error seed, result: ${result}"
        exit 1
    fi

    echo "=========== # unlock wallet ============="
    result=$(${1} wallet unlock -p 1314 -t 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        exit 1
    fi

    echo "=========== # import private key ============="
    echo "key: ${2}"
    result=$(${1} account import_key -k "${2}" -l paraAuthAccount | jq ".label")
    if [ -z "${result}" ]; then
        exit 1
    fi

    echo "=========== # close auto mining ============="
    result=$(${1} wallet auto_mine -f 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        exit 1
    fi
    echo "=========== # wallet status ============="
    ${1} wallet status
}

function para_transfer() {
    echo "=========== # para chain transfer ============="
    para_transfer2account "1Q8hGLfoGe63efeWa8fJ4Pnukhkngt6poK"
    para_transfer2account "1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4"
    para_transfer2account "1JRNjdEqp4LJ5fqycUBm9ayCKSeeskgMKR"
    para_transfer2account "1NLHPEcbTWWxxU3dGUZBhayjrCHD3psX7k"
    para_transfer2account "1MCftFynyvG2F4ED5mdHYgziDxx6vDrScs"
    block_wait "${CLI}" 1

    echo "=========== # para chain send config ============="
    para_configkey "${CLI}" "paracross-nodes-user.p.${PARANAME}." "1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4"
    para_configkey "${CLI}" "paracross-nodes-user.p.${PARANAME}." "1JRNjdEqp4LJ5fqycUBm9ayCKSeeskgMKR"
    para_configkey "${CLI}" "paracross-nodes-user.p.${PARANAME}." "1NLHPEcbTWWxxU3dGUZBhayjrCHD3psX7k"
    para_configkey "${CLI}" "paracross-nodes-user.p.${PARANAME}." "1MCftFynyvG2F4ED5mdHYgziDxx6vDrScs"
    block_wait "${CLI}" 1

    txhash=$(para_configkey "${PARA_CLI}" "token-blacklist" "BTY")
    echo "txhash=$txhash"
    block_wait "${PARA_CLI}" 1
    $PARA_CLI tx query -s "${txhash}"

}

function para_transfer2account() {
    echo "${1}"
    hash1=$(${CLI} send coins transfer -a 10 -n test -t "${1}" -k 4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01)
    echo "${hash1}"
}

function para_configkey() {
    tx=$(${1} config config_tx -o add -k "${2}" -v "${3}")
    sign=$(${CLI} wallet sign -k 0xc34b5d9d44ac7b754806f761d3d4d2c4fe5214f6b074c19f069c4f5c2a29c8cc -d "${tx}")
    send=$(${CLI} wallet send -d "${sign}")
    echo "${send}"
}

function token_create() {
    echo "=========== # para token test ============="
    echo "=========== # 1.token precreate ============="
    hash=$(${1} send token precreate -f 0.001 -i test -n guodunjifen -a 1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4 -p 0 -s GD -t 10000 -k 1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4)
    echo "${hash}"
    block_wait "${1}" 3

    ${1} tx query -s "${hash}"
    ${1} token get_precreated
    owner=$(${1} token get_precreated | jq -r ".owner")
    if [ "${owner}" != "1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4" ]; then
        echo "wrong pre create owner"
        exit 1
    fi
    total=$(${1} token get_precreated | jq -r ".total")
    if [ "${total}" != 10000 ]; then
        echo "wrong pre create total"
        exit 1
    fi

    echo "=========== # 2.token finish ============="
    hash=$(${1} send token finish -f 0.001 -a 1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4 -s GD -k 0xc34b5d9d44ac7b754806f761d3d4d2c4fe5214f6b074c19f069c4f5c2a29c8cc)
    echo "${hash}"
    block_wait "${1}" 3

    ${1} tx query -s "${hash}"
    ${1} token get_finish_created
    owner=$(${1} token get_finish_created | jq -r ".owner")
    if [ "${owner}" != "1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4" ]; then
        echo "wrong finish created owner"
        exit 1
    fi
    total=$(${1} token get_finish_created | jq -r ".total")
    if [ "${total}" != 10000 ]; then
        echo "wrong finish created total"
        exit 1
    fi

    ${1} token token_balance -a 1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4 -e token -s GD
    balance=$(${1} token token_balance -a 1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4 -e token -s GD | jq -r '.[]|.balance')
    if [ "${balance}" != "10000.0000" ]; then
        echo "wrong para token genesis create, should be 10000.0000"
        exit 1
    fi
    echo "=========== # 2.token transfer ============="
    hash=$(${1} send token transfer -a 11 -s GD -t 1GGF8toZd96wCnfJngTwXZnWCBdWHYYvjw -k 0x6da92a632ab7deb67d38c0f6560bcfed28167998f6496db64c258d5e8393a81b)
    echo "${hash}"
    block_wait "${1}" 3

    ${1} tx query -s "${hash}"
    ${1} token token_balance -a 1GGF8toZd96wCnfJngTwXZnWCBdWHYYvjw -e token -s GD
    balance=$(${1} token token_balance -a 1GGF8toZd96wCnfJngTwXZnWCBdWHYYvjw -e token -s GD | jq -r '.[]|.balance')
    if [ "${balance}" != "11.0000" ]; then
        echo "wrong para token transfer, should be 11.0000"
        exit 1
    fi
}

function para_cross_transfer_withdraw() {
    echo "=========== # para cross transfer/withdraw test ============="
    paracrossAddr=1HPkPopVe3ERfvaAgedDtJQ792taZFEHCe
    ${CLI} account list
    ${CLI} send bty transfer -a 10 -n test -t $paracrossAddr -k 4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01
    hash=$(${CLI} send para asset_transfer --title user.p.para. -a 1.4 -n test -t 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv -k 4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01)
    echo "${hash}"

    sleep 15
    ${CLI} send para asset_withdraw --title user.p.para. -a 0.7 -n test -t 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv -k 4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01

    times=100
    while true; do
        acc=$(${CLI} account balance -e paracross -a 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv | jq -r ".balance")
        echo "account balance is ${acc}, except 9.3 "
        if [ "${acc}" != "9.3000" ]; then
            block_wait "${CLI}" 2
            times=$((times - 1))
            if [ $times -le 0 ]; then
                echo "para_cross_transfer_withdraw failed"
                exit 1
            fi
        else
            echo "para_cross_transfer_withdraw success"
            break
        fi
    done
}

function para_test() {
    echo "=========== # para chain test ============="
    token_create "${PARA_CLI}"
    para_cross_transfer_withdraw
}

function paracross() {
    if [ "${2}" == "init" ]; then
        para_init
    elif [ "${2}" == "config" ]; then
        para_transfer
        para_set_wallet
    elif [ "${2}" == "test" ]; then
        para_test "${1}"
    fi

    if [ "${2}" == "forkInit" ]; then
        para_init
    elif [ "${2}" == "forkConfig" ]; then
        para_transfer
        para_set_wallet
    elif [ "${2}" == "forkCheckRst" ]; then
        checkParaBlockHashfun 30
    fi

    if [ "${2}" == "fork2Init" ]; then
        para_init
    elif [ "${2}" == "fork2Config" ]; then
        para_transfer
        para_set_wallet
    elif [ "${2}" == "fork2CheckRst" ]; then
        checkParaBlockHashfun 30
    fi

}
