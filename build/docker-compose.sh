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
CLI=" docker exec ${1}_chain33_1 /root/chain33-cli"
NODE3="${1}_chain33_1"
CLI="docker exec ${NODE3} /root/chain33-cli"

NODE2="${1}_chain32_1"
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
    sed -i $sedfix 's/^seeds=.*/seeds=["chain33:13802","chain32:13802","chain31:13802"]/g' chain33.toml
    sed -i $sedfix 's/^enable=.*/enable=true/g' chain33.toml
    sed -i $sedfix 's/^isSeed=.*/isSeed=true/g' chain33.toml
    sed -i $sedfix 's/^innerSeedEnable=.*/innerSeedEnable=false/g' chain33.toml
    sed -i $sedfix 's/^useGithub=.*/useGithub=false/g' chain33.toml

    # rpc
    sed -i $sedfix 's/^jrpcBindAddr=.*/jrpcBindAddr="0.0.0.0:8801"/g' chain33.toml
    sed -i $sedfix 's/^grpcBindAddr=.*/grpcBindAddr="0.0.0.0:8802"/g' chain33.toml
    sed -i $sedfix 's/^whitlist=.*/whitlist=["localhost","127.0.0.1","0.0.0.0"]/g' chain33.toml

    # wallet
    sed -i $sedfix 's/^minerdisable=.*/minerdisable=false/g' chain33.toml

    # docker-compose ps
    docker-compose ps

	# remove exsit container
	docker-compose down

	# create and run docker-compose container
	docker-compose up --build -d


    local SLEEP=60
    echo "=========== sleep ${SLEEP}s ============="
    sleep ${SLEEP}

	# docker-compose ps
	docker-compose ps

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

	sleep 2

	echo "=========== # unlock wallet ============="
	result=$(${CLI} wallet unlock -p 1314 -t 0 | jq ".isok")
	if [ "${result}" = "false" ]; then
		exit 1
	fi

	sleep 2

	echo "=========== # import private key returnAddr ============="
	result=$(${CLI} account import_key -k CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944 -l returnAddr | jq ".label")
	echo "${result}"
	if [ -z "${result}" ]; then
		exit 1
	fi

	sleep 2

	echo "=========== # import private key mining ============="
	result=$(${CLI} account import_key -k 4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01 -l minerAddr | jq ".label")
	echo "${result}"
	if [ -z "${result}" ]; then
		exit 1
	fi

	sleep 2
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
	# ${CLI} mempool last_txs
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


