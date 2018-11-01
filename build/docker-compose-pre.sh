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

TESTCASEFILE="testcase.sh"
FORKTESTFILE="fork-test.sh"

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

function run_dapp() {
    app=$1
    test=$2

    echo "============ run dapp=$app start ================="
    rm -rf "${app}"-ci && mkdir -p "${app}"-ci && cp ./"${app}"/* ./"${app}"-ci && echo $?
    cp -n ./* ./"${app}"-ci/ && echo $?
    cd "${app}"-ci/ && pwd

    if [ "$test" == "$FORKTESTFILE" ]; then
        sed -i $sedfix 's/^system_coins_file=.*/system_coins_file="..\/system\/coins\/fork-test.sh"/g' system-fork-test.sh
        if ! ./system-fork-test.sh "${PROJ}" "${app}"; then
            exit 1
        fi
    elif [ "$test" == "$TESTCASEFILE" ]; then
        if ! ./docker-compose.sh "${PROJ}" "${app}"; then
            exit 1
        fi
    fi
    cd ..
    echo "============ run dapp=$app end ================="

}

function run_single_app() {
    app=$1
    testfile=$2

    if [ -d "${app}" ] && [ "${app}" != "system" ]; then
        if [ -e "$app/$testfile" ]; then
            run_dapp "${app}" "$testfile"
            if [ "$#" -gt 2 ]; then
                down_dapp "${app}"
            fi
        else
            echo "#### dapp=$app not exist the test file: $testfile ####"
        fi
    else
        echo "#### dapp=$app not exist or is system dir ####"
    fi
}

function main() {
    if [ "${OP}" == "run" ]; then
        if [ "${DAPP}" == "all" ] || [ "${DAPP}" == "ALL" ]; then
            echo "============ run main start ================="
            if ! ./docker-compose.sh "$PROJ"; then
                exit 1
            fi
            ./docker-compose-down.sh "$PROJ"
            echo "============ run main end ================="

            find . -maxdepth 1 -type d -name "*-ci" -exec rm -rf {} \;
            dir=$(find . -maxdepth 1 -type d ! -name system ! -name . | sed 's/^\.\///')
            for app in $dir; do
                run_single_app "${app}" "$TESTCASEFILE" "down"
            done
        elif [ -n "${DAPP}" ]; then
            run_single_app "${DAPP}" "$TESTCASEFILE"
        else
            ./docker-compose.sh "${PROJ}"
        fi
    elif [ "${OP}" == "down" ]; then
        if [ "${DAPP}" == "all" ] || [ "${DAPP}" == "ALL" ]; then
            dir=$(find . -maxdepth 1 -type d ! -name system ! -name . | sed 's/^\.\///')
            for app in $dir; do
                down_dapp "${app}"
            done
            ./docker-compose-down.sh "${PROJ}"
        elif [ -n "${DAPP}" ]; then
            down_dapp "${DAPP}"
        else
            ./docker-compose-down.sh "${PROJ}"
        fi
    elif [ "${OP}" == "forktest" ]; then
        if [ "${DAPP}" == "all" ] || [ "${DAPP}" == "ALL" ]; then
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
                run_single_app "${app}" "$FORKTESTFILE" "down"
            done
        elif [ -n "${DAPP}" ]; then
            run_single_app "${DAPP}" "$FORKTESTFILE"
        else
            ./system-fork-test.sh "${PROJ}"
        fi
    fi

}

# run script
main
