#!/usr/bin/env bash
# shellcheck disable=SC2128

MAIN_HTTP=""
CASE_ERR=""

#color
RED='\033[1;31m'
GRE='\033[1;32m'
NOC='\033[0m'

# $2=0 means true, other false
echo_rst() {
    if [ "$2" -eq 0 ]; then
        echo -e "${GRE}$1 ok${NOC}"
    else
        echo -e "${RED}$1 fail${NOC}"
        CASE_ERR="err"
    fi

}

chain33_lock() {
    ok=$(curl -ksd '{"method":"Chain33.Lock","params":[]}' ${MAIN_HTTP} | jq -r ".result.isOK")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_unlock() {
    ok=$(curl -ksd '{"method":"Chain33.UnLock","params":[{"passwd":"1314fuzamei","timeout":0}]}' ${MAIN_HTTP} | jq -r ".result.isOK")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"

}

chain33_WalletTxList() {

    req='"method":"Chain33.WalletTxList", "params":[{"fromTx":"", "count":2, "direction":1}]'
    echo "#request: $req"
    resp=$(curl -ksd "{$req}" "$1")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result.txDetails|length == 2)' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"

}

chain33_ImportPrivkey() {

    req='"method":"Chain33.ImportPrivkey", "params":[{"privkey":"0x88b2fb90411935872f0501dd13345aba19b5fac9b00eb0dddd7df977d4d5477e", "label":"testimportkey"}]'
    echo "#request: $req"
    resp=$(curl -ksd "{$req}" "$1")
    #        echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result.label=="testimportkey") and (.result.acc.addr == "1D9xKRnLvV2zMtSxSx33ow1GF4pcbLcNRt")' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"

}

chain33_DumpPrivkey() {

    req='"method":"Chain33.DumpPrivkey", "params":[{"data":"1D9xKRnLvV2zMtSxSx33ow1GF4pcbLcNRt"}]'
    echo "#request: $req"
    resp=$(curl -ksd "{$req}" "$1")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result.data=="0x88b2fb90411935872f0501dd13345aba19b5fac9b00eb0dddd7df977d4d5477e")' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"

}

