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

chain33_GetHexTxByHash() {
    #先获取一笔交易
    reHash=$(curl -k -s --data-binary '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetTxByAddr","params":[{"addr":"14KEKbYtKKQm4wMthSK9J4La4nAiidGozt","flag":0,"count":1,"direction":0,"height":-1,"index":0}]}' -H 'content-type:text/plain;' ${MAIN_HTTP} | jq -r '.result.txInfos[0].hash')
    #查询交易
    resp=$(curl -k -s --data-binary '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetHexTxByHash","params":[{"hash":"'${reHash}'","upgrade":false}]}' -H 'content-type:text/plain;' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result != null)' <<<"$resp")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_QueryTransaction() {
    #先获取一笔交易
    reHash=$(curl -k -s --data-binary '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetTxByAddr","params":[{"addr":"14KEKbYtKKQm4wMthSK9J4La4nAiidGozt","flag":0,"count":1,"direction":0,"height":-1,"index":0}]}' -H 'content-type:text/plain;' ${MAIN_HTTP} | jq -r '.result.txInfos[0].hash')
    #查询交易
    resp=$(curl -k -s --data-binary '{"jsonrpc":"2.0","id":2,"method":"Chain33.QueryTransaction","params":[{"hash":"'${reHash}'","upgrade":false}]}' -H 'content-type:text/plain;' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result.receipt.tyName == "ExecOk") and (.result.height >= 0) and (.result.index >= 0) and (.result.amount >= 0)' <<<"$resp")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_GetBlocks() {
    result=$(curl -k -s --data-binary '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetBlocks","params":[{"start":1,"end":2}]}' -H 'content-type:text/plain;' ${MAIN_HTTP} | jq -r ".result.items[1].block.height")
    [ "$result" -eq 2 ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_GetLastHeader() {
    resp=$(curl -k -s --data-binary '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetLastHeader","params":[{}]}' -H 'content-type:text/plain;' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result.height >= 0) and (.result | [has("version","parentHash", "txHash", "stateHash", "height", "blockTime", "txCount", "hash", "difficulty"),true] | unique | length == 1)' <<<"$resp")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_GetTxByAddr() {
    resp=$(curl -k -s --data-binary '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetTxByAddr","params":[{"addr":"14KEKbYtKKQm4wMthSK9J4La4nAiidGozt","flag":0,"count":1,"direction":0,"height":-1,"index":0}]}' -H 'content-type:text/plain;' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result.txInfos[0].index >= 0) and (.result.txInfos[0] | [has("hash", "height", "index", "assets"),true] | unique | length == 1)' <<<"$resp")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_GetTxByHashes() {
    resp=$(curl -k -s --data-binary '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetTxByHashes","params":[{"hashes":["0x8040109d3859827d0f0c80ce91cc4ec80c496c45250f5e5755064b6da60842ab","0x501b910fd85d13d1ab7d776bce41a462f27c4bfeceb561dc47f0a11b10f452e4"]}]}' -H 'content-type:text/plain;' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result.txs|length == 2)' <<<"$resp")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_GetMempool() {
    resp=$(curl -k -s --data-binary '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetMempool","params":[{}]}' -H 'content-type:text/plain;' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result.txs|length >= 0)' <<<"$resp")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_GetAccountsV2() {
    resp=$(curl -k -s --data-binary '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetAccountsV2","params":[{}]}' -H 'content-type:text/plain;' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result.wallets|length >= 0)' <<<"$resp")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_GetAccounts() {
    resp=$(curl -k -s --data-binary '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetAccounts","params":[{}]}' -H 'content-type:text/plain;' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result.wallets|length >= 0)' <<<"$resp")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_NewAccount() {
    resp=$(curl -k -s --data-binary '{"jsonrpc":"2.0","id":2,"method":"Chain33.NewAccount","params":[{"label":"test169"}]}' -H 'content-type:text/plain;' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result.label == "test169") and (.result.acc | [has("addr"),true] | unique | length == 1)' <<<"$resp")
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

    chain33_GetHexTxByHash
    chain33_QueryTransaction
    chain33_GetBlocks
    chain33_GetLastHeader
    chain33_GetTxByAddr
    chain33_GetTxByHashes
    chain33_GetMempool
    chain33_GetAccountsV2
    chain33_GetAccounts
    chain33_NewAccount

    if [ -n "$CASE_ERR" ]; then
        echo "there some case error"
        exit 1
    fi
}

#system_test_rpc $1
