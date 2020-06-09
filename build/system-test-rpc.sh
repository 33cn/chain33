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
        echo -e "${RED}$3 ${NOC}"
        CASE_ERR="err"
    fi
}

http_req() {
    #  echo "#$4 request: $1"
    body=$(curl -ksd "$1" "$2")
    #  echo "#response: $body"
    ok=$(echo "$body" | jq -r "$3")
    [ "$ok" == true ]
    rst=$?
    echo_rst "$4" "$rst" "$body"
}

chain33_lock() {
    http_req '{"method":"Chain33.Lock","params":[]}' ${MAIN_HTTP} ".result.isOK" "$FUNCNAME"
}

chain33_unlock() {
    http_req '{"method":"Chain33.UnLock","params":[{"passwd":"1314fuzamei","timeout":0}]}' ${MAIN_HTTP} ".result.isOK" "$FUNCNAME"
}

chain33_WalletTxList() {
    req='{"method":"Chain33.WalletTxList", "params":[{"fromTx":"", "count":2, "direction":1}]}'
    http_req "$req" ${MAIN_HTTP} '(.error|not) and (.result.txDetails|length == 2)' "$FUNCNAME"
}

chain33_ImportPrivkey() {
    req='{"method":"Chain33.ImportPrivkey", "params":[{"privkey":"0x88b2fb90411935872f0501dd13345aba19b5fac9b00eb0dddd7df977d4d5477e", "label":"testimportkey"}]}'
    http_req "$req" ${MAIN_HTTP} '(.error|not) and (.result.label=="testimportkey") and (.result.acc.addr == "1D9xKRnLvV2zMtSxSx33ow1GF4pcbLcNRt")' "$FUNCNAME"
}

chain33_DumpPrivkey() {
    req='{"method":"Chain33.DumpPrivkey", "params":[{"data":"1D9xKRnLvV2zMtSxSx33ow1GF4pcbLcNRt"}]}'
    http_req "$req" ${MAIN_HTTP} '(.error|not) and (.result.data=="0x88b2fb90411935872f0501dd13345aba19b5fac9b00eb0dddd7df977d4d5477e")' "$FUNCNAME"
}

chain33_DumpPrivkeysFile() {
    req='{"method":"Chain33.DumpPrivkeysFile", "params":[{"fileName":"PrivkeysFile","passwd":"123456"}]}'
    http_req "$req" ${MAIN_HTTP} '(.error|not) and .result.isOK' "$FUNCNAME"
}

chain33_ImportPrivkeysFile() {
    req='{"method":"Chain33.ImportPrivkeysFile", "params":[{"fileName":"PrivkeysFile","passwd":"123456"}]}'
    http_req "$req" ${MAIN_HTTP} '(.error|not) and .result.isOK' "$FUNCNAME"
    # rm -rf ./PrivkeysFile
}

chain33_SendToAddress() {
    req='{"method":"Chain33.SendToAddress", "params":[{"from":"12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv","to":"1D9xKRnLvV2zMtSxSx33ow1GF4pcbLcNRt", "amount":100000000, "note":"test\n"}]}'
    http_req "$req" ${MAIN_HTTP} '(.error|not) and (.result.hash|length==66)' "$FUNCNAME"
}

chain33_SetTxFee() {
    http_req '{"method":"Chain33.SetTxFee", "params":[{"amount":100000}]}' ${MAIN_HTTP} '(.error|not) and .result.isOK' "$FUNCNAME"
}

chain33_SetLabl() {
    req='{"method":"Chain33.SetLabl", "params":[{"addr":"1D9xKRnLvV2zMtSxSx33ow1GF4pcbLcNRt", "label":"updatetestimport"}]}'
    http_req "$req" ${MAIN_HTTP} '(.error|not) and (.result.label=="updatetestimport") and (.result.acc.addr == "1D9xKRnLvV2zMtSxSx33ow1GF4pcbLcNRt")' "$FUNCNAME"
}

