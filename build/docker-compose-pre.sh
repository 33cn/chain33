#!/usr/bin/env bash

set -e
set -o pipefail
#set -o verbose
#set -o xtrace

OP="${1}"
PROJ="${2}"
DAPP="${3}"

function run_dapp(){
    app=$1
    if [ -d "${app}" ] && [ "${app}" != "system" ]; then
        echo "============ run dapp=$app start ================="
        rm -rf "${app}"-ci && mkdir -p "${app}"-ci && cp ./"${app}"/* ./"${app}"-ci && cp -n * ./"${app}"-ci/ && echo $?
        cd "${app}"-ci/ && pwd
        if ! ./docker-compose.sh "${PROJ}" "${app}"
        then
            exit 1
        fi
        cd ..
        echo "============ run dapp=$app end ================="
    fi

}

function down_dapp(){
    app=$1
    if [ -d "${app}" ] && [ "${app}" != "system" ]; then
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

function forktest_dapp(){
    app=$1
    if [ -d "${app}" ] ; then
        echo "============ run dapp=$app start ================="
        rm -rf "${app}"-ci && mkdir -p "${app}"-ci && cp ./"${app}"/* ./"${app}"-ci && echo $?
        cp -n * ./"${app}"-ci/ && echo $?
        cd "${app}"-ci/ && pwd
        appfile="fork-test.sh"
        if [ -e $appfile ] || [ "${app}" == "system" ]; then
            if ! ./system-fork-test.sh "${PROJ}" "${app}"
            then
                exit 1
            fi
        else
            echo "#### note: dapp=$app not exit the fork test file: fork-test.sh ####"
        fi
        cd ..
        echo "============ run dapp=$app end ================="
    fi

}

function main() {
    if [ "${OP}" == "run" ];then
        if [ "${DAPP}" == "all" ] ; then
            echo "============ run main start ================="
    #        ./docker-compose.sh $PROJ
    #        [ $? != 0 ] && exit 1
    #        ./docker-compose-down.sh $PROJ
            echo "============ run main end ================="

            dir=$(ls -l ./ |awk '/^d/ {print $NF}')
            for app in $dir
            do
                run_dapp "${app}"
                down_dapp "${app}"
            done
        elif [ -n "${DAPP}" ] ; then
            run_dapp "${DAPP}"
        else
           ./docker-compose.sh "${PROJ}"
        fi
    elif [ "${OP}" == "down" ];then
        if [ -n "${DAPP}" ] ; then
            down_dapp "${DAPP}"
        else
            ./docker-compose-down.sh "${PROJ}"
        fi
    elif [ "${OP}" == "forktest" ];then
        if [ "${DAPP}" == "all" ] ; then
            dir=$(ls -l ./ |awk '/^d/ {print $NF}')
            for app in $dir
            do
                forktest_dapp "${app}"
                down_dapp "${app}"
            done
        elif [ -n "${DAPP}" ] ; then
            forktest_dapp "${DAPP}"
        else
           forktest_dapp "system"
        fi
    fi

}

# run script
main
