#!/usr/bin/env bash
withdraw="$(./chain33-cli bty withdraw -n "withdraw" -a 500 -e "ticket")"
send="$(./chain33-cli bty transfer -n "" -a 4000 -t 1QJ7pCaL3s4Qnt7nhB76GuSLARgJuLA9ot)"
echo "cli wallet sign -d $withdraw -a 15ZUJCGdaS5VpeK421iMLh7uRppe2My8nq -e 1h > d:/a.txt"
echo "cli wallet sign -d $send -a 15ZUJCGdaS5VpeK421iMLh7uRppe2My8nq -e 1h >> d:/a.txt"