chain33_GetPeerInfo() {
    if [ "$IS_PARA" == true ]; then
        echo_rst "$FUNCNAME" 2
    else
        resok='(.error|not) and (.result.peers|length >= 1) and (.result.peers[0] | [has("addr", "port", "name", "mempoolSize", "self", "header"), true] | unique | length == 1)'
        http_req '{"method":"Chain33.GetPeerInfo", "params":[{}]}' ${MAIN_HTTP} "$resok" "$FUNCNAME"
    fi
}

chain33_GetHeaders() {
    resok='(.error|not) and (.result.items|length == 2) and (.result.items[0] | [has("version","parentHash", "txHash", "stateHash", "height", "blockTime", "txCount", "hash", "difficulty"),true] | unique | length == 1 )'
    http_req '{"method":"Chain33.GetHeaders", "params":[{"start":1, "end":2, "isDetail":true}]}' ${MAIN_HTTP} "$resok" "$FUNCNAME"
}

chain33_GetLastMemPool() {
    if [ "$IS_PARA" == true ]; then
        echo_rst "$FUNCNAME" 2
    else
        http_req '{"method":"Chain33.GetLastMemPool", "params":[{}]}' ${MAIN_HTTP} '(.error|not) and (.result.txs|length >= 0)' "$FUNCNAME"
    fi
}

chain33_GetProperFee() {
    http_req '{"method":"Chain33.GetProperFee", "params":[{}]}' ${MAIN_HTTP} '(.error|not) and (.result.properFee > 10000)' "$FUNCNAME"
}

chain33_GetBlockOverview() {
    hash=$(curl -ksd '{"method":"Chain33.GetHeaders", "params":[{"start":1, "end":1, "isDetail":true}]}' ${MAIN_HTTP} | jq '.result.items[0].hash')
    req='{"method":"Chain33.GetBlockOverview", "params":[{"hash":'"$hash"'}]}'
    http_req "$req" ${MAIN_HTTP} '(.error|not) and (.result| [has("head", "txCount", "txHashes"), true]|unique|length == 1) and (.result.txCount == (.result.txHashes|length))' "$FUNCNAME"
}

chain33_GetAddrOverview() {
    req='{"method":"Chain33.GetAddrOverview", "params":[{"addr":"12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"}]}'
    http_req "$req" ${MAIN_HTTP} '(.error|not) and (.result|[has("reciver", "balance", "txCount"), true]|unique|length == 1)' "$FUNCNAME"
}

chain33_SetPasswd() {
    http_req '{"method":"Chain33.SetPasswd", "params":[{"oldPass":"1314fuzamei", "newPass":"1314fuzamei"}]}' ${MAIN_HTTP} '(.error|not) and .result.isOK' "$FUNCNAME"
}

chain33_MergeBalance() {
    http_req '{"method":"Chain33.MergeBalance", "params":[{"to":"12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"}]}' ${MAIN_HTTP} '(.error|not) and (.result.hashes|length > 0)' "$FUNCNAME"
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

    req='{"method":"Chain33.QueryTotalFee","params":[{"keys":["'$base64_hash'"]}]}'
    http_req "$req" ${MAIN_HTTP} '(.result.txCount >= 0)' "$FUNCNAME"
}

chain33_GetNetInfo() {
    if [ "$IS_PARA" == true ]; then
        echo_rst "$FUNCNAME" 2
    else
        http_req '{"method":"Chain33.GetNetInfo", "params":[]}' ${MAIN_HTTP} '(.result.externalAddr| length > 0)' "$FUNCNAME"
    fi
}

chain33_GetFatalFailure() {
    http_req '{"method":"Chain33.GetFatalFailure", "params":[]}' ${MAIN_HTTP} '(.error|not) and (.result | 0)' "$FUNCNAME"
}

chain33_DecodeRawTransaction() {
    tx="0a05636f696e73122c18010a281080c2d72f222131477444795771577233553637656a7663776d333867396e7a6e7a434b58434b7120a08d0630a696c0b3f78dd9ec083a2131477444795771577233553637656a7663776d333867396e7a6e7a434b58434b71"
    http_req '{"method":"Chain33.DecodeRawTransaction", "params":[{"txHex":"'$tx'"}]}' ${MAIN_HTTP} '(.result.txs[0].execer == "coins")' "$FUNCNAME"
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
    if [ "$IS_PARA" == true ]; then
        echo_rst "$FUNCNAME" 2
    else
        http_req '{"method":"Chain33.GetLastBlockSequence","params":[]}' ${MAIN_HTTP} ".result >= 0" "$FUNCNAME"
    fi
}

