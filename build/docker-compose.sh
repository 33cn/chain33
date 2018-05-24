#!/usr/bin/env bash

set -e -o pipefail

# first, you must install jq tool of json
# sudo apt-get install jq

# p2p
sed -i 's/^seeds=.*/seeds=["172.18.18.151:13802","172.18.18.97:13802","172.18.18.177:13802"]/g' chain33.toml
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

echo "=========== sleep 25s ============="
sleep 25

# docker-compose ps
sudo docker-compose ps

# query node run status
./chain33-cli block last_header
./chain33-cli net info

./chain33-cli net peer_info
peersCount=$(./chain33-cli net peer_info | jq '.[] | length')
echo ${peersCount}
if [ "${peersCount}" != "3" ]; then
    echo "peers error"
    exit 1
fi

#echo "=========== # create seed for wallet ============="
#seed=$(./chain33-cli seed generate -l 0 | jq ".seed")
#if [ -z "${seed}" ]; then
#    echo "create seed error"
#    exit 1
#fi

echo "=========== # save seed to wallet ============="
result=$(./chain33-cli seed save -p 1314 -s "tortoise main civil member grace happy century convince father cage beach hip maid merry rib" | jq ".isok")
if [ "${result}" = "false" ]; then
    echo "save seed to wallet error seed: ""${seed}"", result: ""$result"
    exit 1
fi

sleep 2

echo "=========== # unlock wallet ============="
result=$(./chain33-cli wallet unlock -p 1314 -t 0 | jq ".isok")
if [ "${result}" = "false" ]; then
    exit 1
fi

sleep 2

echo "=========== # import private key transfer ============="
result=$(./chain33-cli account import_key -k 56942AD84CCF4788ED6DACBC005A1D0C4F91B63BCF0C99A02BE03C8DEAE71138 -l transfer | jq ".label")
echo ${result}
if [ -z "${result}" ]; then
    exit 1
fi

sleep 2

echo "=========== # import private key mining ============="
result=$(./chain33-cli account import_key -k 4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01 -l mining | jq ".label")
echo ${result}
if [ -z "${result}" ]; then
    exit 1
fi

sleep 2

echo "=========== # transfer ============="
sleep 60
#before=$(./chain33-cli account balance -a 1PUiGcbsccfxW3zuvHXZBJfznziph5miAo -e coins | jq ".balance")
hashes=()
for((i=0;i<10;i++))
do
    hash=$(./chain33-cli send bty transfer -a 1 -n test -t 16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp -k 56942AD84CCF4788ED6DACBC005A1D0C4F91B63BCF0C99A02BE03C8DEAE71138)
    hashes=(${hashes[*]} $hash)
    sleep 1
done
echo ${hashes[@]}
if [ ${#hashes[*]} != 10 ]; then
    echo tx number wrong
    exit 1
fi
sleep 30
for((i=0;i<${#hashes[*]};i++))
do
    txs=$(./chain33-cli tx query_hash -s ${hashes} | jq ".txs")
    if [ $txs -eq "null" ]; then
        echo cannot find tx
        exit 1
    fi
done

sleep 2

echo "=========== # withdraw ============="
before=$(./chain33-cli account balance -a 1PUiGcbsccfxW3zuvHXZBJfznziph5miAo -e ticket | jq ".balance")
before=$(echo $before | bc)
if [ ${before} == 0 ]; then
    echo wrong ticket balance
    exit 1
fi
./chain33-cli ticket close
sleep 30
after=$(./chain33-cli account balance -a 1PUiGcbsccfxW3zuvHXZBJfznziph5miAo -e ticket | jq ".balance")
after=$(echo $after | bc)
if [ ${after} != 0 ]; then
    echo wrong ticket balance
    exit 1
fi

echo "=========== # set auto mining ============="
result=$(./chain33-cli wallet auto_mine -f 1 | jq ".isok")
if [ "${result}" = "false" ]; then
    exit 1
fi

echo "=========== sleep 60s ============="
sleep 60

echo "=========== check genesis hash ========== "
./chain33-cli block hash -t 0
result=$(./chain33-cli block hash -t 0 | jq ".hash")
#echo $result
#if [ ${result} != '0x67c58d6ba9175313f0468ae4e0ddec946549af7748037c2fdd5d54298afd20b6' ]; then
#    exit 1
#fi

echo "=========== query height ========== "
./chain33-cli block last_header
result=$(./chain33-cli block last_header | jq ".height")
if [ ${result} -lt 1 ]; then
    exit 1
fi

./chain33-cli wallet status
./chain33-cli account list
./chain33-cli mempool list
#./chain33-cli mempool last_txs






