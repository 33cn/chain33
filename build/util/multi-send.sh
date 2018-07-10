#!/bin/bash
i=0
while IFS='' read -r line || [[ -n $line ]]; do
    [ -z "$line" ] && continue
    ((i++))
    echo "sh hotsend.sh $line $i"
done <"$1"