chain33_GetBlockSequences() {
    if [ "$IS_PARA" == true ]; then
        echo_rst "$FUNCNAME" 2
    else
        http_req '{"method":"Chain33.GetBlockSequences","params":[{"start":1,"end":3,"isDetail":true}]}' ${MAIN_HTTP} ".result.blkseqInfos|length==3" "$FUNCNAME"
    fi
}

chain33_GetBlockByHashes() {
    if [ "$IS_PARA" == true ]; then
        geneis=$(curl -ksd '{"method":"Chain33.GetBlockHash", "params":[{"height":0}]}' "${MAIN_HTTP}" | jq -r '(.result.hash)')
        req='{"method":"Chain33.GetBlockByHashes","params":[{"hashes":["'"${geneis}"'"]}]}'
        http_req "$req" ${MAIN_HTTP} '(.result.items[0].block.parentHash == "0x0000000000000000000000000000000000000000000000000000000000000000")' "$FUNCNAME"
    else
        hash0=$(curl -ksd '{"method":"Chain33.GetBlockSequences","params":[{"start":1,"end":3,"isDetail":true}]}' ${MAIN_HTTP} | jq -r ".result.blkseqInfos[0].hash")
        hash1=$(curl -ksd '{"method":"Chain33.GetBlockSequences","params":[{"start":1,"end":3,"isDetail":true}]}' ${MAIN_HTTP} | jq -r ".result.blkseqInfos[1].hash")
        hash2=$(curl -ksd '{"method":"Chain33.GetBlockSequences","params":[{"start":1,"end":3,"isDetail":true}]}' ${MAIN_HTTP} | jq -r ".result.blkseqInfos[2].hash")

        req='{"method":"Chain33.GetBlockByHashes","params":[{"hashes":["'"$hash1"'","'"$hash2"'"]}]}'
        resok='(.result.items[0].block.parentHash == "'"$hash0"'") and (.result.items[1].block.parentHash =="'"$hash1"'")'
        http_req "$req" ${MAIN_HTTP} "$resok" "$FUNCNAME"
    fi
}

chain33_ConvertExectoAddr() {
    http_req '{"method":"Chain33.ConvertExectoAddr","params":[{"execname":"coins"}]}' ${MAIN_HTTP} '(.result == "1GaHYpWmqAJsqRwrpoNcB8VvgKtSwjcHqt")' "$FUNCNAME"
}

chain33_GetExecBalance() {
    local height=6802
    statehash=$(curl -ksd '{"method":"Chain33.GetBlocks","params":[{"start":'$height',"end":'$height',"isDetail":false}]}' ${MAIN_HTTP} | jq -r ".result.items[0].block.stateHash")
    state_base64=$(echo -n "$statehash" | cut -d " " -f 1 | xxd -r -p | base64)
    addr="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
    addr_base64=$(echo -n "$addr" | base64)

    req='{"method":"Chain33.GetExecBalance","params":[{"symbol":"bty","stateHash":"'$state_base64'","addr":"'$addr_base64'","execer":"coins","count":100}]}'
    http_req "$req" ${MAIN_HTTP} "(.error|not)" "$FUNCNAME"
}

chain33_AddPushSubscribe() {
    http_req '{"method":"Chain33.AddPushSubscribe","params":[{"name":"test","url":"http://test","encode":"json"}]}' ${MAIN_HTTP} '(.result.isOk == true)' "$FUNCNAME"
}

chain33_ListPushes() {
    http_req '{"method":"Chain33.ListPushes","params":[]}' ${MAIN_HTTP} ' (.result.pushes[0].name == "test")' "$FUNCNAME"
}

chain33_GetPushSeqLastNum() {
    http_req '{"method":"Chain33.GetPushSeqLastNum","params":[{"data":"test-another"}]}' ${MAIN_HTTP} '(.result.data == -1)' "$FUNCNAME"
}

