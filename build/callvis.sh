#!/usr/bin/env bash

LIMIT="-limit github.com/ethereum/go-ethereum"
MAIN=" github.com/ethereum/go-ethereum/cmd/geth"
PNG=" | dot -Tpng -o output.png"
INFO=${LIMIT}${MAIN}${PNG}

echo ${INFO}


module=("executor" "p2p" "rpc" "queue" "consensus")

for package in ${module[@]}
do
     echo ${package}
#     go-callvis -focus ${package} -debug -minlen 3 -nostd -group pkg -ignore github.com/ethereum/go-ethereum/log -limit github.com/ethereum/go-ethereum github.com/ethereum/go-ethereum/cmd/geth | dot -Tpng -o ${package}.png
     go-callvis -focus ${package} -debug -minlen 3 -nostd -group pkg -limit gitlab.33.cn/chain33/chain33 gitlab.33.cn/chain33/chain33 | dot -Tpng -o ${package}.png
done ;
