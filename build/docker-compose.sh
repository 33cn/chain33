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
# ./docker-compose.sh build

PWD=$(cd "$(dirname "$0")" && pwd)
export PATH="$PWD:$PATH"

NODE3="${1}_chain33_1"
CLI="docker exec ${NODE3} /root/chain33-cli"

NODE2="${1}_chain32_1"

NODE1="${1}_chain31_1"

NODE4="${1}_chain30_1"

NODE5="${1}_chain29_1"
CLI5="docker exec ${NODE5} /root/chain33-cli"

BTCD="${1}_btcd_1"

RELAYD="${1}_relayd_1"

containers=("${NODE1}" "${NODE2}" "${NODE3}" "${NODE4}" "${BTCD}" "${RELAYD}")

# shellcheck disable=SC1091
source ci-para-test.sh
# shellcheck disable=SC1091
source ci-relay-test.sh

sedfix=""
if [ "$(uname)" == "Darwin" ]; then
    sedfix=".bak"
fi

function init() {
    # update test environment
    sed -i $sedfix 's/^Title.*/Title="local"/g' chain33.toml
    sed -i $sedfix 's/^TestNet=.*/TestNet=true/g' chain33.toml

    # p2p
    sed -i $sedfix 's/^seeds=.*/seeds=["chain33:13802","chain32:13802","chain31:13802"]/g' chain33.toml
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

    # relayd
    sed -i $sedfix 's/^btcdOrWeb.*/btcdOrWeb = 0/g' relayd.toml
    sed -i $sedfix 's/^Tick33.*/Tick33 = 5/g' relayd.toml
    sed -i $sedfix 's/^TickBTC.*/TickBTC = 5/g' relayd.toml
    sed -i $sedfix 's/^pprof.*/pprof = false/g' relayd.toml
    sed -i $sedfix 's/^watch.*/watch = false/g' relayd.toml

}

function start() {
    # docker-compose ps
    docker-compose ps

    # remove exsit container
    docker-compose down

    # create and run docker-compose container
    docker-compose -f docker-compose.yml -f docker-compose-para.yml up --build -d

    local SLEEP=60
    echo "=========== sleep ${SLEEP}s ============="
    sleep ${SLEEP}

    docker-compose ps

    wait_btcd_up
    run_relayd_with_btcd
    ping_btcd

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

#some times btcwallet bin 18554 server port fail in btcd docker, restart btcd will be ok
# [WRN] BTCW: Can't listen on [::1]:18554: listen tcp6 [::1]:18554: bind: cannot assign requested address
function wait_btcd_up() {
    count=20
    while [ $count -gt 0 ]; do
        status=$(docker-compose ps | grep btcd | awk '{print $5}')
        if [ "${status}" == "Up" ]; then
            break
        fi
        docker-compose logs btcd
        docker-compose restart btcd
        docker-compose ps
        echo "==============btcd fail $count  ================="
        ((count--))
        if [ $count == 0 ]; then
            echo "wait btcd up 20 times"
            exit 1
        fi
        #btcd restart need wait 30s
        sleep 30
    done
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

function check_docker_container() {
    echo "============== check_docker_container ==============================="
    for con in "${containers[@]}"; do
        runing=$(docker inspect "${con}" | jq '.[0].State.Running')
        if [ ! "${runing}" ]; then
            docker inspect "${con}"
            echo "check ${con} not actived!"
            exit 1
        fi
    done
}

function wait_btc_height() {
    if [ "$#" -lt 2 ]; then
        echo "wrong wait_btc_height params"
        exit 1
    fi
    count=100
    wait_sec=0
    while [ $count -gt 0 ]; do
        cur=$(${1} relay btc_cur_height | jq ".curHeight")
        if [ "${cur}" -ge "${2}" ]; then
            break
        fi
        ((count--))
        wait_sec=$((wait_sec + 1))
        sleep 1
    done
    echo "wait btc blocks ${wait_sec} s"

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
    echo "=========== stop  ${NODE5} node========== "
    docker stop "${NODE5}"
    sleep 20

    echo "=========== start ${NODE5} node========== "
    docker start "${NODE5}"

    sleep 1
    sync_status "${CLI5}"
}

function transfer() {
    echo "=========== # transfer ============="
    hashes=()
    for ((i = 0; i < 10; i++)); do
        hash=$(${CLI} send coins transfer -a 1 -n test -t 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt -k 4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01)
        hashes=("${hashes[@]}" "$hash")
    done
    block_wait "${CLI}" 1
    echo "len: ${#hashes[@]}"
    if [ "${#hashes[@]}" != 10 ]; then
        echo "tx number wrong"
        exit 1
    fi

    for ((i = 0; i < ${#hashes[*]}; i++)); do
        txs=$(${CLI} tx query_hash -s "${hashes[$i]}" | jq ".txs")
        if [ -z "${txs}" ]; then
            echo "cannot find tx"
            exit 1
        fi
    done

    echo "=========== # withdraw ============="
    hash=$(${CLI} send coins transfer -a 2 -n deposit -t 1wvmD6RNHzwhY4eN75WnM6JcaAvNQ4nHx -k CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944)
    echo "${hash}"
    block_wait "${CLI}" 1
    before=$(${CLI} account balance -a 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt -e retrieve | jq -r ".balance")
    if [ "${before}" == "0.0000" ]; then
        echo "wrong ticket balance, should not be zero"
        exit 1
    fi

    hash=$(${CLI} send coins withdraw -a 1 -n withdraw -e retrieve -k CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944)
    echo "${hash}"
    block_wait "${CLI}" 1
    txs=$(${CLI} tx query_hash -s "${hash}" | jq ".txs")
    if [ "${txs}" == "null" ]; then
        echo "withdraw cannot find tx"
        exit 1
    fi
}

function main() {
    echo "==========================================main begin========================================================"
    init
    para_init
    start
    para_transfer
    para_set_wallet
    sync
    transfer

    para
    relay "${CLI}"
    # TODO other work!!!

    check_docker_container
    echo "==========================================main end========================================================="
}

# run script
main
