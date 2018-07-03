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
CLI2="docker exec ${NODE2} /root/chain33-cli"

NODE1="${1}_chain31_1"
CLI3="docker exec ${NODE1} /root/chain33-cli"

NODE4="${1}_chain30_1"
CLI4="docker exec ${NODE4} /root/chain33-cli"

NODE5="${1}_chain29_1"
CLI5="docker exec ${NODE5} /root/chain33-cli"

NODE6="${1}_chain28_1"
CLI6="docker exec ${NODE6} /root/chain33-cli"
BTCD="${1}_btcd_1"
BTC_CTL="docker exec ${BTCD} btcctl"

RELAYD="${1}_relayd_1"

containers=("${NODE1}" "${NODE2}" "${NODE3}" "${NODE4}" "${NODE5}" "${NODE6}" "${BTCD}" "${RELAYD}")
forkContainers=("${CLI3}" "${CLI2}" "${CLI}" "${CLI4}" "${CLI5}" "${CLI6}")

sedfix=""
if [ "$(uname)" == "Darwin" ]; then
    sedfix=".bak"
fi

. ./fork-test.sh

function init() {
    # update test environment
    sed -i $sedfix 's/^Title.*/Title="local"/g' chain33.toml
    sed -i $sedfix 's/^TestNet=.*/TestNet=true/g' chain33.toml

    # p2p
    sed -i $sedfix 's/^seeds=.*/seeds=["chain33:13802","chain32:13802","chain31:13802"]/g' chain33.toml
    sed -i $sedfix 's/^enable=.*/enable=true/g' chain33.toml
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

    # docker-compose ps
    docker-compose ps

    # remove exsit container
    docker-compose down

    # create and run docker-compose container
    docker-compose up --build -d

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


    ## 2nd mining
    echo "=========== # save seed to wallet ============="
    result=$(${CLI4} seed save -p 1314 -s "tortoise main civil member grace happy century convince father cage beach hip maid merry rib" | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "save seed to wallet error seed, result: ${result}"
        exit 1
    fi

    sleep 1

    echo "=========== # unlock wallet ============="
    result=$(${CLI4} wallet unlock -p 1314 -t 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        exit 1
    fi

    sleep 1

    echo "=========== # import private key returnAddr ============="
    result=$(${CLI4} account import_key -k 2AFF1981291355322C7A6308D46A9C9BA311AA21D94F36B43FC6A6021A1334CF -l returnAddr | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi

    sleep 1

    echo "=========== # import private key mining ============="
    result=$(${CLI4} account import_key -k 2116459C0EC8ED01AA0EEAE35CAC5C96F94473F7816F114873291217303F6989 -l minerAddr | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi

    sleep 1
    echo "=========== # close auto mining ============="
    result=$(${CLI4} wallet auto_mine -f 0 | jq ".isok")
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
    echo "=========== stop  ${NODE2} node========== "
    docker stop "${NODE2}"
    sleep 20

    echo "=========== start ${NODE2} node========== "
    docker start "${NODE2}"

    for i in $(seq 20); do
        sleep 1
        ${CLI2} net is_sync
        ${CLI2} block last_header
    done

    sync_status "${CLI2}"
}

function transfer() {
    echo "=========== # transfer ============="
    hashes=()
    for ((i = 0; i < 10; i++)); do
        hash=$(${CLI} send bty transfer -a 1 -n test -t 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt -k 4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01)
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
    hash=$(${CLI} send bty transfer -a 2 -n deposit -t 1wvmD6RNHzwhY4eN75WnM6JcaAvNQ4nHx -k CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944)
    echo "${hash}"
    block_wait "${CLI}" 1
    before=$(${CLI} account balance -a 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt -e retrieve | jq -r ".balance")
    if [ "${before}" == "0.0000" ]; then
        echo "wrong ticket balance, should not be zero"
        exit 1
    fi

    hash=$(${CLI} send bty withdraw -a 1 -n withdraw -e retrieve -k CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944)
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

    echo "=========== # get real btc account ============="
    newacct="mdj"
    ${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet --wallet walletpassphrase password 100000000
    ${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet --wallet createnewaccount "${newacct}"
    btcrcv_addr=$(${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet --wallet getaccountaddress "${newacct}")
    echo "btcrcvaddr=${btcrcv_addr}"

    echo "=========== # get real btc account ============="
    real_buy_addr=$(${1} account list | jq -r '.wallets[] | select(.label=="node award") | .acc.addr')
    echo "realbuyaddr=${real_buy_addr}"

    echo "=========== # transfer to relay ============="
    hash=$(${1} send bty transfer -a 1000 -t 1rhRgzbz264eyJu7Ac63wepsm9TsEpwXM -n "transfer to relay" -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
    echo "${hash}"
    hash=$(${1} send bty transfer -a 1000 -t 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt -n "transfer to accept addr" -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
    echo "${hash}"
    hash=$(${1} send bty transfer -a 100 -t "${real_buy_addr}" -n "transfer to accept addr" -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
    echo "${hash}"

    block_wait "${1}" 1
    before=$(${CLI} account balance -a 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv -e relay | jq -r ".balance")
    if [ "${before}" == "0.0000" ]; then
        echo "wrong relay addr balance, should not be zero"
        exit 1
    fi
    before=$(${CLI} account balance -a 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt -e coins | jq -r ".balance")
    if [ "${before}" == "0.0000" ]; then
        echo "wrong accept addr balance, should not be zero"
        exit 1
    fi
    before=$(${CLI} account balance -a "${real_buy_addr}" -e coins | jq -r ".balance")
    if [ "${before}" == "0.0000" ]; then
        echo "wrong real accept addr balance, should not be zero"
        exit 1
    fi

    echo "=========== # create buy order ============="
    buy_hash=$(${1} send relay create -o 0 -c BTC -a 1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT -m 2.99 -f 0.02 -b 200 -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
    echo "${buy_hash}"
    echo "=========== # create sell order ============="
    sell_hash=$(${1} send relay create -o 1 -c BTC -a 2Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT -m 2.99 -f 0.02 -b 200 -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
    echo "${sell_hash}"
    echo "=========== # create real buy order ============="
    realbuy_hash=$(${1} send relay create -o 0 -c BTC -a "${btcrcv_addr}" -m 10 -f 0.02 -b 200 -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
    echo "${realbuy_hash}"
    echo "=========== # transfer to relay ============="
    hash=$(${1} send bty transfer -a 300 -t 1rhRgzbz264eyJu7Ac63wepsm9TsEpwXM -n "send to relay" -k 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt)
    echo "${hash}"

    block_wait "${1}" 1

    coinaddr=$(${CLI} tx query -s "${buy_hash}" | jq -r ".receipt.logs[2].log.coinAddr")
    if [ "${coinaddr}" != "1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT" ]; then
        echo "wrong create order to coinaddr"
        exit 1
    fi
    buy_id=$(${CLI} tx query -s "${buy_hash}" | jq -r ".receipt.logs[2].log.orderId")
    if [ -z "${buy_id}" ]; then
        echo "wrong buy id"
        exit 1
    fi
    oper=$(${CLI} tx query -s "${buy_hash}" | jq -r ".receipt.logs[2].log.coinOperation")
    if [ "${oper}" != "buy" ]; then
        echo "wrong buy operation"
        exit 1
    fi

    status=$(${CLI} tx query -s "${sell_hash}" | jq -r ".receipt.logs[1].log.curStatus")
    if [ "${status}" != "pending" ]; then
        echo "wrong create sell order status"
        exit 1
    fi
    sell_id=$(${CLI} tx query -s "${sell_hash}" | jq -r ".receipt.logs[1].log.orderId")
    if [ -z "${sell_id}" ]; then
        echo "wrong sell id"
        exit 1
    fi
    oper=$(${CLI} tx query -s "${sell_hash}" | jq -r ".receipt.logs[1].log.coinOperation")
    if [ "${oper}" != "sell" ]; then
        echo "wrong sell operation"
        exit 1
    fi
    realbuy_id=$(${CLI} tx query -s "${realbuy_hash}" | jq -r ".receipt.logs[2].log.orderId")
    if [ -z "${realbuy_id}" ]; then
        echo "wrong realbuy_id "
        exit 1
    fi
    before=$(${CLI} account balance -a 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt -e relay | jq -r ".balance")
    if [ "${before}" == "0.0000" ]; then
        echo "wrong relay balance, should not be zero"
        exit 1
    fi

    id=$(${CLI} relay status -s 1 | jq -sr '.[] | select(.coinoperation=="buy")| select(.coinaddr=="1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT") |.orderid')
    if [ "${id}" != "${buy_id}" ]; then
        echo "wrong relay status buy order id"
        exit 1
    fi
    id=$(${CLI} relay status -s 1 | jq -sr '.[] | select(.coinoperation=="buy")| select(.coinamount=="10.0000") |.orderid')
    if [ "${id}" != "${realbuy_id}" ]; then
        echo "wrong relay status real buy order id"
        exit 1
    fi

    id=$(${CLI} relay status -s 1 | jq -sr '.[] | select(.coinoperation=="sell")|.orderid')
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

    id=$(${CLI} relay status -s 2 | jq -sr '.[] | select(.coinoperation=="buy") | select(.coinaddr=="1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT") |.orderid')
    if [ "${id}" != "${buy_id}" ]; then
        echo "wrong relay status buy order id"
        exit 1
    fi
    id=$(${CLI} relay status -s 2 | jq -sr '.[] | select(.coinoperation=="buy")| select(.coinamount=="10.0000")|.orderid')
    if [ "${id}" != "${realbuy_id}" ]; then
        echo "wrong relay status real buy order id"
        exit 1
    fi

    id=$(${CLI} relay status -s 2 | jq -sr '.[] | select(.coinoperation=="sell")|.orderid')
    if [ "${id}" != "${sell_id}" ]; then
        echo "wrong relay status sell order id"
        exit 1
    fi

    echo "=========== # btc generate 40 blocks ============="
    ## for unlock order's 36 blocks waiting
    current=$(${1} relay btc_cur_height | jq ".curHeight")
    ${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet generate 40
    wait_btc_height "${1}" $((current + 40))

    echo "=========== # btc tx to real order ============="
    btc_tx_hash=$(${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet --wallet sendfrom default "${btcrcv_addr}" 10)
    echo "${btc_tx_hash}"
    ${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet generate 4
    blockhash=$(${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet --wallet gettransaction "${btc_tx_hash}" | jq -r ".blockhash")
    blockheight=$(${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet --wallet getblockheader "${blockhash}" | jq -r ".height")
    echo "blcockheight=${blockheight}"
    ${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet --wallet getreceivedbyaddress "${btcrcv_addr}"

    wait_btc_height "${1}" $((current + 40 + 4))

    echo "=========== # unlock buy order ==========="
    acceptHeight=$(${CLI} tx query -s "${buy_hash}" | jq -r ".receipt.logs[1].log.coinHeight")

    if [ "${acceptHeight}" -lt "${btc_cur_height}" ]; then
        echo "accept height less previous height"
        exit 1
    fi

    expect=$((acceptHeight + 36))
    wait_btc_height "${1}" $((acceptHeight + 36))

    revoke_hash=$(${1} send relay revoke -a 0 -t 1 -f 0.01 -i "${buy_id}" -k 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt)
    echo "${revoke_hash}"
    echo "=========== # confirm real buy order ============="
    confirm_hash=$(${1} send relay confirm -f 0.001 -t "${btc_tx_hash}" -o "${realbuy_id}" -k "${real_buy_addr}")
    echo "${confirm_hash}"
    echo "=========== # confirm sell order ============="
    confirm_hash=$(${1} send relay confirm -f 0.001 -t 6359f0868171b1d194cbee1af2f16ea598ae8fad666d9b012c8ed2b79a236ec4 -o "${sell_id}" -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
    echo "${confirm_hash}"

    block_wait "${1}" 1

    id=$(${CLI} relay status -s 1 | jq -sr '.[] | select(.coinoperation=="buy")|.orderid')
    if [ "${id}" != "${buy_id}" ]; then
        echo "wrong relay pending status unlock buy order id"
        exit 1
    fi

    id=$(${CLI} relay status -s 3 | jq -sr '.[] | select(.coinoperation=="buy")|.orderid')
    if [ "${id}" != "${realbuy_id}" ]; then
        echo "wrong relay status confirming real buy order id"
        exit 1
    fi
    id=$(${CLI} relay status -s 3 | jq -sr '.[] | select(.coinoperation=="sell")|.orderid')
    if [ "${id}" != "${sell_id}" ]; then
        echo "wrong relay status confirming sell order id"
        exit 1
    fi

    echo "=========== # btc generate 200 blocks  ==="
    current=$(${1} relay btc_cur_height | jq ".curHeight")
    ${BTC_CTL} --rpcuser=root --rpcpass=1314 --simnet generate 200
    wait_btc_height "${1}" $((current + 200))

    echo "=========== # unlock sell order ==="
    confirmHeight=$(${CLI} tx query -s "${confirm_hash}" | jq -r ".receipt.logs[1].log.coinHeight")
    if [ "${confirmHeight}" -lt "${btc_cur_height}" ]; then
        echo "wrong confirm height"
        exit 1
    fi

    wait_btc_height "${1}" $((confirmHeight + 144))

    revoke_hash=$(${1} send relay revoke -a 0 -t 0 -f 0.01 -i "${sell_id}" -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
    echo "${revoke_hash}"
    echo "=========== # test cancel create order ==="
    cancel_hash=$(${1} send relay create -o 0 -c BTC -a 1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT -m 2.99 -f 0.02 -b 200 -k 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt)
    echo "${cancel_hash}"

    block_wait "${1}" 1

    cancel_id=$(${CLI} tx query -s "${cancel_hash}" | jq -r ".receipt.logs[2].log.orderId")
    if [ -z "${cancel_id}" ]; then
        echo "wrong buy id"
        exit 1
    fi
    id=$(${CLI} relay status -s 1 | jq -sr '.[] | select(.coinoperation=="sell")| select(.address=="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv") | .orderid')
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
        id=$(${CLI} relay status -s 4 | jq -sr '.[] | select(.coinoperation=="buy")|.orderid')
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

    before=$(${CLI} account balance -a "${real_buy_addr}" -e relay | jq -r ".balance")
    if [ "${before}" != "200.0000" ]; then
        echo "wrong relay real buy addr balance, should be 200"
        exit 1
    fi

    echo "=========== # cancel order ============="
    hash=$(${1} send relay revoke -a 1 -t 0 -f 0.01 -i "${cancel_id}" -k 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt)
    echo "${hash}"
    block_wait "${1}" 1

    status=$(${CLI} relay status -s 5 | jq -r ".status")
    if [ "${status}" != "canceled" ]; then
        echo "wrong relay order pending status"
        exit 1
    fi
    id=$(${CLI} relay status -s 5 | jq -sr '.[] | select(.coinoperation=="buy")|.orderid')
    if [ "${id}" != "${cancel_id}" ]; then
        echo "wrong relay status cancel order id"
        exit 1
    fi

}

function main() {
    echo "==========================================main begin========================================================"
    init
    sync
    transfer
    relay "${CLI}"
    # TODO other work!!!
    # 构造分叉测试
    optDockerfun

    check_docker_container
    echo "==========================================main end========================================================="
}

# run script
main