chain33_GetCoinSymbol() {
    symbol="bty"
    if [ "$IS_PARA" == true ]; then
        symbol="para"
    fi

    resok='(.result.data == "'"$symbol"'")'
    http_req '{"method":"Chain33.GetCoinSymbol","params":[]}' ${MAIN_HTTP} "$resok" "$FUNCNAME"
}

chain33_GetHexTxByHash() {
    #先获取一笔交易
    reHash=$(curl -ksd '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetTxByAddr","params":[{"addr":"14KEKbYtKKQm4wMthSK9J4La4nAiidGozt","flag":0,"count":1,"direction":0,"height":-1,"index":0}]}' -H 'content-type:text/plain;' ${MAIN_HTTP} | jq -r '.result.txInfos[0].hash')
    #查询交易
    req='{"method":"Chain33.GetHexTxByHash","params":[{"hash":"'"$reHash"'","upgrade":false}]}'
    http_req "$req" ${MAIN_HTTP} '(.error|not) and (.result != null)' "$FUNCNAME"
}

chain33_QueryTransaction() {
    #先获取一笔交易
    reHash=$(curl -ksd '{"jsonrpc":"2.0","id":2,"method":"Chain33.GetTxByAddr","params":[{"addr":"14KEKbYtKKQm4wMthSK9J4La4nAiidGozt","flag":0,"count":1,"direction":0,"height":-1,"index":0}]}' -H 'content-type:text/plain;' ${MAIN_HTTP} | jq -r '.result.txInfos[0].hash')
    #查询交易
    req='{"method":"Chain33.QueryTransaction","params":[{"hash":"'"$reHash"'","upgrade":false}]}'
    http_req "$req" ${MAIN_HTTP} '(.error|not) and (.result.receipt.tyName == "ExecOk") and (.result.height >= 0) and (.result.index >= 0) and (.result.amount >= 0)' "$FUNCNAME"
}

chain33_GetBlocks() {
    http_req '{"method":"Chain33.GetBlocks","params":[{"start":1,"end":2}]}' ${MAIN_HTTP} '(.result.items[1].block.height == 2)' "$FUNCNAME"
}

chain33_GetLastHeader() {
    resok='(.error|not) and (.result.height >= 0) and (.result | [has("version","parentHash", "txHash", "stateHash", "height", "blockTime", "txCount", "hash", "difficulty"),true] | unique | length == 1)'
    http_req '{"method":"Chain33.GetLastHeader","params":[{}]}' ${MAIN_HTTP} "$resok" "$FUNCNAME"
}

chain33_GetTxByAddr() {
    req='{"method":"Chain33.GetTxByAddr","params":[{"addr":"14KEKbYtKKQm4wMthSK9J4La4nAiidGozt","flag":0,"count":1,"direction":0,"height":-1,"index":0}]}'
    resok='(.error|not) and (.result.txInfos[0].index >= 0) and (.result.txInfos[0] | [has("hash", "height", "index", "assets"),true] | unique | length == 1)'
    http_req "$req" ${MAIN_HTTP} "$resok" "$FUNCNAME"
}

chain33_GetTxByHashes() {
    req='{"method":"Chain33.GetTxByHashes","params":[{"hashes":["0x8040109d3859827d0f0c80ce91cc4ec80c496c45250f5e5755064b6da60842ab","0x501b910fd85d13d1ab7d776bce41a462f27c4bfeceb561dc47f0a11b10f452e4"]}]}'
    http_req "$req" ${MAIN_HTTP} '(.error|not) and (.result.txs|length == 2)' "$FUNCNAME"
}

chain33_GetMempool() {
    if [ "$IS_PARA" == true ]; then
        echo_rst "$FUNCNAME" 2
    else
        http_req '{"method":"Chain33.GetMempool","params":[{}]}' ${MAIN_HTTP} '(.error|not) and (.result.txs|length >= 0)' "$FUNCNAME"
    fi
}

chain33_GetAccountsV2() {
    http_req '{"method":"Chain33.GetAccountsV2","params":[{}]}' ${MAIN_HTTP} '(.error|not) and (.result.wallets|length >= 0)' "$FUNCNAME"
}

chain33_GetAccounts() {
    http_req '{"method":"Chain33.GetAccounts","params":[{}]}' ${MAIN_HTTP} '(.error|not) and (.result.wallets|length >= 0)' "$FUNCNAME"
}

