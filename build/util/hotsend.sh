#!/usr/bin/env bash
withdraw="$(../chain33-cli bty withdraw -n "withdraw" -a 4580624 -e "ticket")"
send="$(../chain33-cli bty transfer -n "" -a 4580624 -t 1FCX9XJTZXvZteagTrefJEBPZMt8BFmdoi)"
echo "cli wallet sign -d $withdraw -a 1Lw6QLShKVbKM6QvMaCQwTh5Uhmy4644CG -e 1h > d:/a.txt"
echo "cli wallet sign -d $send -a 1Lw6QLShKVbKM6QvMaCQwTh5Uhmy4644CG -e 1h >> d:/a.txt"
