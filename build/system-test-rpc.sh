#!/usr/bin/env bash
# shellcheck disable=SC2128

MAIN_HTTP=""
CASE_ERR=""

# $2=0 means true, other false
echo_rst() {
    if [ "$2" -eq 0 ]; then
        echo "$1 ok"
    else
        echo "$1 err"
        CASE_ERR="err"
    fi

}

chain33_lock() {
    ok=$(curl -k -s --data-binary '{"jsonrpc":"2.0","id":2,"method":"Chain33.Lock","params":[]}' -H 'content-type:text/plain;' ${MAIN_HTTP} | jq -r ".result.isOK")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_unlock() {
    ok=$(curl -k -s --data-binary '{"jsonrpc":"2.0","id":2,"method":"Chain33.UnLock","params":[{"passwd":"1314fuzamei","timeout":0}]}' -H 'content-type:text/plain;' ${MAIN_HTTP} | jq -r ".result.isOK")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"

}
function system_test_rpc() {
    local ip=$1
    MAIN_HTTP="https://$ip:8801"
    echo "=========== # system rpc test ============="
    echo "ip=$MAIN_HTTP"

    chain33_lock
    chain33_unlock

    if [ -n "$CASE_ERR" ]; then
        echo "there some case error"
        exit 1
    fi
}

#system_test_rpc $1