chain33_NewAccount() {
    http_req '{"method":"Chain33.NewAccount","params":[{"label":"test169"}]}' ${MAIN_HTTP} '(.error|not) and (.result.label == "test169") and (.result.acc | [has("addr"),true] | unique | length == 1)' "$FUNCNAME"
}

# hyb
chain33_CreateRawTransaction() {
    local to="1EDDghAtgBsamrNEtNmYdQzC1QEhLkr87t"
    local amount=10000000
    tx=$(curl -ksd '{"method":"Chain33.CreateRawTransaction","params":[{"to":"'$to'","amount":'$amount'}]}' ${MAIN_HTTP} | jq -r ".result")

    req='{"method":"Chain33.DecodeRawTransaction","params":[{"txHex":"'"$tx"'"}]}'
    resok='(.result.txs[0].payload.transfer.amount == "'$amount'") and (.result.txs[0].to == "'$to'")'
    http_req "$req" ${MAIN_HTTP} "$resok" "$FUNCNAME"
}

chain33_CreateTransaction() {
    local to="1EDDghAtgBsamrNEtNmYdQzC1QEhLkr87t"
    local amount=10000000
    local exec=""

    if [ "$IS_PARA" == true ]; then
        exec="user.p.para.coins"
    else
        exec="coins"
    fi

    tx=$(curl -ksd '{"method":"Chain33.CreateTransaction","params":[{"execer":"'$exec'","actionName":"Transfer","payload":{"to":"'$to'", "amount":'$amount'}}]}' ${MAIN_HTTP} | jq -r ".result")

    req='{"method":"Chain33.DecodeRawTransaction","params":[{"txHex":"'"$tx"'"}]}'
    resok='(.result.txs[0].payload.transfer.amount == "'$amount'") and (.result.txs[0].payload.transfer.to == "'$to'")'
    http_req "$req" ${MAIN_HTTP} "$resok" "$FUNCNAME"
}

chain33_ReWriteRawTx() {
    local fee=1000000
    local tx1="0a05636f696e73122d18010a291080ade20422223145444467684174674273616d724e45744e6d5964517a43315145684c6b7238377420a08d0630f6db93c0e0d3f1ff5e3a223145444467684174674273616d724e45744e6d5964517a43315145684c6b72383774"
    tx=$(curl -ksd '{"method":"Chain33.ReWriteRawTx","params":[{"expire":"120s","fee":'$fee',"tx":"'$tx1'"}]}' ${MAIN_HTTP} | jq -r ".result")

    req='{"method":"Chain33.DecodeRawTransaction","params":[{"txHex":"'"$tx"'"}]}'
    http_req "$req" ${MAIN_HTTP} '(.error|not) and (.result.txs[0].execer == "coins") and (.result.txs[0].to == "1EDDghAtgBsamrNEtNmYdQzC1QEhLkr87t") and (.result.txs[0].fee == '$fee')' "$FUNCNAME"
}

chain33_CreateRawTxGroup() {
    local to="1DNaSDRG9RD19s59meAoeN4a2F6RH97fSo"
    local exec="user.write"
    local groupCount=2
    tx1="0a0a757365722e7772697465121d236d642368616b6468676f7177656a6872676f716a676f6a71776c6a6720a08d0630a0b7b1b1dda2f4c5743a2231444e615344524739524431397335396d65416f654e34613246365248393766536f"
    tx2="0a0a757365722e7772697465121d236d642368616b6468676f7177656a6872676f716a676f6a71776c6a6720a08d0630c5838f94e2f49acb4b3a2231444e615344524739524431397335396d65416f654e34613246365248393766536f"
    tx=$(curl -ksd '{"method":"Chain33.CreateRawTxGroup","params":[{"txs":["'$tx1'","'$tx2'"]}]}' ${MAIN_HTTP} | jq -r ".result")

    req='{"method":"Chain33.DecodeRawTransaction","params":[{"txHex":"'"$tx"'"}]}'
    resok='(.error|not) and (.result.txs[0].execer == "'$exec'") and (.result.txs[0].to == "'$to'") and (.result.txs[0].groupCount == '$groupCount') and (.result.txs[1].execer == "'$exec'") and (.result.txs[1].to == "'$to'") and (.result.txs[1].groupCount == '$groupCount')'
    http_req "$req" ${MAIN_HTTP} "$resok" "$FUNCNAME"
}

