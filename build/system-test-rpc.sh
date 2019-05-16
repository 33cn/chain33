#!/usr/bin/env bash
# shellcheck disable=SC2128

MAIN_HTTP=""
IS_PARA=false
CASE_ERR=""

#color
RED='\033[1;31m'
GRE='\033[1;32m'
NOC='\033[0m'

# $2=0 means true, other false
echo_rst() {
    if [ "$2" -eq 0 ]; then
        echo -e "${GRE}$1 ok${NOC}"
    elif [ "$2" -eq 2 ]; then
        echo -e "${GRE}$1 not support${NOC}"
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

    if [ "$IS_PARA" == true ]; then
        echo_rst "$FUNCNAME" 2
    else
        req='"method":"Chain33.GetPeerInfo", "params":[{}]'
        echo "#request: $req"
        resp=$(curl -ksd "{$req}" "$1")
        #    echo "#response: $resp"
        ok=$(jq '(.error|not) and (.result.peers|length >= 1) and (.result.peers[0] |
        [has("addr", "port", "name", "mempoolSize", "self", "header"), true] | unique | length == 1)' <<<"$resp")
        [ "$ok" == true ]
        echo_rst "$FUNCNAME" "$?"
    fi
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
    if [ "$IS_PARA" == true ]; then
        echo_rst "$FUNCNAME" 2
    else
        method="GetNetInfo"
        addr=$(curl -ksd '{"method":"Chain33.'"$method"'","params":[]}' ${MAIN_HTTP} | jq -r ".result.externalAddr")
        service=$(curl -ksd '{"method":"Chain33.GetNetInfo","params":[]}' ${MAIN_HTTP} | jq -r ".result.service")
        [ "$addr" != "null" ] && [ "$service" == "true" ]
        echo_rst "$FUNCNAME" "$?"
    fi
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
    r1=$(curl -ksd '{"method":"Chain33.GetBlockSequences","params":[{"start":1,"end":3,"isDetail":true}]}' ${MAIN_HTTP} | jq ".result.blkseqInfos|length==3")
    [ "$r1" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GetBlockByHashes() {
    if [ "$IS_PARA" == true ]; then
        geneis="0x97162f9d4a888121fdba2fb1ab402596acdbcb602121bd12284adb739d85f225"
        statehash=$(curl -ksd '{"method":"Chain33.GetBlockByHashes","params":[{"hashes":["'"$geneis"'"]}]}' ${MAIN_HTTP} | jq -r ".result.items[0].block.stateHash")
        [ "$statehash" == "0x2863c8dbc7fe3146c8d4e7acf2b8bbe4666264d658356e299e240f462a382a51" ]
        echo_rst "$FUNCNAME" "$?"
    else
        hash0=$(curl -ksd '{"method":"Chain33.GetBlockSequences","params":[{"start":1,"end":3,"isDetail":true}]}' ${MAIN_HTTP} | jq -r ".result.blkseqInfos[0].hash")
        hash1=$(curl -ksd '{"method":"Chain33.GetBlockSequences","params":[{"start":1,"end":3,"isDetail":true}]}' ${MAIN_HTTP} | jq -r ".result.blkseqInfos[1].hash")
        hash2=$(curl -ksd '{"method":"Chain33.GetBlockSequences","params":[{"start":1,"end":3,"isDetail":true}]}' ${MAIN_HTTP} | jq -r ".result.blkseqInfos[2].hash")

        # curl -ksd '{"method":"Chain33.GetBlockByHashes","params":[{"hashes":["'$hash1'","'$hash2'"]}]}'  ${MAIN_HTTP}
        # shellcheck disable=SC2086
        p1=$(curl -ksd '{"method":"Chain33.GetBlockByHashes","params":[{"hashes":["'"$hash1"'","'"$hash2"'"]}]}' ${MAIN_HTTP} | jq -r ".result.items[0].block.parentHash")
        # shellcheck disable=SC2086
        p2=$(curl -ksd '{"method":"Chain33.GetBlockByHashes","params":[{"hashes":["'"$hash1"'","'"$hash2"'"]}]}' ${MAIN_HTTP} | jq -r ".result.items[1].block.parentHash")
        [ "$p1" == "$hash0" ] && [ "$p2" == "$hash1" ]
        echo_rst "$FUNCNAME" "$?"
    fi
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
    if [ "$IS_PARA" == true ]; then
        echo_rst "$FUNCNAME" 2
    else
        r1=$(curl -ksd '{"method":"Chain33.AddSeqCallBack","params":[{"name":"test","url":"http://test","encode":"json"}]}' ${MAIN_HTTP} | jq -r ".result.isOK")
        [ "$r1" == "true" ]
        echo_rst "$FUNCNAME" "$?"
    fi
}

chain33_ListSeqCallBack() {
    if [ "$IS_PARA" == true ]; then
        echo_rst "$FUNCNAME" 2
    else
        r1=$(curl -ksd '{"method":"Chain33.ListSeqCallBack","params":[]}' ${MAIN_HTTP} | jq -r ".result.items[0].name")
        [ "$r1" == "test" ]
        echo_rst "$FUNCNAME" "$?"
    fi
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

chain33_GetHexTxByHash() {
    #先获取一笔交易
    reHash=$(curl -ksd '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetTxByAddr","params":[{"addr":"14KEKbYtKKQm4wMthSK9J4La4nAiidGozt","flag":0,"count":1,"direction":0,"height":-1,"index":0}]}' -H 'content-type:text/plain;' ${MAIN_HTTP} | jq -r '.result.txInfos[0].hash')
    #查询交易
    resp=$(curl -ksd '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetHexTxByHash","params":[{"hash":"'"$reHash"'","upgrade":false}]}' -H 'content-type:text/plain;' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result != null)' <<<"$resp")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_QueryTransaction() {
    #先获取一笔交易
    reHash=$(curl -ksd '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetTxByAddr","params":[{"addr":"14KEKbYtKKQm4wMthSK9J4La4nAiidGozt","flag":0,"count":1,"direction":0,"height":-1,"index":0}]}' -H 'content-type:text/plain;' ${MAIN_HTTP} | jq -r '.result.txInfos[0].hash')
    #查询交易
    resp=$(curl -ksd '{"jsonrpc":"2.0","id":2,"method":"Chain33.QueryTransaction","params":[{"hash":"'"$reHash"'","upgrade":false}]}' -H 'content-type:text/plain;' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result.receipt.tyName == "ExecOk") and (.result.height >= 0) and (.result.index >= 0) and (.result.amount >= 0)' <<<"$resp")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_GetBlocks() {
    result=$(curl -ksd '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetBlocks","params":[{"start":1,"end":2}]}' -H 'content-type:text/plain;' ${MAIN_HTTP} | jq -r ".result.items[1].block.height")
    [ "$result" -eq 2 ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_GetLastHeader() {
    resp=$(curl -ksd '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetLastHeader","params":[{}]}' -H 'content-type:text/plain;' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result.height >= 0) and (.result | [has("version","parentHash", "txHash", "stateHash", "height", "blockTime", "txCount", "hash", "difficulty"),true] | unique | length == 1)' <<<"$resp")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_GetTxByAddr() {
    resp=$(curl -ksd '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetTxByAddr","params":[{"addr":"14KEKbYtKKQm4wMthSK9J4La4nAiidGozt","flag":0,"count":1,"direction":0,"height":-1,"index":0}]}' -H 'content-type:text/plain;' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result.txInfos[0].index >= 0) and (.result.txInfos[0] | [has("hash", "height", "index", "assets"),true] | unique | length == 1)' <<<"$resp")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_GetTxByHashes() {
    resp=$(curl -ksd '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetTxByHashes","params":[{"hashes":["0x8040109d3859827d0f0c80ce91cc4ec80c496c45250f5e5755064b6da60842ab","0x501b910fd85d13d1ab7d776bce41a462f27c4bfeceb561dc47f0a11b10f452e4"]}]}' -H 'content-type:text/plain;' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result.txs|length == 2)' <<<"$resp")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_GetMempool() {
    resp=$(curl -ksd '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetMempool","params":[{}]}' -H 'content-type:text/plain;' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result.txs|length >= 0)' <<<"$resp")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_GetAccountsV2() {
    resp=$(curl -ksd '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetAccountsV2","params":[{}]}' -H 'content-type:text/plain;' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result.wallets|length >= 0)' <<<"$resp")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_GetAccounts() {
    resp=$(curl -ksd '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetAccounts","params":[{}]}' -H 'content-type:text/plain;' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result.wallets|length >= 0)' <<<"$resp")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_NewAccount() {
    resp=$(curl -ksd '{"jsonrpc":"2.0","id":2,"method":"Chain33.NewAccount","params":[{"label":"test169"}]}' -H 'content-type:text/plain;' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result.label == "test169") and (.result.acc | [has("addr"),true] | unique | length == 1)' <<<"$resp")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

# hyb

chain33_CreateRawTransaction() {
    local to="1EDDghAtgBsamrNEtNmYdQzC1QEhLkr87t"
    local exec="coins"
    local amount=10000000
    tx=$(curl -ksd '{"method":"Chain33.CreateRawTransaction","params":[{"to":"'$to'","amount":'$amount'}]}' ${MAIN_HTTP} | jq -r ".result")

    data=$(curl -ksd '{"method":"Chain33.DecodeRawTransaction","params":[{"txHex":"'"$tx"'"}]}' ${MAIN_HTTP} | jq -r ".result.txs[0]")
    ok=$(jq '(.execer == "'$exec'") and (.to == "'$to'")' <<<"$data")

    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_CreateTransaction() {
    local to="1EDDghAtgBsamrNEtNmYdQzC1QEhLkr87t"
    local exec="coins"
    local amount=10000000

    tx=$(curl -ksd '{"method":"Chain33.CreateTransaction","params":[{"execer":"coins","actionName":"Transfer","payload":{"to":"'$to'", "amount":'$amount'}}]}' ${MAIN_HTTP} | jq -r ".result")

    data=$(curl -ksd '{"method":"Chain33.DecodeRawTransaction","params":[{"txHex":"'"$tx"'"}]}' ${MAIN_HTTP} | jq -r ".result.txs[0]")
    ok=$(jq '(.execer == "'$exec'") and ((.payload.transfer.to) == "'$to'")' <<<"$data")

    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_ReWriteRawTx() {
    local fee=1000000
    local exec="coins"
    local to="1EDDghAtgBsamrNEtNmYdQzC1QEhLkr87t"
    local tx1="0a05636f696e73122d18010a291080ade20422223145444467684174674273616d724e45744e6d5964517a43315145684c6b7238377420a08d0630f6db93c0e0d3f1ff5e3a223145444467684174674273616d724e45744e6d5964517a43315145684c6b72383774"
    tx=$(curl -ksd '{"method":"Chain33.ReWriteRawTx","params":[{"expire":"120s","fee":'$fee',"tx":"'$tx1'"}]}' ${MAIN_HTTP} | jq -r ".result")

    data=$(curl -ksd '{"method":"Chain33.DecodeRawTransaction","params":[{"txHex":"'"$tx"'"}]}' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result.txs[0].execer == "'$exec'") and (.result.txs[0].to == "'$to'") and (.result.txs[0].fee == '$fee')' <<<"$data")

    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_CreateRawTxGroup() {
    local to="1DNaSDRG9RD19s59meAoeN4a2F6RH97fSo"
    local exec="user.write"
    local groupCount=2

    fee=1000000
    tx1="0a0a757365722e7772697465121d236d642368616b6468676f7177656a6872676f716a676f6a71776c6a6720a08d0630a0b7b1b1dda2f4c5743a2231444e615344524739524431397335396d65416f654e34613246365248393766536f"
    tx2="0a0a757365722e7772697465121d236d642368616b6468676f7177656a6872676f716a676f6a71776c6a6720a08d0630c5838f94e2f49acb4b3a2231444e615344524739524431397335396d65416f654e34613246365248393766536f"
    tx=$(curl -ksd '{"method":"Chain33.CreateRawTxGroup","params":[{"txs":["'$tx1'","'$tx2'"]}]}' ${MAIN_HTTP} | jq -r ".result")

    data=$(curl -ksd '{"method":"Chain33.DecodeRawTransaction","params":[{"txHex":"'"$tx"'"}]}' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result.txs[0].execer == "'$exec'") and (.result.txs[0].to == "'$to'") and (.result.txs[0].groupCount == '$groupCount') and (.result.txs[1].execer == "'$exec'") and (.result.txs[1].to == "'$to'") and (.result.txs[1].groupCount == '$groupCount')' <<<"$data")

    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_SignRawTx() {
    local fee=1000000
    local exec="coins"
    local to="1EDDghAtgBsamrNEtNmYdQzC1QEhLkr87t"
    local from="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
    local privkey="CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944"

    tx1="0a05636f696e73122d18010a291080ade20422223145444467684174674273616d724e45744e6d5964517a43315145684c6b7238377420a08d0628e1ddcae60530f6db93c0e0d3f1ff5e3a223145444467684174674273616d724e45744e6d5964517a43315145684c6b72383774"
    tx=$(curl -ksd '{"method":"Chain33.SignRawTx","params":[{"expire":"120s","fee":'$fee',"privkey":"'$privkey'","txHex":"'$tx1'"}]}' ${MAIN_HTTP} | jq -r ".result")

    data=$(curl -ksd '{"method":"Chain33.DecodeRawTransaction","params":[{"txHex":"'"$tx"'"}]}' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result.txs[0].execer == "'$exec'") and (.result.txs[0].to == "'$to'") and (.result.txs[0].fee == '$fee') and (.result.txs[0].from == "'$from'")' <<<"$data")

    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"

}

chain33_SendTransaction() {
    local fee=1000000
    local exec="coins"
    local to="1EDDghAtgBsamrNEtNmYdQzC1QEhLkr87t"
    local from="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
    local privkey="CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944"

    tx1="0a05636f696e73122d18010a291080ade20422223145444467684174674273616d724e45744e6d5964517a43315145684c6b7238377420a08d0628e1ddcae60530f6db93c0e0d3f1ff5e3a223145444467684174674273616d724e45744e6d5964517a43315145684c6b72383774"
    tx=$(curl -ksd '{"method":"Chain33.SignRawTx","params":[{"expire":"120s","fee":'$fee',"privkey":"'$privkey'","txHex":"'$tx1'"}]}' ${MAIN_HTTP} | jq -r ".result")

    data=$(curl -ksd '{"method":"Chain33.SendTransaction","params":[{"data":"'"$tx"'"}]}' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result != null)' <<<"$data")

    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_CreateNoBalanceTransaction() {
    local to="1EDDghAtgBsamrNEtNmYdQzC1QEhLkr87t"
    local txHex="0a05636f696e73122d18010a291080ade20422223145444467684174674273616d724e45744e6d5964517a43315145684c6b7238377420a08d0630a1938af2e88e97fb0d3a223145444467684174674273616d724e45744e6d5964517a43315145684c6b72383774"

    tx=$(curl -ksd '{"method":"Chain33.CreateNoBalanceTransaction","params":[{"txHex":"'$txHex'"}]}' ${MAIN_HTTP} | jq -r ".result")

    data=$(curl -ksd '{"method":"Chain33.DecodeRawTransaction","params":[{"txHex":"'"$tx"'"}]}' ${MAIN_HTTP})
    ok=$(jq '(.error|not) and (.result.txs[0].execer == "none") and (.result.txs[0].groupCount == 2) and (.result.txs[1].execer == "coins") and (.result.txs[1].groupCount == 2) and (.result.txs[1].to == "'$to'")' <<<"$data")

    [ "$ok" == true ]
    rst=$?
    echo_rst "$FUNCNAME" "$rst"
}

chain33_GetBlockHash() {
    req='{"method":"Chain33.GetBlockHash", "params":[{"height":1}]}'
    resp=$(curl -ksd "$req" "${MAIN_HTTP}")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result| has("hash"))' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GenSeed() {
    req='{"method":"Chain33.GenSeed", "params":[{"lang":0}]}'
    resp=$(curl -ksd "$req" "${MAIN_HTTP}")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result| has("seed"))' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
    seed=$(jq '(.result.seed)' <<<"$resp")
}

chain33_SaveSeed() {
    req='{"method":"Chain33.SaveSeed", "params":[{"seed":'"$seed"', "passwd": "1314fuzamei"}]}'
    resp=$(curl -ksd "$req" "${MAIN_HTTP}")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result| has("isOK"))' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GetSeed() {
    req='{"method":"Chain33.GetSeed", "params":[{"passwd": "1314fuzamei"}]}'
    resp=$(curl -ksd "$req" "${MAIN_HTTP}")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result| has("seed"))' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_testSeed() {
    seed=""
    chain33_GenSeed
    chain33_SaveSeed
    chain33_GetSeed
}

chain33_GetWalletStatus() {
    req='{"method":"Chain33.GetWalletStatus", "params":[{}]}'
    resp=$(curl -ksd "$req" "${MAIN_HTTP}")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result| [has("isWalletLock", "isAutoMining", "isHasSeed", "isTicketLock"), true] | unique | length == 1)' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GetBalance() {
    req='{"method":"Chain33.GetBalance", "params":[{"addresses" : ["14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"], "execer" : "coins"}]}'
    resp=$(curl -ksd "$req" "${MAIN_HTTP}")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result[0] | [has("balance", "frozen"), true] | unique | length == 1)' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GetAllExecBalance() {
    req='{"method":"Chain33.GetAllExecBalance", "params":[{"addr" : "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"}]}'
    resp=$(curl -ksd "$req" "${MAIN_HTTP}")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result| [has("addr", "execAccount"), true] | unique | length == 1)' <<<"$resp")
    [ "$ok" == true ]
    ok=$(jq '(.result.execAccount |  [map(has("execer", "account")), true] | flatten | unique | length == 1)' <<<"$resp")
    [ "$ok" == true ]
    ok=$(jq '(.result.execAccount[].account |  [.] | [map(has("balance", "frozen")), true] | flatten | unique | length == 1)' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_ExecWallet() {
    req='{"method":"Chain33.ExecWallet", "params":[{"funcName" : "NewAccountByIndex", "payload" : {"data" : 100000009}, "stateHash" : "", "execer" : "wallet" }]}'
    resp=$(curl -ksd "$req" "${MAIN_HTTP}")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result | has("data"))' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_Query() {
    req='{"method":"Chain33.Query", "params":[{ "execer":"coins", "funcName": "GetTxsByAddr", "payload" : {"addr" : "1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4"}}]}'
    resp=$(curl -ksd "$req" "${MAIN_HTTP}")
    #    echo "#response: $resp"
    ok=$(jq '(. | has("result"))' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_Version() {
    req='{"method":"Chain33.Version", "params":[{}]}'
    resp=$(curl -ksd "$req" "${MAIN_HTTP}")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result)' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_GetTotalCoins() {
    req='{"method":"Chain33.GetTotalCoins", "params":[{"symbol" : "bty", "stateHash":"", "startKey":"", "count":2, "execer":"coins"}]}'
    resp=$(curl -ksd "$req" "${MAIN_HTTP}")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (.result| has("count"))' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_IsSync() {
    req='{"method":"Chain33.IsSync", "params":[{}]}'
    resp=$(curl -ksd "$req" "${MAIN_HTTP}")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (. | has("result"))' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

chain33_IsNtpClockSync() {
    req='{"method":"Chain33.IsNtpClockSync", "params":[{}]}'
    resp=$(curl -ksd "$req" "${MAIN_HTTP}")
    #    echo "#response: $resp"
    ok=$(jq '(.error|not) and (. | has("result"))' <<<"$resp")
    [ "$ok" == true ]
    echo_rst "$FUNCNAME" "$?"
}

run_testcases() {
    #    set -x
    set +e
    IS_PARA=$(echo '"'"${1}"'"' | jq '.|contains("8901")')
    echo "ipara=$IS_PARA"

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

    chain33_CreateRawTransaction
    chain33_CreateTransaction
    chain33_ReWriteRawTx
    chain33_CreateRawTxGroup
    chain33_SignRawTx
    chain33_SendTransaction
    chain33_CreateNoBalanceTransaction

    chain33_GetBlockHash
    chain33_testSeed
    chain33_GetWalletStatus
    chain33_GetBalance
    chain33_GetAllExecBalance
    chain33_ExecWallet
    chain33_Query
    chain33_Version
    chain33_GetTotalCoins
    chain33_IsSync
    chain33_IsNtpClockSync

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

#system_test_rpc $1
