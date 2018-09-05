#!/usr/bin/env bash
withdraw="$(../chain33-cli bty withdraw -n "withdraw" -a "$2" -e "ticket")"
send="$(../chain33-cli bty transfer -n "" -a "$2" -t 1FCX9XJTZXvZteagTrefJEBPZMt8BFmdoi)"
echo "cli wallet sign -d $withdraw -a \"$1\" -e 1h > d:/a.txt"
echo "cli wallet sign -d $send -a \"$1\" -e 1h >> d:/a.txt"
