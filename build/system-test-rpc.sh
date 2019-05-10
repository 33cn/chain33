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
    #    echo "#response: $resp"
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

function system_test_rpc() {
    local ip=$1
    MAIN_HTTP=$1
    echo "=========== # system rpc test ============="
    echo "ip=$MAIN_HTTP"
    #    set -x
    set +e
    chain33_lock
    chain33_unlock

    chain33_WalletTxList "$ip"
    chain33_ImportPrivkey "$ip"
    chain33_DumpPrivkey "$ip"
    chain33_SendToAddress "$ip"
    chain33_SetTxFee "$ip"
    chain33_SetLabl "$ip"
    chain33_GetPeerInfo "$ip"
    chain33_GetHeaders "$ip"
    chain33_GetLastMemPool "$ip"
    chain33_GetProperFee "$ip"
    chain33_GetBlockOverview "$ip"
    chain33_GetAddrOverview "$ip"
    #这两个测试放在最后
    chain33_SetPasswd "$ip"
    chain33_MergeBalance "$ip"
    set -e

    if [ -n "$CASE_ERR" ]; then
        echo -e "${RED}======system rpc test fail=======${NOC}"
        exit 1
    else
        echo -e "${GRE}======system rpc test pass=======${NOC}"
    fi
}

#system_rpc_test
