#!/bin/bash

cmd=$(sed -n '/^[# ]*\[.*\][ ]*/p' servers.conf)
fileName="servers.conf"
serverStr="servers."

getSections() {
    sections="$cmd"
}

getInfoByIndex() {
    index=$1
    nextIndex=$((index + 1))
    info=$(cat <"$fileName" | sed -n "/^[# ]*\\[servers.${index}/,/^[# ]*\\[servers.${nextIndex}/p")
}

getInfoByIndexAndKey() {
    index=$1
    key=$2
    info=$(cat <"$fileName" | sed -n "/^[# ]*\\[servers.${index}/,/^[# ]*\\[servers.${nextIndex}/p" | grep -i "$key" | awk -F '=' '{print $2}')
}

main() {
    getSections
    for line in $sections; do
        if [[ $line =~ $serverStr ]]; then
            index=$(echo "$line" | awk -F '.' '{print $2}' | awk -F ']' '{print$1}')
            getInfoByIndexAndKey "$index" "userName"
            echo "servers.$index: userName->$info"
            getInfoByIndexAndKey "$index" "hostIp"
            echo "servers.$index: hostIp->$info"
            getInfoByIndexAndKey "$index" "port"
            echo "servers.$index: port->$info"
        fi
    done
}

main