chain33_SignRawTx() {
    local fee=1000000
    local privkey="CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944"

    tx1="0a05636f696e73122d18010a291080ade20422223145444467684174674273616d724e45744e6d5964517a43315145684c6b7238377420a08d0628e1ddcae60530f6db93c0e0d3f1ff5e3a223145444467684174674273616d724e45744e6d5964517a43315145684c6b72383774"
    tx=$(curl -ksd '{"method":"Chain33.SignRawTx","params":[{"expire":"120s","fee":'$fee',"privkey":"'$privkey'","txHex":"'$tx1'"}]}' ${MAIN_HTTP} | jq -r ".result")

    req='{"method":"Chain33.DecodeRawTransaction","params":[{"txHex":"'"$tx"'"}]}'
    resok='(.error|not) and (.result.txs[0].execer == "coins") and (.result.txs[0].to == "1EDDghAtgBsamrNEtNmYdQzC1QEhLkr87t") and (.result.txs[0].fee == '$fee') and (.result.txs[0].from == "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt")'
    http_req "$req" ${MAIN_HTTP} "$resok" "$FUNCNAME"
}

chain33_SendTransaction() {
    local fee=1000000
    local exec="coins"
    local to="1EDDghAtgBsamrNEtNmYdQzC1QEhLkr87t"
    local privkey="CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944"

    tx1="0a05636f696e73122d18010a291080ade20422223145444467684174674273616d724e45744e6d5964517a43315145684c6b7238377420a08d0628e1ddcae60530f6db93c0e0d3f1ff5e3a223145444467684174674273616d724e45744e6d5964517a43315145684c6b72383774"
    tx=$(curl -ksd '{"method":"Chain33.SignRawTx","params":[{"expire":"120s","fee":'$fee',"privkey":"'$privkey'","txHex":"'$tx1'"}]}' ${MAIN_HTTP} | jq -r ".result")

    req='{"method":"Chain33.SendTransaction","params":[{"data":"'"$tx"'"}]}'
    http_req "$req" ${MAIN_HTTP} '(.error|not) and (.result != null)' "$FUNCNAME"
}

chain33_CreateNoBalanceTransaction() {
    local to="1EDDghAtgBsamrNEtNmYdQzC1QEhLkr87t"
    local txHex=""
    local exec=""
    local coinexec=""

    if [ "$IS_PARA" == true ]; then
        exec="user.p.para.none"
        coinexec="user.p.para.coins"
        txHex="0a11757365722e702e706172612e636f696e73122d18010a291080ade20422223145444467684174674273616d724e45744e6d5964517a43315145684c6b7238377420a08d0630e6cbfbf1a7bafcb8263a2231415662506538776f524a7a7072507a4575707735554262433259507331344a4354"
    else
        exec="none"
        coinexec="coins"
        txHex="0a05636f696e73122d18010a291080ade20422223145444467684174674273616d724e45744e6d5964517a43315145684c6b7238377420a08d0630a1938af2e88e97fb0d3a223145444467684174674273616d724e45744e6d5964517a43315145684c6b72383774"
    fi

    tx=$(curl -ksd '{"method":"Chain33.CreateNoBalanceTransaction","params":[{"txHex":"'$txHex'"}]}' ${MAIN_HTTP} | jq -r ".result")
    req='{"method":"Chain33.DecodeRawTransaction","params":[{"txHex":"'"$tx"'"}]}'
    resok='(.error|not) and (.result.txs[0].execer == "'$exec'") and (.result.txs[0].groupCount == 2) and (.result.txs[1].execer == "'$coinexec'") and (.result.txs[1].groupCount == 2) and (.result.txs[1].to == "'$to'")'
    http_req "$req" ${MAIN_HTTP} "$resok" "$FUNCNAME"
}

chain33_GetBlockHash() {
    http_req '{"method":"Chain33.GetBlockHash","params":[{"height":1}]}' ${MAIN_HTTP} '(.error|not) and (.result| has("hash"))' "$FUNCNAME"
}

