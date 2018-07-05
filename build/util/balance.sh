#!/bin/bash
i=0
while IFS='' read -r line || [[ -n $line ]]; do
    [ -z "$line" ] && continue
    ((i++))
    ../chain33-cli account balance -a "$line" -e ticket
done <"$1"
