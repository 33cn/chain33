#!/usr/bin/env bash

set -e
set -o pipefail
#set -o verbose
#set -o xtrace

export COMPOSE_PROJECT_NAME="$1"

DAPP=""
if [ -n "${2}" ]; then
    DAPP=$2
fi
if [ -n "${DAPP}" ]; then
    DAPP_COMPOSE_FILE="docker-compose-${DAPP}.yml"
    if [ -e "$DAPP_COMPOSE_FILE" ]; then
        export COMPOSE_FILE="docker-compose.yml:${DAPP_COMPOSE_FILE}"
    fi

fi

echo "=========== # down docker-compose ============="
echo "=========== # env setting ============="
echo "DAPP=$DAPP"
echo "COMPOSE_FILE=$COMPOSE_FILE"
echo "COMPOSE_PROJECT_NAME=$COMPOSE_PROJECT_NAME"

####################

function down() {
    echo "=========== # docker-compose ps ============="
    docker-compose ps
    # shellchk not recommend the first way
    # remains=( $(docker-compose ps -q | awk '{print $1}') )
    mapfile -t remains < <(docker-compose ps -q | awk '{print $1}')
    num=${#remains[@]}
    echo "container num=$num"
    if [ "$num" -gt 0 ]; then
        # remove exsit container
        echo "=========== # docker-compose down ============="
        docker-compose down --rmi local
    fi

}

# run script
down
