#!/usr/bin/env bash

PARA_CLI="docker exec ${NODE3} /root/chain33-para-cli"

PARA_CLI2="docker exec ${NODE2} /root/chain33-para-cli"
PARA_CLI1="docker exec ${NODE1} /root/chain33-para-cli"
PARA_CLI4="docker exec ${NODE4} /root/chain33-para-cli"

forkParaContainers=("${PARA_CLI}" "${PARA_CLI2}" "${PARA_CLI1}" "${PARA_CLI4}")

PARANAME="para"

xsedfix=""
if [ "$(uname)" == "Darwin" ]; then
    xsedfix=".bak"
fi

function para_init() {
    echo "=========== # para chain init toml ============="
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
    result=$(${1} account import_key -k "${2}" -l returnAddr | jq ".label")
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

    para_config "${CLI}" "paracross-nodes-user.p.${PARANAME}." "1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4"
    para_config "${CLI}" "paracross-nodes-user.p.${PARANAME}." "1JRNjdEqp4LJ5fqycUBm9ayCKSeeskgMKR"
    para_config "${CLI}" "paracross-nodes-user.p.${PARANAME}." "1NLHPEcbTWWxxU3dGUZBhayjrCHD3psX7k"
    para_config "${CLI}" "paracross-nodes-user.p.${PARANAME}." "1MCftFynyvG2F4ED5mdHYgziDxx6vDrScs"

    para_config "${PARA_CLI}" "token-blacklist" "BTY"

}

function para_transfer2account() {
    echo "${1}"
    hash1=$(${CLI} send coins transfer -a 10 -n test -t "${1}" -k 4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01)
    echo "${hash1}"
}

function para_config() {
    echo "=========== # para chain send config ============="
    echo "${3}"
    tx=$(${1} config config_tx -o add -k "${2}" -v "${3}")
    echo "${tx}"
    sign=$(${CLI} wallet sign -k 0xc34b5d9d44ac7b754806f761d3d4d2c4fe5214f6b074c19f069c4f5c2a29c8cc -d "${tx}")
    echo "${sign}"
    send=$(${CLI} wallet send -d "${sign}")
    echo "${send}"
}

function token_create() {
    echo "=========== # para token test ============="
    hash=$(${1} send token precreate -f 0.001 -i test -n guodunjifen -a 1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4 -p 0 -s GD -t 10000 -k 1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4)
    echo "${hash}"
    block_wait "${1}" 2

    owner=$(${1} tx query -s "${hash}" | jq -r ".receipt.logs[0].log.owner")
    if [ "${owner}" != "1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4" ]; then
        echo "wrong pre create owner"
        exit 1
    fi

    hash=$(${1} send token finish -f 0.001 -a 1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4 -s GD -k 0xc34b5d9d44ac7b754806f761d3d4d2c4fe5214f6b074c19f069c4f5c2a29c8cc)
    echo "${hash}"
    block_wait "${1}" 2

    owner=$(${1} tx query -s "${hash}" | jq -r ".receipt.logs[1].log.owner")
    if [ "${owner}" != "1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4" ]; then
        echo "wrong finish create owner"
        exit 1
    fi

}
function para() {
    echo "=========== # para chain test ============="
    token_create "${PARA_CLI}"
}

#================fork-test============================
function checkParaBlockHashfun() {
    echo "====== syn para blockchain ======"

    height=0
    hash=""
    height1=$($PARA_CLI block last_header | jq ".height")
    sleep 1
    height2=$($PARA_CLI4 block last_header | jq ".height")

    if [ "${height2}" -ge "${height1}" ]; then
        height=$height2
        printf "主链为 $PARA_CLI 当前最大高度 %d \\n" "${height}"
        sleep 1
        hash=$($CLI block hash -t "${height}" | jq ".hash")
    else
        height=$height1
        printf "主链为 $PARA_CLI4 当前最大高度 %d \\n" "${height}"
        sleep 1
        hash=$($CLI4 block hash -t "${height}" | jq ".hash")
    fi

    for ((j = 0; j < $1; j++)); do
        for ((k = 0; k < ${#forkParaContainers[*]}; k++)); do
            sleep 1
            height0[$k]=$(${forkParaContainers[$k]} block last_header | jq ".height")
            if [ "${height0[$k]}" -ge "${height}" ]; then
                sleep 1
                hash0[$k]=$(${forkParaContainers[$k]} block hash -t "${height}" | jq ".hash")
            else
                hash0[$k]="${forkParaContainers[$k]}"
            fi
        done

        if [ "${hash0[0]}" = "${hash}" ] && [ "${hash0[1]}" = "${hash}" ] && [ "${hash0[2]}" = "${hash}" ] && [ "${hash0[3]}" = "${hash}" ]; then
            echo "syn para blockchain success break"
            break
        else
            if [ "${hash0[1]}" = "${hash0[0]}" ] && [ "${hash0[2]}" = "${hash0[0]}" ] && [ "${hash0[3]}" = "${hash0[0]}" ]; then
                echo "syn para blockchain success break"
                break
            fi
        fi

        printf '第 %d 次，10s后查询\n' $j
        sleep 10
        #检查是否超过了最大检测次数
        var=$(($1 - 1))
        if [ $j -ge "${var}" ]; then
            echo "====== syn para blockchain fail======"
            exit 1
        fi
    done
    echo "====== syn para blockchain success======"
}