function sync_status(){
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

function transfer(){
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
	sed -i 's/ForkV7AddRelay.*/ForkV7AddRelay = 2/g' ../types/relay.go
}

function relay_after() {
	git checkout ../types/relay.go
}

function relay() {
	echo "================relayd========================"
	while true; do
		${1} block last_header
		result=$(${1} block last_header | jq ".height")
		if [ "${result}" -gt 10 ]; then
			break
		fi
		sleep 1
	done

	${1} relay btc_cur_height
	base_height=$(${1} relay btc_cur_height | jq ".BaseHeight")
	current_height=$(${1} relay btc_cur_height | jq ".CurHeight")
	if [ "${current_height}" == "${base_height}" ]; then
	    echo "height not correct"
	    exit 1
	fi

	echo "=========== # transfer to relay ============="
	hash=$(${1} send bty transfer -a 1000 -t 1rhRgzbz264eyJu7Ac63wepsm9TsEpwXM -n "transfer to relay" -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
	echo "${hash}"
	hash=$(${1} send bty transfer -a 1000 -t 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt -n "transfer to accept addr" -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
	echo "${hash}"
	sleep 25
	before=$(${CLI} account balance -a 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv -e relay | jq ".balance")
	before=$(echo "$before" | bc)
	if [ "${before}" == 0.0000 ]; then
		echo "wrong relay addr balance, should not be zero"
		exit 1
	fi
	before=$(${CLI} account balance -a 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt -e coins | jq ".balance")
	before=$(echo "$before" | bc)
	if [ "${before}" == 0.0000 ]; then
		echo "wrong accept addr balance, should not be zero"
		exit 1
	fi

	echo "=========== # create buy order ============="
	buy_hash=$(${1} send relay create -o 0 -c BTC -a 1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT -m 2.99 -f 0.02 -b 200 -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
	echo "${buy_hash}"
    echo "=========== # create sell order ============="
    sell_hash=$(${1} send relay create -o 1 -c BTC -a 1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT -m 2.99 -f 0.02 -b 200 -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
    echo "${sell_hash}"
	echo "=========== # transfer to relay ============="
	hash=$(${1} send bty transfer -a 300 -t 1rhRgzbz264eyJu7Ac63wepsm9TsEpwXM -n "send to relay" -k 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt)
	echo "${hash}"

	sleep 25
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
	before=$(${CLI} account balance -a 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt -e relay | jq ".balance")
	before=$(echo "$before" | bc)
	if [ "${before}" == 0.0000 ]; then
		echo "wrong relay balance, should not be zero"
		exit 1
	fi

	id=$(${CLI} relay status -s 1 | jq -sr '.[] | select(.coinoperation=="buy")|.orderid')
	if [ "${id}" != "${buy_id}" ]; then
	    echo "wrong relay status buy order id"
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
    echo "=========== # accept sell order ============="
    sell_hash=$(${1} send relay accept -f 0.001 -o "${sell_id}" -a 1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT -k 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt)
    echo "${sell_hash}"
	sleep 25

	id=$(${CLI} relay status -s 2 | jq -sr '.[] | select(.coinoperation=="buy")|.orderid')
	if [ "${id}" != "${buy_id}" ]; then
	    echo "wrong relay status buy order id"
	    exit 1
	fi
	id=$(${CLI} relay status -s 2 | jq -sr '.[] | select(.coinoperation=="sell")|.orderid')
	if [ "${id}" != "${sell_id}" ]; then
	    echo "wrong relay status sell order id"
	    exit 1
	fi

    echo "=========== # unlock buy order ==========="
	acceptHeight=$(${CLI} tx query -s "${buy_hash}" | jq -r ".receipt.logs[1].log.coinHeight")
	echo "${acceptHeight}"
	if [ "${acceptHeight}" -lt "${current_height}" ]; then
		echo "accept height less previous height"
		exit 1
	fi
	expectHeight=$(echo "${acceptHeight}+36" | bc)
	while true; do
		current_height=$(${1} relay btc_cur_height | jq ".CurHeight")
		if [ "${current_height}" -gt "${expectHeight}" ]; then
			break
		fi

	done
	revoke_hash=$(${1} send relay revoke -a 0 -t 1 -f 0.01 -i "${buy_id}" -k 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt)
	echo "${revoke_hash}"
    echo "=========== # confirm sell order ============="
    confirm_hash=$(${1} send relay confirm -f 0.001 -t 6359f0868171b1d194cbee1af2f16ea598ae8fad666d9b012c8ed2b79a236ec4 -o "${sell_id}" -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
    echo "${confirm_hash}"
	sleep 25

	id=$(${CLI} relay status -s 1 | jq -sr '.[] | select(.coinoperation=="buy")|.orderid')
	if [ "${id}" != "${buy_id}" ]; then
	    echo "wrong relay status unlock buy order id"
	    exit 1
	fi

	id=$(${CLI} relay status -s 3 | jq -sr '.[] | select(.coinoperation=="sell")|.orderid')
	if [ "${id}" != "${sell_id}" ]; then
	    echo "wrong relay status unlock buy order id"
	    exit 1
	fi

    echo "=========== # unlock sell order ==="
    confirmHeight=$(${CLI} tx query -s "${confirm_hash}" | jq -r ".receipt.logs[1].log.coinHeight")
    echo "${confirmHeight}"
    if [ "${confirmHeight}" -lt "${current_height}" ]; then
        echo "wrong confirm height"
        exit 1
    fi

	expectHeight=$(echo "${confirmHeight}+144" | bc)
	while true; do
		current_height=$(${1} relay btc_cur_height | jq ".CurHeight")
		if [ "${current_height}" -gt "${expectHeight}" ]; then
			break
		fi

	done
	revoke_hash=$(${1} send relay revoke -a 0 -t 0 -f 0.01 -i "${sell_id}" -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv)
	echo "${revoke_hash}"
    echo "=========== # test cancel create order ==="
	cancel_hash=$(${1} send relay create -o 0 -c BTC -a 1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT -m 2.99 -f 0.02 -b 200 -k 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt)
	echo "${cancel_hash}"
	sleep 25

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

	echo "=========== # cancel order ============="
	hash=$(${1} send relay revoke -a 1 -t 0 -f 0.01 -i "${cancel_id}" -k 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt)
	echo "${hash}"
	sleep 25

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

    ## time limited, the following cases could not be added to CI currently, except add test macro to lock time
    #echo "=========== # test finish order ==="

}

function main() {
	echo "==========================================main begin========================================================"
	init
	sync
	transfer
	relay "${CLI}"

	# TODO other work!!!
	echo "==========================================main end========================================================="
}

# run script
main