chain33_SendToAddress() {

    req='"method":"Chain33.SendToAddress", "params":[{"from":"12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv","to":"1D9xKRnLvV2zMtSxSx33ow1GF4pcbLcNRt", "amount":100000000, "note":"test\n"}]'
    echo "#request: $req"
    resp=$(curl -ksd "{$req}" "$1")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result.hash|length==66)' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_SetTxFee() {

    req='"method":"Chain33.SetTxFee", "params":[{"amount":100000}]'
    echo "#request: $req"
    resp=$(curl -ksd "{$req}" "$1")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and .result.isOK' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_SetLabl() {

    req='"method":"Chain33.SetLabl", "params":[{"addr":"1D9xKRnLvV2zMtSxSx33ow1GF4pcbLcNRt", "label":"updatetestimport"}]'
    echo "#request: $req"
    resp=$(curl -ksd "{$req}" "$1")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result.label=="updatetestimport") and (.result.acc.addr == "1D9xKRnLvV2zMtSxSx33ow1GF4pcbLcNRt")' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GetPeerInfo() {

    req='"method":"Chain33.GetPeerInfo", "params":[{}]'
    echo "#request: $req"
    resp=$(curl -ksd "{$req}" "$1")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result.peers|length >= 1) and (.result.peers[0] |
    [has("addr", "port", "name", "mempoolSize", "self", "header"), true] | unique | length == 1)' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GetHeaders() {

    req='"method":"Chain33.GetHeaders", "params":[{"start":1, "end":2, "isDetail":true}]'
    echo "#request: $req"
    resp=$(curl -ksd "{$req}" "$1")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result.items|length == 2) and (.result.items[0] |
    [has("version","parentHash", "txHash", "stateHash", "height", "blockTime", "txCount", "hash", "difficulty"),true] |
    unique | length == 1 )' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GetLastMemPool() {

    req='"method":"Chain33.GetLastMemPool", "params":[{}]'
    echo "#request: $req"
    resp=$(curl -ksd "{$req}" "$1")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result.txs|length >= 0)' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GetProperFee() {

    req='"method":"Chain33.GetProperFee", "params":[{}]'
    echo "#request: $req"
    resp=$(curl -ksd "{$req}" "$1")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result.properFee > 10000)' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GetBlockOverview() {

    hash=$(curl -ksd '{"method":"Chain33.GetHeaders", "params":[{"start":1, "end":1, "isDetail":true}]}' ${MAIN_HTTP} | jq '.result.items[0].hash')
    req='"method":"Chain33.GetBlockOverview", "params":[{"hash":'"$hash"'}]'
    echo "#request: $req"
    resp=$(curl -ksd "{$req}" "$1")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and
    (.result| [has("head", "txCount", "txHashes"), true]|unique|length == 1) and
    (.result.txCount == (.result.txHashes|length))' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GetAddrOverview() {

    req='"method":"Chain33.GetAddrOverview", "params":[{"addr":"12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"}]'
    echo "#request: $req"
    resp=$(curl -ksd "{$req}" "$1")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result|[has("reciver", "balance", "txCount"), true]|unique|length == 1)' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_SetPasswd() {

    req='"method":"Chain33.SetPasswd", "params":[{"oldPass":"1314fuzamei", "newPass":"1314fuzamei"}]'
    echo "#request: $req"
    resp=$(curl -ksd "{$req}" "$1")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and .result.isOK' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_MergeBalance() {

    req='"method":"Chain33.MergeBalance", "params":[{"to":"12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"}]'
    echo "#request: $req"
    resp=$(curl -ksd "{$req}" "$1")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result.hashes|length > 0)' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_QueryTotalFee() {
    local height=1
    hash=$(curl -ksd '{"method":"Chain33.GetBlockHash","params":[{"height":'$height'}]}' ${MAIN_HTTP} | jq -r ".result.hash")
    if [ -z "$hash" ]; then
        echo "hash is null"
        echo_rst "$FUNCNAME" 1
    fi
    prefixhash_base64=$(echo -n "TotalFeeKey:" | base64)
    blockhash_base64=$(echo -n "$hash" | cut -d " " -f 1 | xxd -r -p | base64)
    base64_hash="$prefixhash_base64$blockhash_base64"
    # shellcheck disable=SC2086
    txs=$(curl -ksd '{"method":"Chain33.QueryTotalFee","params":[{"keys":["'$base64_hash'"]}]}' ${MAIN_HTTP} | jq -r ".result.txCount")
    [ "$txs" -ge 0 ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GetNetInfo() {
    method="GetNetInfo"
    addr=$(curl -ksd '{"method":"Chain33.'$method'","params":[]}' ${MAIN_HTTP} | jq -r ".result.externalAddr")
    service=$(curl -ksd '{"method":"Chain33.GetNetInfo","params":[]}' ${MAIN_HTTP} | jq -r ".result.service")
    [ "$addr" != "null" ] && [ "$service" == "true" ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GetFatalFailure() {
    r1=$(curl -ksd '{"method":"Chain33.GetFatalFailure","params":[]}' ${MAIN_HTTP} | jq -r ".result")
    error=$(curl -ksd '{"method":"Chain33.GetFatalFailure","params":[]}' ${MAIN_HTTP} | jq -r ".error")
    [ "$r1" -eq 0 ] && [ "$error" == null ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_DecodeRawTransaction() {
    tx="0a05636f696e73122c18010a281080c2d72f222131477444795771577233553637656a7663776d333867396e7a6e7a434b58434b7120a08d0630a696c0b3f78dd9ec083a2131477444795771577233553637656a7663776d333867396e7a6e7a434b58434b71"
    r1=$(curl -ksd '{"method":"Chain33.DecodeRawTransaction","params":[{"txHex":"'$tx'"}]}' ${MAIN_HTTP} | jq -r ".result.txs[0].execer")
    [ "$r1" == "coins" ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GetTimeStatus() {
    r1=$(curl -ksd '{"method":"Chain33.GetTimeStatus","params":[]}' ${MAIN_HTTP} | jq -r ".result.localTime")
    if [ -z "$r1" ]; then
        curl -ksd '{"method":"Chain33.GetTimeStatus","params":[]}' ${MAIN_HTTP}
    fi
    [ -n "$r1" ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GetLastBlockSequence() {
    r1=$(curl -ksd '{"method":"Chain33.GetLastBlockSequence","params":[]}' ${MAIN_HTTP} | jq -r ".result")
    [ "$r1" -ge 0 ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GetBlockSequences() {
    r1=$(curl -ksd '{"method":"Chain33.GetBlockSequences","params":[{"start":1,"end":3,"isDetail":true}]}' ${MAIN_HTTP} | jq -r ".result.blkseqInfos[2].hash")
    [ -n "$r1" ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GetBlockByHashes() {
    hash0=$(curl -ksd '{"method":"Chain33.GetBlockSequences","params":[{"start":1,"end":3,"isDetail":true}]}' ${MAIN_HTTP} | jq -r ".result.blkseqInfos[0].hash")
    hash1=$(curl -ksd '{"method":"Chain33.GetBlockSequences","params":[{"start":1,"end":3,"isDetail":true}]}' ${MAIN_HTTP} | jq -r ".result.blkseqInfos[1].hash")
    hash2=$(curl -ksd '{"method":"Chain33.GetBlockSequences","params":[{"start":1,"end":3,"isDetail":true}]}' ${MAIN_HTTP} | jq -r ".result.blkseqInfos[2].hash")

    # curl -ksd '{"method":"Chain33.GetBlockByHashes","params":[{"hashes":["'$hash1'","'$hash2'"]}]}'  ${MAIN_HTTP}
    # shellcheck disable=SC2086
    p1=$(curl -ksd '{"method":"Chain33.GetBlockByHashes","params":[{"hashes":["'$hash1'","'$hash2'"]}]}' ${MAIN_HTTP} | jq -r ".result.items[0].block.parentHash")
    # shellcheck disable=SC2086
    p2=$(curl -ksd '{"method":"Chain33.GetBlockByHashes","params":[{"hashes":["'$hash1'","'$hash2'"]}]}' ${MAIN_HTTP} | jq -r ".result.items[1].block.parentHash")
    [ "$p1" == "$hash0" ] && [ "$p2" == "$hash1" ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_ConvertExectoAddr() {
    r1=$(curl -ksd '{"method":"Chain33.ConvertExectoAddr","params":[{"execname":"coins"}]}' ${MAIN_HTTP} | jq -r ".result")
    [ "$r1" == "1GaHYpWmqAJsqRwrpoNcB8VvgKtSwjcHqt" ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GetExecBalance() {
    local height=6802
    statehash=$(curl -ksd '{"method":"Chain33.GetBlocks","params":[{"start":'$height',"end":'$height',"isDetail":false}]}' ${MAIN_HTTP} | jq -r ".result.items[0].block.stateHash")
    state_base64=$(echo -n "$statehash" | cut -d " " -f 1 | xxd -r -p | base64)
    addr="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
    addr_base64=$(echo -n "$addr" | base64)
    # curl -ksd '{"method":"Chain33.GetExecBalance","params":[{"symbol":"bty","stateHash":"'$state_base64'","addr":"'$addr_base64'","execer":"coins","count":100}]}'  ${MAIN_HTTP}
    # shellcheck disable=SC2086
    r1=$(curl -ksd '{"method":"Chain33.GetExecBalance","params":[{"symbol":"bty","stateHash":"'$state_base64'","addr":"'$addr_base64'","execer":"coins","count":100}]}' ${MAIN_HTTP} | jq -r ".error")
    [ "$r1" == "null" ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_AddSeqCallBack() {
    r1=$(curl -ksd '{"method":"Chain33.AddSeqCallBack","params":[{"name":"test","url":"http://test","encode":"json"}]}' ${MAIN_HTTP} | jq -r ".result.isOK")
    [ "$r1" == "true" ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_ListSeqCallBack() {
    r1=$(curl -ksd '{"method":"Chain33.ListSeqCallBack","params":[]}' ${MAIN_HTTP} | jq -r ".result.items[0].name")
    [ "$r1" == "test" ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GetSeqCallBackLastNum() {
    r1=$(curl -ksd '{"method":"Chain33.GetSeqCallBackLastNum","params":[{"data":"test"}]}' ${MAIN_HTTP} | jq -r ".result.data")
    [ "$r1" == "-1" ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GetCoinSymbol() {
    r1=$(curl -ksd '{"method":"Chain33.GetCoinSymbol","params":[]}' ${MAIN_HTTP} | jq -r ".result.data")
    [ "$r1" == "bty" ]
    echo_rst "$FUNCNAME" "$?"
}

run_testcases() {
    #    set -x
    set +e

    chain33_lock
    chain33_unlock

    chain33_WalletTxList "$1"
    chain33_ImportPrivkey "$1"
    chain33_DumpPrivkey "$1"
    chain33_SendToAddress "$1"
    chain33_SetTxFee "$1"
    chain33_SetLabl "$1"
    chain33_GetPeerInfo "$1"
    chain33_GetHeaders "$1"
    chain33_GetLastMemPool "$1"
    chain33_GetProperFee "$1"
    chain33_GetBlockOverview "$1"
    chain33_GetAddrOverview "$1"

    chain33_QueryTotalFee
    chain33_GetNetInfo
    chain33_GetFatalFailure
    chain33_DecodeRawTransaction
    chain33_GetTimeStatus
    chain33_GetLastBlockSequence
    chain33_GetBlockSequences
    chain33_GetBlockByHashes
    chain33_ConvertExectoAddr
    chain33_GetExecBalance
    chain33_AddSeqCallBack
    chain33_ListSeqCallBack
    chain33_GetSeqCallBackLastNum
    chain33_GetCoinSymbol

    #这两个测试放在最后
    chain33_SetPasswd "$1"
    chain33_MergeBalance "$1"
    set -e

}

function system_test_rpc() {
    MAIN_HTTP=$1
    echo "=========== # system rpc test ============="
    echo "ip=$1"

    run_testcases "$1"

    if [ -n "$CASE_ERR" ]; then
        echo -e "${RED}======system rpc test fail=======${NOC}"
        exit 1
    else
        echo -e "${GRE}======system rpc test pass=======${NOC}"
    fi
}

system_test_rpc $1
