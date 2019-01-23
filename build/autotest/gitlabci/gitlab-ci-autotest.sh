#!/usr/bin/env bash

set -e
set -o pipefail
#set -o verbose
#set -o xtrace

# os: ubuntu16.04 x64
# first, you must install jq tool of json
# sudo apt-get install jq
# sudo apt-get install shellcheck, in order to static check shell script
# sudo apt-get install parallel
# ./run-autotest.sh build

PWD=$(cd "$(dirname "$0")" && pwd)
export PATH="$PWD:$PATH"

PROJECT_NAME="${1}"

NODE3="autotest-chain33"
CLI="docker exec ${NODE3} /root/chain33-cli"

NODE2="autotest-chain32"
CLI2="docker exec ${NODE2} /root/chain33-cli"

sedfix=""
if [ "$(uname)" == "Darwin" ]; then
    sedfix=".bak"
fi

function init() {
    # update test environment
    sed -i $sedfix 's/^Title.*/Title="local"/g' chain33.toml
    sed -i $sedfix 's/^TestNet=.*/TestNet=true/g' chain33.toml

    # p2p
    sed -i $sedfix 's/^seeds=.*/seeds=["chain33:13802","chain32:13802"]/g' chain33.toml
    #sed -i $sedfix 's/^enable=.*/enable=true/g' chain33.toml
    sed -i $sedfix '0,/^enable=.*/s//enable=true/' chain33.toml
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

    # remove exsit container
    docker-compose -p "${PROJECT_NAME}" -f compose-autotest.yml down --remove-orphans

    # create and run docker-compose container
    docker-compose -p "${PROJECT_NAME}" -f compose-autotest.yml up --build -d

    local SLEEP=30
    echo "=========== sleep ${SLEEP}s ============="
    sleep ${SLEEP}

    # docker-compose ps
    docker-compose -p "${PROJECT_NAME}" -f compose-autotest.yml ps

    # query node run status
    ${CLI} block last_header
    ${CLI} net info

    ${CLI} net peer_info
    peersCount=$(${CLI} net peer_info | jq '.[] | length')
    echo "${peersCount}"
    if [ "${peersCount}" -lt 2 ]; then
        echo "peers error"
        exit 1
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

    echo "=========== # import private key returnAddr ============="
    result=$(${CLI} account import_key -k CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944 -l returnAddr | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi

    echo "=========== # import private key mining ============="
    result=$(${CLI} account import_key -k 4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01 -l minerAddr | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi

    echo "=========== # close auto mining ============="
    result=$(${CLI} wallet auto_mine -f 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        exit 1
    fi

    echo "=========== sleep ${SLEEP}s ============="
    sleep ${SLEEP}

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

    sync_status "${CLI}"

    ${CLI} wallet status
    ${CLI} account list
    ${CLI} mempool list
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

function sync_status() {
    echo "=========== query sync status========== "
    local sync_status
    local count=100
    local wait_sec=0
    while [ $count -gt 0 ]; do
        sync_status=$(${1} net is_sync)
        if [ "${sync_status}" = "true" ]; then
            break
        fi
        ((count--))
        wait_sec=$((wait_sec + 1))
        sleep 1
    done
    echo "sync wait  ${wait_sec} s"

    echo "=========== query clock sync status========== "
    sync_status=$(${1} net is_clock_sync)
    if [ "${sync_status}" = "false" ]; then
        exit 1
    fi
}

function sync() {
    echo "=========== stop  ${NODE2} node========== "
    docker stop "${NODE2}"
    sleep 20

    echo "=========== start ${NODE2} node========== "
    docker start "${NODE2}"

    sleep 1
    sync_status "${CLI2}"
}

function auto_test() {

    echo "=========== #run auto test ============="
    echo "=========== #transfer to token amdin ============="
    hash=$(${CLI} send coins transfer -a 10 -n test -t 1Q8hGLfoGe63efeWa8fJ4Pnukhkngt6poK -k 4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01)

    block_wait "${CLI}" 2
    txs=$(${CLI} tx query_hash -s "${hash}" | jq ".txs")
    if [ "${txs}" == "null" ]; then
        echo "transferTokenAdmin cannot find tx"
        exit 1
    fi

    echo "=========== #config token blacklist ============="
    rawData=$(${CLI} config config_tx -c token-blacklist -o add -v BTC)
    signData=$(${CLI} wallet sign -d "${rawData}" -k 0xc34b5d9d44ac7b754806f761d3d4d2c4fe5214f6b074c19f069c4f5c2a29c8cc)
    hash=$(${CLI} wallet send -d "${signData}")
    block_wait "${CLI}" 2

    echo "=========== #start auto-test program ============="
    docker exec "${NODE3}" /root/autotest
}

function stop_chain33() {

    rv=$?
    echo "=========== #stop docker-compose ============="
    docker-compose -p "${PROJECT_NAME}" -f compose-autotest.yml down && rm -rf ./chain33* ./*.toml ./autotest
    echo "=========== #remove related images ============"
    docker rmi "${PROJECT_NAME}"_autotest || true
    exit ${rv}
}

function main() {
    echo "==========================================main begin========================================================"
    init
    start
    auto_test
    echo "==========================================main end========================================================="
}

# check args
if [ "$#" -ne 1 ]; then
    echo "Suggest Usage: $0 build"
    exit 1
fi

#trap exit
trap "stop_chain33" INT TERM EXIT

# run script
main
