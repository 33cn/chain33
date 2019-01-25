#!/bin/bash
i=0
while IFS='' read -r line || [[ -n $line ]]; do
    [ -z "$line" ] && continue
    ((i++))
    ../chain33-cli --rpc_laddr https://mainnode.bityuan.com:8801 account balance -a "$line" -e ticket
done <"$1"
