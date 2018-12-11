#!/usr/bin/env bash

set -e
set -o pipefail
#set -o verbose
#set -o xtrace

# os: ubuntu16.04 x64

#chain33 dapp autotest root directory
declare -a Chain33AutoTestDirs=("system" "plugin" "vendor/github.com/33cn/chain33/system" "vendor/github.com/33cn/plugin/plugin")

#copy auto test to specific directory
# check args
if [ "$#" -lt 1 ]; then
    echo "Usage: $0 directory list"
    exit 1
fi

function copyAutoTestConfig() {

    echo "#copy auto test config to path \"$1\""
    local AutoTestConfigFile="$1/autotest.toml"

    #pre config auto test
    {

        echo 'cliCmd="./chain33-cli"'
        echo "checkTimeout=60"
    } >"${AutoTestConfigFile}"

    #copy all the dapp test case config file
    for rootDir in "${Chain33AutoTestDirs[@]}"; do

        if [ ! -d ../../"${rootDir}" ]; then
            continue
        fi

        testDirArr=$(find ../../"${rootDir}" -type d -name autotest)

        for autotest in ${testDirArr}; do

            dapp=$(basename "$(dirname "${autotest}")")
            dappConfig=${autotest}/${dapp}.toml

            #make sure dapp have auto test config
            if [ -e "${dappConfig}" ]; then

                cp "${dappConfig}" "$1"/

                #add dapp test case config
                {
                    echo "[[TestCaseFile]]"
                    echo "dapp=\"$dapp\""
                    echo "filename=\"$dapp.toml\""
                } >>"${AutoTestConfigFile}"

            fi

        done
    done
}

function copyChain33() {

    echo "# copy chain33 bin to path \"$1\", make sure build chain33"
    cp ../chain33 ../chain33-cli ../chain33.toml "$1"
    find ../../ -path '*cmd/chain33/chain33.test.toml' -exec cp {} "$1" ';'
}

for dir in "$@"; do

    #check dir exist
    if [ ! -d "${dir}" ]; then
        mkdir "${dir}"
    fi
    cp autotest "${dir}"
    copyAutoTestConfig "${dir}"
    copyChain33 "${dir}"
    echo "# all copy have done!"

done
