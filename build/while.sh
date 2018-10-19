#!/bin/bash
CURDIR=$(
    cd $(dirname ${BASH_SOURCE[0]})
    pwd
)
echo $CURDIR
while :; do
    ./chain33-cli net time
done
