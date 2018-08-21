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
PARA_CLI="docker exec ${NODE3} /root/chain33-para-cli"

NODE2="${1}_chain32_1"
CLI2="docker exec ${NODE2} /root/chain33-cli"
PARA_CLI2="docker exec ${NODE2} /root/chain33-para-cli"

NODE1="${1}_chain31_1"
#CLI1="docker exec ${NODE1} /root/chain33-cli"
PARA_CLI1="docker exec ${NODE1} /root/chain33-para-cli"

NODE0="${1}_chain30_1"
PARA_CLI0="docker exec ${NODE0} /root/chain33-para-cli"

NODE9="${1}_chain29_1"
CLI9="docker exec ${NODE9} /root/chain33-cli"

BTCD="${1}_btcd_1"
BTC_CTL="docker exec ${BTCD} btcctl"

RELAYD="${1}_relayd_1"

containers=("${NODE1}" "${NODE2}" "${NODE3}" "${BTCD}" "${RELAYD}")

PARANAME="para"
CLIS=("${PARA_CLI0}" "${PARA_CLI1}" "${PARA_CLI2}" "${PARA_CLI}")

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

function para_set_env() {
    echo "=========== # para chain init toml ============="

    para_set_toml chain33.para33.toml
    para_set_toml chain33.para32.toml
    para_set_toml chain33.para31.toml
    para_set_toml chain33.para30.toml

    sed -i $sedfix 's/^authAccount=.*/authAccount="1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4"/g' chain33.para33.toml
    sed -i $sedfix 's/^authAccount=.*/authAccount="1JRNjdEqp4LJ5fqycUBm9ayCKSeeskgMKR"/g' chain33.para32.toml
    sed -i $sedfix 's/^authAccount=.*/authAccount="1NLHPEcbTWWxxU3dGUZBhayjrCHD3psX7k"/g' chain33.para31.toml
    sed -i $sedfix 's/^authAccount=.*/authAccount="1MCftFynyvG2F4ED5mdHYgziDxx6vDrScs"/g' chain33.para30.toml
}

