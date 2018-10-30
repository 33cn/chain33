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

PWD=$(cd "$(dirname "$0")" && pwd)
export PATH="$PWD:$PATH"

CLI="./chain33-cli"

sedfix=""
if [ "$(uname)" == "Darwin" ]; then
    sedfix=".bak"
fi

chain33Config="chain33.test.toml"
chain33BlockTime=2
function init() {
    # update test environment

    echo "# copy chain33 for solo test"
    cp ../../chain33 ./
    cp ../../chain33-cli ./
    cp ../../../cmd/chain33/chain33.test.toml ./

}

function config_chain33() {

    # shellcheck disable=SC2015
    echo "# config chain33 solo test"
    # update test environment
    sed -i $sedfix 's/^Title.*/Title="local"/g' ${chain33Config}
    # grep -q '^TestNet' ${chain33Config} && sed -i $sedfix 's/^TestNet.*/TestNet=true/' ${chain33Config} || sed -i '/^Title/a TestNet=true' ${chain33Config}

    if grep -q '^TestNet' ${chain33Config}; then
        sed -i $sedfix 's/^TestNet.*/TestNet=true/' ${chain33Config}
    else
        sed -i $sedfix '/^Title/a TestNet=true' ${chain33Config}
    fi

    #update fee
    sed -i $sedfix 's/Fee=.*/Fee=100000/' ${chain33Config}

    #update block time

    #update wallet store driver
    sed -i $sedfix '/^\[wallet\]/,/^\[wallet./ s/^driver.*/driver="leveldb"/' ${chain33Config}
}

autotestConfig="autotest.toml"
autotestTempConfig="autotest.temp.toml"
function config_autotest() {

    #delete all blank lines
    echo "# config autotest"
    sed -i $sedfix '/^\s*$/d' ${autotestConfig}

    if [[ $1 == "" ]] || [[ $1 == "all" ]]; then
        cp ${autotestConfig} ${autotestTempConfig}
    else
        #copy config before [
        sed -n '/^\[/!p;//q' ${autotestConfig} >${autotestTempConfig}

        #copy specific dapp cofig

        for dapp in "$@"; do
            {
                echo "[[TestCaseFile]]"
                echo "contract=\"$dapp\""
                echo "filename=\"$dapp.toml\""
            } >>${autotestTempConfig}

        done
    fi

    sed -i $sedfix 's/^checkSleepTime.*/checkSleepTime='${chain33BlockTime}'/' ${autotestTempConfig}
}

function start_chain33() {

    echo "# start solo chain33, make sure there is no chain33 instance running"
    rm -rf ../autotest/datadir ../autotest/logs ../autotest/grpc33.log
    ./chain33 -f chain33.test.toml >/dev/null 2>&1 &

    local SLEEP=5
    echo "=========== sleep ${SLEEP}s ============="
    sleep ${SLEEP}

    # query node run status
    ${CLI} block last_header

    echo "=========== # save seed to wallet ============="
    result=$(${CLI} seed save -p 1314 -s "tortoise main civil member grace happy century convince father cage beach hip maid merry rib" | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "save seed to wallet error seed, result: ${result}"
        exit 1
    fi

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

    echo "=========== # import test addr1 ============="
    result=$(${CLI} account import_key -k 0x88b2fb90411935872f0501dd13345aba19b5fac9b00eb0dddd7df977d4d5477e -l test_addr1 | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi

    echo "=========== # import test addr2 ============="
    result=$(${CLI} account import_key -k 0xa0c6f46de8d275ce21e935afa5363e9b8a087fe604e05f7a9eef1258dc781c3a -l test_addr2 | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi

    echo "=========== # import test addr3 ============="
    result=$(${CLI} account import_key -k 0x9d4f8ab11361be596468b265cb66946c87873d4a119713fd0c3d8302eae0a8e4 -l test_addr3 | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi

    echo "=========== #transfer to miner addr ============="
    hash=$(${CLI} send coins transfer -a 10000 -n test -t 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv -k CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944)

    sleep ${chain33BlockTime}
    txs=$(${CLI} tx query_hash -s "${hash}" | jq ".txs")
    if [ "${txs}" == "null" ]; then
        echo "transferTokenAdmin cannot find tx"
        exit 1
    fi

    echo "=========== #transfer to token amdin ============="
    hash=$(${CLI} send coins transfer -a 10 -n test -t 1Q8hGLfoGe63efeWa8fJ4Pnukhkngt6poK -k CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944)

    sleep ${chain33BlockTime}
    txs=$(${CLI} tx query_hash -s "${hash}" | jq ".txs")
    if [ "${txs}" == "null" ]; then
        echo "transferTokenAdmin cannot find tx"
        exit 1
    fi

    echo "=========== #config token blacklist ============="
    rawData=$(${CLI} config config_tx -k token-blacklist -o add -v BTC)
    signData=$(${CLI} wallet sign -d "${rawData}" -k 0xc34b5d9d44ac7b754806f761d3d4d2c4fe5214f6b074c19f069c4f5c2a29c8cc)
    hash=$(${CLI} wallet send -d "${signData}")

    sleep ${chain33BlockTime}
    txs=$(${CLI} tx query_hash -s "${hash}" | jq ".txs")
    if [ "${txs}" == "null" ]; then
        echo "transferTokenAdmin cannot find tx"
        exit 1
    fi

    ${CLI} wallet status
    ${CLI} account list
    ${CLI} mempool list
}

function start_autotest() {

    echo "=========== #run autotest, make sure saving autotest.log.last file============="

    if [ -e autotest.log ]; then
        cat autotest.log >autotest.log.last
        rm autotest.log
    fi

    ./autotest -f ${autotestTempConfig}

}

function stop_chain33() {

    echo "=========== #stop chain33 ============="
    ${CLI} close
    #wait close
    sleep ${chain33BlockTime}
}

function main() {
    echo "==========================================main begin========================================================"
    config_autotest "$@"
    init
    config_chain33
    start_chain33
    start_autotest
    stop_chain33

    echo "==========================================main end========================================================="
}

main "$@"
