#!/usr/bin/env bash

set -e
set -o pipefail
#set -o verbose
#set -o xtrace

sedfix=""
if [ "$(uname)" == "Darwin" ]; then
    sedfix=".bak"
fi

OP="${1}"
PROJ="${2}"
DAPP="${3}"

function run_dapp() {
    app=$1
    if [ -d "${app}" ]; then
        echo "============ run dapp=$app start ================="
        rm -rf "${app}"-ci && mkdir -p "${app}"-ci && cp ./"${app}"/* ./"${app}"-ci && cp -n ./* ./"${app}"-ci/ && echo $?
        cd "${app}"-ci/ && pwd
        if ! ./docker-compose.sh "${PROJ}" "${app}"; then
            exit 1
        fi
        cd ..
        echo "============ run dapp=$app end ================="
    fi

}

function down_dapp() {
    app=$1
    if [ -d "${app}" ] && [ "${app}" != "system" ] && [ -d "${app}-ci" ]; then
        cd "${app}"-ci/ && pwd

        echo "============ down dapp=$app start ================="
        ./docker-compose-down.sh "${PROJ}" "${app}"
        echo "============ down dapp=$app end ================="

        cd .. && rm -rf ./"${app}"-ci
    fi
}

function forktest_dapp() {
    app=$1
    if [ -d "${app}" ]; then
        echo "============ run dapp=$app start ================="
        rm -rf "${app}"-ci && mkdir -p "${app}"-ci && cp ./"${app}"/* ./"${app}"-ci && echo $?
        cp -n ./* ./"${app}"-ci/ && echo $?
        cd "${app}"-ci/ && pwd
        sed -i $sedfix 's/^system_coins_file=.*/system_coins_file="..\/system\/coins\/fork-test.sh"/g' system-fork-test.sh
        appfile="fork-test.sh"
        if [ -e $appfile ]; then
            if ! ./system-fork-test.sh "${PROJ}" "${app}"; then
                exit 1
            fi
        else
            echo "#### note: dapp=$app not exit the fork test file: fork-test.sh ####"
        fi
        cd ..
        echo "============ run dapp=$app end ================="
    fi

}

function forktest_down() {
    app=$1
    if [ -d "${app}" ] && [ "${app}" != "system" ] && [ -d "${app}-ci" ]; then
        cd "${app}"-ci/ && pwd
        appfile="fork-test.sh"
        if [ -e $appfile ]; then
            echo "============ down dapp=$app start ================="
            ./docker-compose-down.sh "${PROJ}" "${app}"
            echo "============ down dapp=$app end ================="
        fi
        cd .. && rm -rf ./"${app}"-ci
    fi
}

function main() {
    if [ "${OP}" == "run" ]; then
        if [ "${DAPP}" == "all" ]; then
            echo "============ run main start ================="
            if ! ./docker-compose.sh "$PROJ"; then
                exit 1
            fi
            ./docker-compose-down.sh "$PROJ"
            echo "============ run main end ================="

            #dir=$(ls -l ./ |awk '/^d/ {print $NF}')
            find . -maxdepth 1 -type d -name "*-ci" -exec rm -rf {} \;
            dir=$(find . -maxdepth 1 -type d ! -name system ! -name . | sed 's/^\.\///')
            for app in $dir; do
                run_dapp "${app}"
                down_dapp "${app}"
            done
        elif [ -n "${DAPP}" ]; then
            run_dapp "${DAPP}"
        else
            ./docker-compose.sh "${PROJ}"
        fi
    elif [ "${OP}" == "down" ]; then
        if [ "${DAPP}" == "all" ]; then
            dir=$(find . -maxdepth 1 -type d ! -name system ! -name . | sed 's/^\.\///')
            for app in $dir; do
                down_dapp "${app}"
            done
            ./docker-compose-down.sh "${PROJ}"
            remains=$(docker ps -a -q | awk '{print $1}')
            echo $remains
            for rems in $remains; do
                docker rm "$rems" -f
            done
            docker ps -q
        elif [ -n "${DAPP}" ]; then
            down_dapp "${DAPP}"
        else
            ./docker-compose-down.sh "${PROJ}"
        fi
    elif [ "${OP}" == "forktest" ]; then
        if [ "${DAPP}" == "all" ]; then
            echo "============ run main start ================="
            if ! ./system-fork-test.sh "$PROJ"; then
                exit 1
            fi
            echo "============ down main test ================="
            ./docker-compose-down.sh "$PROJ"
            echo "============ run main end ================="
            # remove all the *-ci folders
            find . -maxdepth 1 -type d -name "*-ci" -exec rm -rf {} \;
            dir=$(find . -maxdepth 1 -type d ! -name system ! -name . | sed 's/^\.\///')
            for app in $dir; do
                forktest_dapp "${app}"
                forktest_down "${app}"
            done
        elif [ -n "${DAPP}" ]; then
            forktest_dapp "${DAPP}"
        else
            ./system-fork-test.sh "${PROJ}"
        fi
    fi

}

# run script
main