chain33_GenSeed() {
    http_req '{"method":"Chain33.GenSeed", "params":[{"lang":0}]}' ${MAIN_HTTP} '(.error|not) and (.result| has("seed"))' "$FUNCNAME"
    seed=$(curl -ksd '{"method":"Chain33.GenSeed", "params":[{"lang":0}]}' ${MAIN_HTTP} | jq -r ".result.seed")
}

chain33_SaveSeed() {
    req='{"method":"Chain33.SaveSeed", "params":[{"seed":"'"$seed"'", "passwd": "1314fuzamei"}]}'
    http_req "$req" ${MAIN_HTTP} '(.error|not) and (.result| has("isOK"))' "$FUNCNAME"
}

chain33_GetSeed() {
    http_req '{"method":"Chain33.GetSeed", "params":[{"passwd": "1314fuzamei"}]}' ${MAIN_HTTP} '(.error|not) and (.result| has("seed"))' "$FUNCNAME"
}

chain33_testSeed() {
    seed=""
    chain33_GenSeed
    chain33_SaveSeed
    chain33_GetSeed
}

chain33_GetWalletStatus() {
    http_req '{"method":"Chain33.GetWalletStatus","params":[{}]}' ${MAIN_HTTP} '(.error|not) and (.result| [has("isWalletLock", "isAutoMining", "isHasSeed", "isTicketLock"), true] | unique | length == 1)' "$FUNCNAME"
}

chain33_GetBalance() {
    http_req '{"method":"Chain33.GetBalance","params":[{"addresses" : ["14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"], "execer" : "coins"}]}' ${MAIN_HTTP} '(.error|not) and (.result[0] | [has("balance", "frozen"), true] | unique | length == 1)' "$FUNCNAME"
}

chain33_GetAllExecBalance() {
    resok='(.error|not) and (.result| [has("addr", "execAccount"), true] | unique | length == 1) and (.result.execAccount | [map(has("execer", "account")), true] | flatten | unique | length == 1) and ([.result.execAccount[].account] | [map(has("balance", "frozen")), true] | flatten | unique | length == 1)'
    http_req '{"method":"Chain33.GetAllExecBalance", "params":[{"addr" : "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"}]}' ${MAIN_HTTP} "$resok" "$FUNCNAME"
}

chain33_ExecWallet() {
    req='{"method":"Chain33.ExecWallet", "params":[{"funcName" : "NewAccountByIndex", "payload" : {"data" : 100000009}, "stateHash" : "", "execer" : "wallet" }]}'
    http_req "$req" ${MAIN_HTTP} '(.error|not) and (.result | has("data"))' "$FUNCNAME"
}

chain33_Query() {
    http_req '{"method":"Chain33.Query", "params":[{ "execer":"coins", "funcName": "GetTxsByAddr", "payload" : {"addr" : "1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4"}}]}' ${MAIN_HTTP} '(. | has("result"))' "$FUNCNAME"
}

chain33_Version() {
    http_req '{"method":"Chain33.Version", "params":[{}]}' ${MAIN_HTTP} '(.error|not) and (.result)' "$FUNCNAME"
}

chain33_GetTotalCoins() {
    http_req '{"method":"Chain33.GetTotalCoins", "params":[{"symbol" : "bty", "stateHash":"", "startKey":"", "count":2, "execer":"coins"}]}' ${MAIN_HTTP} '(.error|not) and (.result| has("count"))' "$FUNCNAME"
}

chain33_IsSync() {
    http_req '{"method":"Chain33.IsSync", "params":[{}]}' ${MAIN_HTTP} '(.error|not) and (. | has("result"))' "$FUNCNAME"
}

chain33_IsNtpClockSync() {
    http_req '{"method":"Chain33.IsNtpClockSync", "params":[{}]}' ${MAIN_HTTP} '(.error|not) and (. | has("result"))' "$FUNCNAME"
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
    chain33_DumpPrivkeysFile "$1"
    chain33_ImportPrivkeysFile "$1"
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
    chain33_AddPushSubscribe
    chain33_ListPushes
    chain33_GetPushSeqLastNum
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
