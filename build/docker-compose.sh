#!/usr/bin/env bash

set -e -o pipefail

# os: ubuntu16.04 x64
# first, you must install jq tool of json
# sudo apt-get install jq
# sudo apt-get install shellcheck, in order to static check shell script
#

PWD=$(cd "$(dirname "$0")" && pwd)
export PATH="$PWD:$PATH"

function init(){
    # update test environment
    sed -i 's/^TestNet=.*/TestNet=true/g' chain33.toml

    # p2p
    sed -i 's/^seeds=.*/seeds=["172.18.18.151:13802","172.18.18.97:13802","172.18.18.177:13802","172.18.18.132:13802","172.18.18.139:13802","172.18.18.100:13802"]/g' chain33.toml
    sed -i 's/^enable=.*/enable=true/g' chain33.toml
    sed -i 's/^isSeed=.*/isSeed=true/g' chain33.toml
    sed -i 's/^innerSeedEnable=.*/innerSeedEnable=false/g' chain33.toml
    sed -i 's/^useGithub=.*/useGithub=false/g' chain33.toml

    # rpc
    sed -i 's/^jrpcBindAddr=.*/jrpcBindAddr="0.0.0.0:8801"/g' chain33.toml
    sed -i 's/^grpcBindAddr=.*/grpcBindAddr="0.0.0.0:8802"/g' chain33.toml
    sed -i 's/^whitlist=.*/whitlist=["localhost","127.0.0.1","0.0.0.0"]/g' chain33.toml

    # wallet
    sed -i 's/^minerdisable=.*/minerdisable=false/g' chain33.toml

    # docker-compose ps
    sudo docker-compose ps

    # remove exsit container
    sudo docker-compose down

    # create and run docker-compose container
    sudo docker-compose up --build -d

    echo "=========== sleep 60s ============="
    sleep 60

    # docker-compose ps
    sudo docker-compose ps

    # query node run status
    chain33-cli block last_header
    chain33-cli net info

    chain33-cli net peer_info
    peersCount=$(chain33-cli net peer_info | jq '.[] | length')
    echo "${peersCount}"
    if [ "${peersCount}" != "6" ]; then
        echo "peers error"
        exit 1
    fi

    #echo "=========== # create seed for wallet ============="
    #seed=$(chain33-cli seed generate -l 0 | jq ".seed")
    #if [ -z "${seed}" ]; then
    #    echo "create seed error"
    #    exit 1
    #fi

    echo "=========== start set wallet1 ============="

    echo "=========== # save seed to wallet ============="
    result=$(sudo docker exec -it build_chain30_1 ./chain33-cli seed save -p 1314 -s "tortoise main civil member grace happy century convince father cage beach hip maid merry rib" | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "save seed to wallet error seed: ""${seed}"", result: ""$result"
        exit 1
    fi

    sleep 2

    echo "=========== # unlock wallet ============="
    result=$(sudo docker exec -it build_chain30_1 ./chain33-cli wallet unlock -p 1314 -t 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        exit 1
    fi

    sleep 2

    echo "=========== # import private key transer ============="
    result=$(sudo docker exec -it build_chain30_1 ./chain33-cli account import_key -k 2AFF1981291355322C7A6308D46A9C9BA311AA21D94F36B43FC6A6021A1334CF -l transer | jq ".label")
    echo ${result}
    if [ -z "${result}" ]; then
        exit 1
    fi

    sleep 2

    echo "=========== # import private key minig ============="
    result=$(sudo docker exec -it build_chain30_1 ./chain33-cli account import_key -k 2116459C0EC8ED01AA0EEAE35CAC5C96F94473F7816F114873291217303F6989 -l mining | jq ".label")
    echo ${result}
    if [ -z "${result}" ]; then
        exit 1
    fi

    sleep 2

    echo "=========== # set auto mining ============="
    result=$(sudo docker exec -it build_chain30_1 ./chain33-cli wallet auto_mine -f 1 | jq ".isok")
    if [ "${result}" = "false" ]; then
        exit 1
    fi
    echo "=========== set wallet1 end ============="

    echo "=========== start set wallet2 ============="
    echo "=========== # save seed to wallet ============="
    result=$(chain33-cli seed save -p 1314 -s "tortoise main civil member grace happy century convince father cage beach hip maid merry rib" | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "save seed to wallet error seed, result: ${result}"
        exit 1
    fi

    sleep 2

    echo "=========== # unlock wallet ============="
    result=$(chain33-cli wallet unlock -p 1314 -t 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        exit 1
    fi

    sleep 2

    echo "=========== # import private key transfer ============="
    result=$(chain33-cli account import_key -k CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944 -l transfer | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi

    sleep 2

    echo "=========== # import private key mining ============="
    result=$(chain33-cli account import_key -k 4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01 -l mining | jq ".label")
    echo "${result}"
    if [ -z "${result}" ]; then
        exit 1
    fi

    sleep 2
    echo "=========== # set auto mining ============="
    result=$(chain33-cli wallet auto_mine -f 1 | jq ".isok")
    if [ "${result}" = "false" ]; then
        exit 1
    fi
    echo "=========== set wallet2 end ============="

    echo "=========== sleep 60s ============="
    sleep 60

    echo "=========== check genesis hash ========== "
    chain33-cli block hash -t 0
    res=$(chain33-cli block hash -t 0 | jq ".hash")
    count=$(echo "$res" | grep -c "0x67c58d6ba9175313f0468ae4e0ddec946549af7748037c2fdd5d54298afd20b6")
    if [ "${count}" != 1 ]; then
        echo "genesis hash error!"
        exit 1
    fi

    echo "=========== query height ========== "
    chain33-cli block last_header
    result=$(chain33-cli block last_header | jq ".height")
    if [ "${result}" -lt 1 ]; then
        exit 1
    fi


    chain33-cli wallet status
    chain33-cli account list
    chain33-cli mempool list
    # chain33-cli mempool last_txs
    #sudo docker-compose logs -f
}

function transfer(){
    echo "=========== # transfer ============="
    hashes=()
    for((i=0;i<10;i++));do
        hash=$(chain33-cli send bty transfer -a 1 -n test -t 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt -k 4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01)
        hashes=(${hashes[*]} $hash)
        sleep 1
    done
    echo "len: ${#hashes[@]}"
    if [ ${#hashes[*]} != 10 ]; then
        echo tx number wrong
        exit 1
    fi

    sleep 20
    for((i=0;i<${#hashes[*]};i++))
    do
        txs=$(chain33-cli tx query_hash -s "${hashes[$i]}" | jq ".txs")
        if [ -z "${txs}" ]; then
            echo "cannot find tx"
            exit 1
        fi
    done

    sleep 2
    echo "=========== # withdraw ============="
    hash=$(chain33-cli send bty send_exec -a 2 -n deposit -e ticket -k CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944)
    echo "${hash}"
    sleep 20
    before=$(chain33-cli account balance -a 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt -e ticket | jq ".balance")
    before=$(echo "$before" | bc)
    if [ "${before}" == 0.0000 ]; then
        echo "wrong ticket balance, should not be zero"
        exit 1
    fi

    hash=$(chain33-cli send bty withdraw -a 1 -n withdraw -e ticket -k CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944)
    echo "${hash}"
    sleep 25
    txs=$(chain33-cli tx query_hash -s "${hash}" | jq ".txs")
    if [ "${txs}" == "null" ]; then
        echo "withdraw cannot find tx"
        exit 1
    fi
}

function main(){
    echo "==========================================main begin========================================================"
    init
    #transfer
    # TODO other work!!!
    echo "==========================================main end========================================================="
}

# run script
main




