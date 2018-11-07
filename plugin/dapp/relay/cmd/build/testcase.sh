#!/usr/bin/env bash

BTCD="${1}_btcd_1"

RELAYD="${1}_relayd_1"

BTC_CTL="docker exec ${BTCD} btcctl"

# shellcheck disable=SC2086,2154
function relay_init() {
    # relayd
    sed -i $sedfix 's/^btcdOrWeb.*/btcdOrWeb = 0/g' relayd.toml
    sed -i $sedfix 's/^Tick33.*/Tick33 = 5/g' relayd.toml
    sed -i $sedfix 's/^TickBTC.*/TickBTC = 5/g' relayd.toml
    sed -i $sedfix 's/^pprof.*/pprof = false/g' relayd.toml
    sed -i $sedfix 's/^watch.*/watch = false/g' relayd.toml

}

function run_relayd_with_btcd() {
    echo "============== run_relayd_with_btcd ==============================="
    docker cp "${BTCD}:/root/rpc.cert" ./rpc.cert
    docker cp ./rpc.cert "${RELAYD}:/root/rpc.cert"
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

function relay_config() {
    wait_btcd_up
    run_relayd_with_btcd
    ping_btcd

}

#some times btcwallet bin 18554 server port fail in btcd docker, restart btcd will be ok
# [WRN] BTCW: Can't listen on [::1]:18554: listen tcp6 [::1]:18554: bind: cannot assign requested address
# shellcheck disable=SC2068
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
        mod=$((count % 4))
        if [ $mod == 0 ]; then
            docker-compose down
            sleep 5
            docker-compose up --build -d
            sleep 60
            continue
        fi
        #btcd restart need wait 30s
        sleep 30
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

function relay_test() {
    echo "================relayd test========================"
    block_wait "${1}" 2

    times=100
    while true; do
        ${1} relay btc_cur_height
        base_height=$(${1} relay btc_cur_height | jq ".baseHeight")
        btc_cur_height=$(${1} relay btc_cur_height | jq ".curHeight")
        if [ "${btc_cur_height}" == "${base_height}" ]; then
            echo "height not correct, wait 2 block.."
            block_wait "${1}" 2
            times=$((times - 1))
            if [ $times -le 0 ]; then
                echo "height not correct failed"
                exit 1
            fi
        else
            echo "btc height correct, pass"
            break
        fi
    done

    echo "=========== # get real BTC account ============="
    newacct="relay"
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
    if [ "${frozen}" != "10000000000" ]; then
        echo "wrong buy frozen account, should be 100"
        ${1} tx query -s "${buy_hash}"
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
    echo "${revoke_hash}"
    ${1} tx query -s "${revoke_hash}"

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
    echo "${revoke_hash}"
    ${1} tx query -s "${revoke_hash}"

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
    echo "${hash}"
    ${1} tx query -s "${hash}"

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

function relay() {
    if [ "${2}" == "init" ]; then
        relay_init
    elif [ "${2}" == "config" ]; then
        relay_config
    elif [ "${2}" == "test" ]; then
        relay_test "${1}"
    fi

}
