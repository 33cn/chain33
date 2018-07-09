#!/usr/bin/env bash
withdraw="$(../chain33-cli bty withdraw -n "withdraw" -a 19000000 -e "ticket")"
send="$(../chain33-cli bty transfer -n "" -a 19000000 -t 1FCX9XJTZXvZteagTrefJEBPZMt8BFmdoi)"
echo "cli wallet sign -d $withdraw -a 1AH9HRd4WBJ824h9PP1jYpvRZ4BSA4oN6Y -e 1h > d:/a.txt"
echo "cli wallet sign -d $send -a 1AH9HRd4WBJ824h9PP1jYpvRZ4BSA4oN6Y -e 1h >> d:/a.txt"
