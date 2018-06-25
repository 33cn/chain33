#!/usr/bin/env bash
bind=$(./chain33-cli ticket bind_miner -b "$2" -o "$1")
send=$(./chain33-cli bty transfer -n "coins->ticket" -a "$3" -t 16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp)
echo "chain33-cli wallet sign -d $bind -a $1 -e 1h > miner.txt"
echo "chain33-cli wallet sign -d $send -a $1 -e 1h >> miner.txt"