function para_set_toml() {
    cp chain33.para.toml "${1}"

    sed -i $sedfix 's/^Title.*/Title="user.p.'''$PARANAME'''."/g' "${1}"
    sed -i $sedfix 's/^# TestNet=.*/TestNet=true/g' "${1}"
    sed -i $sedfix 's/^startHeight=.*/startHeight=20/g' "${1}"
    sed -i $sedfix 's/^emptyBlockInterval=.*/emptyBlockInterval=4/g' "${1}"

    # rpc
    sed -i $sedfix 's/^jrpcBindAddr=.*/jrpcBindAddr="0.0.0.0:8901"/g' "${1}"
    sed -i $sedfix 's/^grpcBindAddr=.*/grpcBindAddr="0.0.0.0:8902"/g' "${1}"
    sed -i $sedfix 's/^whitelist=.*/whitelist=["localhost","127.0.0.1","0.0.0.0"]/g' "${1}"
}

function para_init() {
    echo "=========== # para chain init  ============="
    echo "=========== # save seed to wallet ============="
    for cli in "${CLIS[@]}"; do
        result=$(${cli} seed save -p 1314 -s "tortoise main civil member grace happy century convince father cage beach hip maid merry rib" | jq ".isok")
        if [ "${result}" = "false" ]; then
            echo "save seed to wallet error seed, result: ${result}"
            exit 1
        fi
    done

    sleep 1

    echo "=========== # unlock wallet ============="
    for cli in "${CLIS[@]}"; do
        result=$(${cli} wallet unlock -p 1314 -t 0 | jq ".isok")
        if [ "${result}" = "false" ]; then
            exit 1
        fi
    done

    echo "=========== # import private key to PARA ============="
    result=$(${PARA_CLI} account import_key -k 6da92a632ab7deb67d38c0f6560bcfed28167998f6496db64c258d5e8393a81b -l returnAddr | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi
    result=$(${PARA_CLI2} account import_key -k 0x19c069234f9d3e61135fefbeb7791b149cdf6af536f26bebb310d4cd22c3fee4 -l returnAddr | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi
    result=$(${PARA_CLI1} account import_key -k 0x7a80a1f75d7360c6123c32a78ecf978c1ac55636f87892df38d8b85a9aeff115 -l returnAddr | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi
    result=$(${PARA_CLI0} account import_key -k 0xcacb1f5d51700aea07fca2246ab43b0917d70405c65edea9b5063d72eb5c6b71 -l returnAddr | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi

    echo "=========== # close auto mining ============="
    for cli in "${CLIS[@]}"; do
        result=$(${cli} wallet auto_mine -f 0 | jq ".isok")
        if [ "${result}" = "false" ]; then
            exit 1
        fi
        ${cli} wallet status
    done

}

function para_transfer() {
    echo "=========== # para chain transfer ============="
    #hash1=$(${CLI} send coins transfer -a 10 -n test -t 1Q8hGLfoGe63efeWa8fJ4Pnukhkngt6poK -k 4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01)
    para_transfer2accout "1Q8hGLfoGe63efeWa8fJ4Pnukhkngt6poK"
    para_transfer2accout "1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4"
    para_transfer2accout "1JRNjdEqp4LJ5fqycUBm9ayCKSeeskgMKR"
    para_transfer2accout "1NLHPEcbTWWxxU3dGUZBhayjrCHD3psX7k"
    para_transfer2accout "1MCftFynyvG2F4ED5mdHYgziDxx6vDrScs"
    block_wait "${CLI}" 1

    para_config "${CLI}" "paracross-nodes-user.p.${PARANAME}." "1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4"
    para_config "${CLI}" "paracross-nodes-user.p.${PARANAME}." "1JRNjdEqp4LJ5fqycUBm9ayCKSeeskgMKR"
    para_config "${CLI}" "paracross-nodes-user.p.${PARANAME}." "1NLHPEcbTWWxxU3dGUZBhayjrCHD3psX7k"
    para_config "${CLI}" "paracross-nodes-user.p.${PARANAME}." "1MCftFynyvG2F4ED5mdHYgziDxx6vDrScs"
    para_config "${CLI}" "paracross-nodes-user.p.${PARANAME}." "1MCftFynyvG2F4ED5mdHYgziDxx6vDrScs"

    para_config "${PARA_CLI}" "token-blacklist" "BTY"

}

function para_transfer2accout() {
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

function run_relayd_with_btcd() {
    echo "============== run_relayd_with_btcd ==============================="
    docker cp "${BTCD}:/root/rpc.cert" ./
    docker cp ./rpc.cert "${RELAYD}:/root/"
    docker restart "${RELAYD}"
}

function ping_btcd() {
    echo "============== ping_btcd ==============================="
    ${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet --wallet listaccounts
    ${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet generate 100
    ${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet generate 1
    ${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet --wallet getaddressesbyaccount default
    ${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet --wallet listaccounts
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
    sync_status=$(${1} net is_sync)
    if [ "${sync_status}" = "false" ]; then
        ${1} block last_header
        exit 1
    fi

    echo "=========== query clock sync status========== "
    sync_status=$(${1} net is_clock_sync)
    if [ "${sync_status}" = "false" ]; then
        exit 1
    fi
}

function sync() {
    echo "=========== stop  ${NODE9} node========== "
    docker stop "${NODE9}"
    sleep 20

    echo "=========== start ${NODE9} node========== "
    docker start "${NODE9}"

    for i in $(seq 20); do
        sleep 1
        ${CLI2} net is_sync
        ${CLI2} block last_header
    done

    sync_status "${CLI9}"
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

function relay_before() {
    sed -i $sedfix 's/ForkV7AddRelay.*/ForkV7AddRelay = 2/g' ../types/relay.go
}

function relay_after() {
    git checkout ../types/relay.go
}

function relay() {
    echo "================relayd========================"

    block_wait "${1}" 2

    ${1} relay btc_cur_height
    base_height=$(${1} relay btc_cur_height | jq ".baseHeight")
    btc_cur_height=$(${1} relay btc_cur_height | jq ".curHeight")
    if [ "${btc_cur_height}" == "${base_height}" ]; then
        echo "height not correct"
        exit 1
    fi

    echo "=========== # get real BTC account ============="
    newacct="mdj"
    ${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet --wallet walletpassphrase password 100000000
    ${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet --wallet createnewaccount "${newacct}"
    btcrcv_addr=$(${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet --wallet getaccountaddress "${newacct}")
    echo "btcrcvaddr=${btcrcv_addr}"

    echo "=========== # get real BTY buy account ============="
    real_buy_addr=$(${1} account list | jq -r '.wallets[] | select(.label=="node award") | .acc.addr')
    echo "realbuyaddr=${real_buy_addr}"

    echo "=========== # transfer to relay ============="
    hash=$(${1} send coins transfer -a 1000 -t 1rhRgzbz264eyJu7Ac63wepsm9TsEpwXM -n "transfer to relay" -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
    echo "${hash}"
    hash=$(${1} send coins transfer -a 1000 -t 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt -n "transfer to accept addr" -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
    echo "${hash}"
    hash=$(${1} send coins transfer -a 200 -t "${real_buy_addr}" -n "transfer to accept addr" -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
    echo "${hash}"

    block_wait "${1}" 1
    before=$(${1} account balance -a 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv -e relay | jq -r ".balance")
    if [ "${before}" == "0.0000" ]; then
        echo "wrong relay addr balance, should not be zero"
        exit 1
    fi
    before=$(${1} account balance -a 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt -e coins | jq -r ".balance")
    if [ "${before}" == "0.0000" ]; then
        echo "wrong accept addr balance, should not be zero"
        exit 1
    fi
    before=$(${1} account balance -a "${real_buy_addr}" -e coins | jq -r ".balance")
    if [ "${before}" == "0.0000" ]; then
        echo "wrong real accept addr balance, should not be zero"
        exit 1
    fi

    echo "=========== # create buy order ============="
    buy_hash=$(${1} send relay create -m 2.99 -o 0 -c BTC -a 1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT -f 0.02 -b 200 -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
    echo "${buy_hash}"
    echo "=========== # create sell order ============="
    sell_hash=$(${1} send relay create -m 2.99 -o 1 -c BTC -a 2Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT -f 0.02 -b 200 -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
    echo "${sell_hash}"
    echo "=========== # create real buy order ============="
    realbuy_hash=$(${1} send relay create -m 10 -o 0 -c BTC -a "${btcrcv_addr}" -f 0.02 -b 200 -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
    echo "${realbuy_hash}"
    echo "=========== # transfer to relay ============="
    hash=$(${1} send coins transfer -a 300 -t 1rhRgzbz264eyJu7Ac63wepsm9TsEpwXM -n "send to relay" -k 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt)
    echo "${hash}"
    hash=$(${1} send coins transfer -a 100 -t 1rhRgzbz264eyJu7Ac63wepsm9TsEpwXM -n "send to relay" -k "${real_buy_addr}")
    echo "${hash}"

    block_wait "${1}" 1

    coinaddr=$(${1} tx query -s "${buy_hash}" | jq -r ".receipt.logs[2].log.coinAddr")
    if [ "${coinaddr}" != "1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT" ]; then
        echo "wrong create order to coinaddr"
        exit 1
    fi
    buy_id=$(${1} tx query -s "${buy_hash}" | jq -r ".receipt.logs[2].log.orderId")
    if [ -z "${buy_id}" ]; then
        echo "wrong buy id"
        exit 1
    fi
    oper=$(${1} tx query -s "${buy_hash}" | jq -r ".receipt.logs[2].log.coinOperation")
    if [ "${oper}" != "buy" ]; then
        echo "wrong buy operation"
        exit 1
    fi

    status=$(${1} tx query -s "${sell_hash}" | jq -r ".receipt.logs[2].log.curStatus")
    if [ "${status}" != "pending" ]; then
        echo "wrong create sell order status"
        exit 1
    fi
    sell_id=$(${1} tx query -s "${sell_hash}" | jq -r ".receipt.logs[2].log.orderId")
    if [ -z "${sell_id}" ]; then
        echo "wrong sell id"
        exit 1
    fi
    oper=$(${1} tx query -s "${sell_hash}" | jq -r ".receipt.logs[2].log.coinOperation")
    if [ "${oper}" != "sell" ]; then
        echo "wrong sell operation"
        exit 1
    fi
    realbuy_id=$(${1} tx query -s "${realbuy_hash}" | jq -r ".receipt.logs[2].log.orderId")
    if [ -z "${realbuy_id}" ]; then
        echo "wrong realbuy_id "
        exit 1
    fi
    before=$(${1} account balance -a 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt -e relay | jq -r ".balance")
    if [ "${before}" == "0.0000" ]; then
        echo "wrong relay balance, should not be zero"
        exit 1
    fi
    before=$(${1} account balance -a "${real_buy_addr}" -e relay | jq -r ".balance")
    if [ "${before}" != "100.0000" ]; then
        echo "wrong relay real buy balance, should be 100"
        exit 1
    fi

    id=$(${1} relay status -s 1 | jq -sr '.[] | select(.coinoperation=="buy")| select(.coinaddr=="1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT") |.orderid')
    if [ "${id}" != "${buy_id}" ]; then
        echo "wrong relay status buy order id"
        exit 1
    fi
    id=$(${1} relay status -s 1 | jq -sr '.[] | select(.coinoperation=="buy")| select(.coinamount=="10.0000") |.orderid')
    if [ "${id}" != "${realbuy_id}" ]; then
        echo "wrong relay status real buy order id"
        exit 1
    fi

    id=$(${1} relay status -s 1 | jq -sr '.[] | select(.coinoperation=="sell")|.orderid')
    if [ "${id}" != "${sell_id}" ]; then
        echo "wrong relay status sell order id"
        exit 1
    fi

    echo "=========== # accept buy order ============="
    buy_hash=$(${1} send relay accept -f 0.001 -o "${buy_id}" -a 1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT -k 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt)
    echo "${buy_hash}"
    echo "=========== # accept real buy order ============="
    realbuy_hash=$(${1} send relay accept -f 0.001 -o "${realbuy_id}" -a 1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT -k "${real_buy_addr}")
    echo "${realbuy_hash}"
    echo "=========== # accept sell order ============="
    sell_hash=$(${1} send relay accept -f 0.001 -o "${sell_id}" -a 1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT -k 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt)
    echo "${sell_hash}"
    block_wait "${1}" 1
    frozen=$(${1} tx query -s "${buy_hash}" | jq -r ".receipt.logs[1].log.current.frozen")
    if [ "${frozen}" != "100.0000" ]; then
        echo "wrong buy frozen account, should be 100"
        exit 1
    fi

    id=$(${1} relay status -s 2 | jq -sr '.[] | select(.coinoperation=="buy") | select(.coinaddr=="1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT") |.orderid')
    if [ "${id}" != "${buy_id}" ]; then
        echo "wrong relay status buy order id"
        exit 1
    fi
    id=$(${1} relay status -s 2 | jq -sr '.[] | select(.coinoperation=="buy")| select(.coinamount=="10.0000")|.orderid')
    if [ "${id}" != "${realbuy_id}" ]; then
        echo "wrong relay status real buy order id"
        exit 1
    fi

    id=$(${1} relay status -s 2 | jq -sr '.[] | select(.coinoperation=="sell")|.orderid')
    if [ "${id}" != "${sell_id}" ]; then
        echo "wrong relay status sell order id"
        exit 1
    fi

    echo "=========== # btc generate 80 blocks ============="
    ## for unlock order's 36 blocks waiting
    current=$(${1} relay btc_cur_height | jq ".curHeight")
    ${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet generate 80
    wait_btc_height "${1}" $((current + 80))

    echo "=========== # btc tx to real order ============="
    btc_tx_hash=$(${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet --wallet sendfrom default "${btcrcv_addr}" 10)
    echo "${btc_tx_hash}"
    ${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet generate 4
    blockhash=$(${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet --wallet gettransaction "${btc_tx_hash}" | jq -r ".blockhash")
    blockheight=$(${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet --wallet getblockheader "${blockhash}" | jq -r ".height")
    echo "blcockheight=${blockheight}"
    ${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet --wallet getreceivedbyaddress "${btcrcv_addr}"

    wait_btc_height "${1}" $((current + 80 + 4))

    echo "=========== # unlock buy order ==========="
    acceptHeight=$(${1} tx query -s "${buy_hash}" | jq -r ".receipt.logs[2].log.coinHeight")
    if [ "${acceptHeight}" -lt "${btc_cur_height}" ]; then
        echo "accept height less previous height"
        exit 1
    fi

    expect=$((acceptHeight + 72))
    wait_btc_height "${1}" $((acceptHeight + 72))

    revoke_hash=$(${1} send relay revoke -a 0 -t 1 -f 0.01 -i "${buy_id}" -k 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt)
    echo "${revoke_hash}"
    echo "=========== # confirm real buy order ============="
    confirm_hash=$(${1} send relay confirm -f 0.001 -t "${btc_tx_hash}" -o "${realbuy_id}" -k "${real_buy_addr}")
    echo "${confirm_hash}"
    echo "=========== # confirm sell order ============="
    confirm_hash=$(${1} send relay confirm -f 0.001 -t 6359f0868171b1d194cbee1af2f16ea598ae8fad666d9b012c8ed2b79a236ec4 -o "${sell_id}" -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
    echo "${confirm_hash}"

    block_wait "${1}" 1

    id=$(${1} relay status -s 1 | jq -sr '.[] | select(.coinoperation=="buy")|.orderid')
    if [ "${id}" != "${buy_id}" ]; then
        echo "wrong relay pending status unlock buy order id"
        exit 1
    fi

    id=$(${1} relay status -s 3 | jq -sr '.[] | select(.coinoperation=="buy")|.orderid')
    if [ "${id}" != "${realbuy_id}" ]; then
        echo "wrong relay status confirming real buy order id"
        exit 1
    fi
    id=$(${1} relay status -s 3 | jq -sr '.[] | select(.coinoperation=="sell")|.orderid')
    if [ "${id}" != "${sell_id}" ]; then
        echo "wrong relay status confirming sell order id"
        exit 1
    fi

    echo "=========== # btc generate 300 blocks  ==="
    current=$(${1} relay btc_cur_height | jq ".curHeight")
    ${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet generate 300
    wait_btc_height "${1}" $((current + 300))

    echo "=========== # unlock sell order ==="
    confirmHeight=$(${1} tx query -s "${confirm_hash}" | jq -r ".receipt.logs[1].log.coinHeight")
    if [ "${confirmHeight}" -lt "${btc_cur_height}" ]; then
        echo "wrong confirm height"
        exit 1
    fi

    wait_btc_height "${1}" $((confirmHeight + 288))

    revoke_hash=$(${1} send relay revoke -a 0 -t 0 -f 0.01 -i "${sell_id}" -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
    echo "${revoke_hash}"
    echo "=========== # test cancel create order ==="
    cancel_hash=$(${1} send relay create -m 2.99 -o 0 -c BTC -a 1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT -f 0.02 -b 200 -k 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt)
    echo "${cancel_hash}"

    block_wait "${1}" 1

    cancel_id=$(${1} tx query -s "${cancel_hash}" | jq -r ".receipt.logs[2].log.orderId")
    if [ -z "${cancel_id}" ]; then
        echo "wrong buy id"
        exit 1
    fi
    id=$(${1} relay status -s 1 | jq -sr '.[] | select(.coinoperation=="sell")| select(.address=="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv") | .orderid')
    if [ "${id}" != "${sell_id}" ]; then
        echo "wrong relay revoke order id "
        exit 1
    fi

    echo "=========== # wait relayd verify order ======="
    ## for relayd verify tick 5s
    block_wait "${1}" 3

    echo "=========== # check finish order ============="
    count=30
    while true; do
        id=$(${1} relay status -s 4 | jq -sr '.[] | select(.coinoperation=="buy")|.orderid')
        if [ "${id}" == "${realbuy_id}" ]; then
            break
        fi
        block_wait "${1}" 1
        count=$((count - 1))
        if [ $count -le 0 ]; then
            echo "wrong relay status finish real buy order id"
            exit 1
        fi
    done

    before=$(${1} account balance -a "${real_buy_addr}" -e relay | jq -r ".balance")
    if [ "${before}" != "300.0000" ]; then
        echo "wrong relay real buy addr balance, should be 300"
        exit 1
    fi

    echo "=========== # cancel order ============="
    hash=$(${1} send relay revoke -a 1 -t 0 -f 0.01 -i "${cancel_id}" -k 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt)
    echo "${hash}"
    block_wait "${1}" 1

    status=$(${1} relay status -s 5 | jq -r ".status")
    if [ "${status}" != "canceled" ]; then
        echo "wrong relay order pending status"
        exit 1
    fi
    id=$(${1} relay status -s 5 | jq -sr '.[] | select(.coinoperation=="buy")|.orderid')
    if [ "${id}" != "${cancel_id}" ]; then
        echo "wrong relay status cancel order id"
        exit 1
    fi

}

function para() {
    echo "=========== # para chain test ============="
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

function main() {
    echo "==========================================main begin========================================================"
    init
    para_set_env
    start
    para_init
    para_transfer
    sync
    transfer

    para "${PARA_CLI}"
    relay "${CLI}"
    # TODO other work!!!

    check_docker_container
    echo "==========================================main end========================================================="
}

# run script
main
