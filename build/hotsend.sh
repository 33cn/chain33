#!/usr/bin/env bash
withdraw="$(./chain33-cli bty withdraw -n "withdraw" -a 1400000 -e "ticket")"
send="$(./chain33-cli bty transfer -n "" -a 1400000 -t 1FCX9XJTZXvZteagTrefJEBPZMt8BFmdoi)"
echo "cli wallet sign -d $withdraw -a 1KNGHukhbBnbWWnMYxu1C7YMoCj45Z3amm -e 1h > d:/a.txt"
echo "cli wallet sign -d $send -a 1KNGHukhbBnbWWnMYxu1C7YMoCj45Z3amm -e 1h >> d:/a.txt"
