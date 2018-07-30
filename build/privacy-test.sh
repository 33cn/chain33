#!/usr/bin/env bash

# 引用测试基础库
source comm-test.sh

containers=("${NODE1}" "${NODE2}" "${NODE3}" "${NODE4}" "${NODE5}" "${NODE6}")
forkContainers=("${CLI3}" "${CLI2}" "${CLI}" "${CLI4}" "${CLI5}" "${CLI6}")

password=1314
walletSeed="tortoise main civil member grace happy century convince father cage beach hip maid merry rib"
privExecAddr="1FeyE6VDZ4FYgpK1n2okWMDAtPkwBuooQd"

CLIfromAddr1="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
CLIprivKey1="0a9d212b2505aefaa8da370319088bbccfac097b007f52ed71d8133456c8185823c8eac43c5e937953d7b6c8e68b0db1f4f03df4946a29f524875118960a35fb"
CLIfromAddr2="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
CLIprivKey2="fcbb75f2b96b6d41f301f2d1abc853d697818427819f412f8e4b4e12cacc0814d2c3914b27bea9151b8968ed1732bd241c8788a332b295b731aee8d39a060388"
CLI4fromAddr1="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
CLI4privKey11="069fdcd7a2d7cf30dfc87df6f277ae451a78cae6720a6bb05514a4a43e0622d55c854169cc63b6353234c3e65db75e7b205878b1bd94e9f698c7043b27fa162b"
CLI4fromAddr2="1KcCVZLSQYRUwE5EXTsAoQs9LuJW6xwfQa"
CLI4privKey12="d5672eeafbcdf53c8fc27969a5d9797083bb64fb4848bd391cd9b3919c4a1d3cb8534f12e09de3cc541eaaf45acccacaf808a6804fd10a976804397e9ecaf96f"

privacyExecAddr="1FeyE6VDZ4FYgpK1n2okWMDAtPkwBuooQd"

priTotalAmount1="300.0000"
priTotalAmount2="300.0000"
priTxHashs1=("")
priTxHashs2=("")
priTxFee1=0
priTxFee2=0

function saveSeedToWallet() {
    saveSeed "${CLI}" ${password} "${walletSeed}"
    unlockWallet "${CLI}" ${password}
    saveSeed "${CLI4}" ${password} "${walletSeed}"
    unlockWallet "${CLI4}" ${password}
}

function importPrivateKey() {
    importKey "${CLI}" CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944 returnAddr
    importKey "${CLI}" 4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01 minerAddr
    importKey "${CLI4}" 2AFF1981291355322C7A6308D46A9C9BA311AA21D94F36B43FC6A6021A1334CF returnAddr
    importKey "${CLI4}" 2116459C0EC8ED01AA0EEAE35CAC5C96F94473F7816F114873291217303F6989 minerAddr

    setAutomine "${CLI}" 0
    setAutomine "${CLI4}" 0

    local SLEEP=20
    echo "=========== 私钥导入完毕,等待 ${SLEEP}s ============="
    sleep ${SLEEP}
}

function initPrivacyAccount() {
    saveSeedToWallet
    importPrivateKey

    # 向隐私账户打入交易需要的金额
    name="${CLI}"
    fromAdd=$CLIfromAddr1
    execAdd=$privacyExecAddr
    note="test"
    amount=$priTotalAmount1
    SendToPrivacyExec "${name}" $fromAdd $execAdd $note $amount
    block_wait_timeout "${name}" 2 50

    name="${CLI4}"
    fromAdd=$CLI4fromAddr1
    execAdd=$privacyExecAddr
    note="test"
    amount=$priTotalAmount2
    SendToPrivacyExec "${name}" $fromAdd $execAdd $note $amount

    block_wait_timeout "${name}" 2 50

    name="${CLI}"
    fromAdd=$CLIfromAddr1
    showPrivacyExec "${name}" $fromAdd

    sleep 1

    name="${CLI4}"
    fromAdd=$CLI4fromAddr1
    showPrivacyExec "${name}" $fromAdd
}

function checkMineHeight() {
    result=$($CLI4 wallet auto_mine -f 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "stop wallet1 mine fail"
        return 1
    fi
    sleep 1
    result=$($CLI wallet auto_mine -f 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "stop wallet2 mine fail"
        return 1
    fi
    echo "====== stop all wallet mine success ======"

    echo "====== syn blockchain ======"
    syn_block_timeout "${CLI}" 5 50 "${containers[@]}"

    height=0
    height1=$($CLI4 block last_header | jq ".height")
    sleep 1
    height2=$($CLI block last_header | jq ".height")
    sleep 1
    if [ "${height2}" -ge "${height1}" ]; then
        height=$height2
        printf '当前最大高度 %s \n' "${height}"
    else
        height=$height1
        printf '当前最大高度 %s \n' "${height}"
    fi

    if [ "${height}" -eq 0 ]; then
        echo "获取当前最大高度失败"
        return 1
    fi
    loopCount=20
    for ((k = 0; k < ${#forkContainers[*]}; k++)); do
        for ((j = 0; j < loopCount; j++)); do
            height1=$(${forkContainers[$k]} block last_header | jq ".height")
            if [ "${height1}" -gt "${height}" ]; then #如果大于说明区块还没有完全产生完，替换期望高度
                height=$height1
                printf '查询 %s 目前区块最高高度为第 %s \n' "${containers[$k]}" "${height}"
            elif [ "${height1}" -eq "${height}" ]; then #找到目标高度
                break
            else
                printf '查询 %s 第 %d 次，当前高度 %d, 需要高度%d, 同步中，sleep 60s 后查询\n' "${containers[$k]}" $j "${height1}" "${height}"
                sleep 60
            fi
            #检查是否超过了最大检测次数
            if [ $j -ge $((loopCount - 1)) ]; then
                echo "====== syn blockchain fail======"
                return 1
            fi
        done
    done

    return 0
}

function checkPeersInTheSameHeight() {
    checkMineHeight
    status=$?
    if [ $status -eq 0 ]; then
        echo "====== All peers is the same height ======"
    else
        echo "====== All peers is the different height, syn blockchain fail======"
        exit 1
    fi
}

# $1 name
# $2 fromAddr1  发起者的地址,以及私对公的接收者
# $3 pk1        公对私的接收者
# $4 pk2        私对私的接收者
# $5 group      1表示Docker分组1,2表示Docker分组2
function makeTransactionIn3Step() {
    name=$1
    fromAddr1=$2
    pk1=$3
    pk2=$4
    group=$5
    echo "当前操作的链节点为：${name}, 分组类型为${group}"

    priTxindex=0

    height=$(${name} block last_header | jq ".height")
    amount=10
    expire=$((height + 4))
    printf '公对私交易 高度为:%s 转账金额为:%s \n' "${height}" "${amount}"
    createPrivacyPub2PrivTx "${name}" $pk1 $amount $expire
    signRawTx "${name}" $fromAddr1 $returnStr1
    sendRawTx "${name}" $returnStr1
    echo $returnStr1
    block_wait_timeout "${name}" 1 15

    height=$(${name} block last_header | jq ".height")
    printf '发送私对私交易当前高度 %s \n' "${height}"
    amount=4
    expire=$((height + 4))
    printf '私对私交易 高度为:%s 转账金额为:%s \n' "${height}" "${amount}"
    createPrivacyPriv2PrivTx "${name}" $pk2 $amount $fromAddr1 $expire
    signRawTx "${name}" $fromAddr1 $returnStr1
    sendRawTx "${name}" $returnStr1
    echo $returnStr1
    if [ $group -eq 1 ]; then
        priTxHashs1[$priTxindex]=$returnStr1
        priTxindex=$((priTxindex + 1))
    else
        priTxHashs2[$priTxindex]=$returnStr1
        priTxindex=$((priTxindex + 1))
    fi
    block_wait_timeout "${name}" 1 15

    height=$(${name} block last_header | jq ".height")
    printf '发送私对公交易当前高度 %s \n' "${height}"
    amount=4
    from=$fromAddr1
    to=$fromAddr1
    # 4个区块高度以后过期
    expire=$((height + 4))
    createPrivacyPriv2PubTx "${name}" $from $to $amount $expire
    signRawTx "${name}" $from $returnStr1
    sendRawTx "${name}" $returnStr1
    echo $returnStr1
    if [ $group -eq 1 ]; then
        priTxHashs1[$priTxindex]=$returnStr1
        priTxindex=$((priTxindex + 1))
    else
        priTxHashs2[$priTxindex]=$returnStr1
        priTxindex=$((priTxindex + 1))
    fi
    block_wait_timeout "${name}" 1 15
}

function buildTransactionInGroup1() {
    makeTransactionIn3Step "${CLI}" $CLIfromAddr1 $CLIprivKey1 $CLIprivKey2 1
}

function buildTransactionInGroup2() {
    makeTransactionIn3Step "${CLI4}" $CLI4fromAddr1 $CLI4privKey11 $CLI4privKey12 2
}

function checkBlockHashfun() {
    echo "====== syn blockchain ======"
    syn_block_timeout "${CLI}" 10 50 "${containers[@]}"

    height=0
    hash=""
    height1=$($CLI block last_header | jq ".height")
    sleep 1
    height2=$($CLI4 block last_header | jq ".height")
    if [ "${height2}" -ge "${height1}" ]; then
        height=$height2
        printf "主链为 $CLI 当前最大高度 %d \\n" "${height}"
        sleep 1
        hash=$($CLI block hash -t "${height}" | jq ".hash")
    else
        height=$height1
        printf "主链为 $CLI4 当前最大高度 %d \\n" "${height}"
        sleep 1
        hash=$($CLI4 block hash -t "${height}" | jq ".hash")
    fi

    for ((j = 0; j < $1; j++)); do
        for ((k = 0; k < ${#forkContainers[*]}; k++)); do
            sleep 1
            height0[$k]=$(${forkContainers[$k]} block last_header | jq ".height")
            if [ "${height0[$k]}" -ge "${height}" ]; then
                sleep 1
                hash0[$k]=$(${forkContainers[$k]} block hash -t "${height}" | jq ".hash")
            else
                hash0[$k]="${forkContainers[$k]}"
            fi
        done

        if [ "${hash0[0]}" = "${hash}" ] && [ "${hash0[1]}" = "${hash}" ] && [ "${hash0[2]}" = "${hash}" ] && [ "${hash0[3]}" = "${hash}" ] && [ "${hash0[4]}" = "${hash}" ] && [ "${hash0[5]}" = "${hash}" ]; then
            echo "syn blockchain success break"
            break
        else
            if [ "${hash0[1]}" = "${hash0[0]}" ] && [ "${hash0[2]}" = "${hash0[0]}" ] && [ "${hash0[3]}" = "${hash0[0]}" ] && [ "${hash0[4]}" = "${hash0[0]}" ] && [ "${hash0[5]}" = "${hash0[0]}" ]; then
                echo "syn blockchain success break"
                break
            fi
        fi
        peersCount=0
        peersCount=$(${forkContainers[0]} net peer_info | jq '.[] | length')
        printf '第 %d 次，未查询到网络同步，当前节点数 %d 个，100s后查询\n' $j "${peersCount}"
        sleep 100
        #检查是否超过了最大检测次数
        var=$(($1 - 1))
        if [ $j -ge "${var}" ]; then
            echo "====== syn blockchain fail======"
            exit 1
        fi
    done
    echo "====== syn blockchain success======"
}

function checkPrivacyRunResult() {
    name1=${CLI}
    name2=${CLI4}
    echo "====================检查第一组docker运行结果================="
    for ((i = 0; i < ${#priTxHashs1[*]}; i++)); do
        txHash=${priTxHashs1[$i]}
        txQuery "${name1}" $txHash
        result=$?
        if [ $result -eq 0 ]; then
            priTxFee1=$((priTxFee1 + 1))
        fi
        sleep 1
    done
    showPrivacyExec "${name1}" $CLIfromAddr1
    value1=$returnStr1
    sleep 1
    showPrivacyBalance "${name1}" $CLIfromAddr1
    value2=$returnStr1
    sleep 1
    showPrivacyBalance "${name1}" $CLIfromAddr2
    value3=$returnStr1
    sleep 1
    printf '中间交易费为%d \n' "${priTxFee1}"
    actTotal=$(echo "$value1 + $value2 + $value3 + $priTxFee1" | bc)
    echo "${name1} 实际金额：$actTotal"

    echo "====================检查第二组docker运行结果================="
    for ((i = 0; i < ${#priTxHashs2[*]}; i++)); do
        txHash=${priTxHashs2[$i]}
        txQuery "${name2}" $txHash
        result=$?
        if [ $result -eq 0 ]; then
            priTxFee2=$((priTxFee2 + 1))
        fi
        sleep 1
    done
    showPrivacyExec "${name2}" $CLI4fromAddr1
    value1=$returnStr1
    sleep 1
    showPrivacyBalance "${name2}" $CLI4fromAddr1
    value2=$returnStr1
    sleep 1
    showPrivacyBalance "${name2}" $CLI4fromAddr2
    value3=$returnStr1
    sleep 1
    printf '中间交易费为%d \n' "${priTxFee2}"
    actTotal=$(echo "$value1 + $value2 + $value3 + $priTxFee2" | bc)
    echo "${name2} 实际金额：$actTotal"
}

function switchToDockerGroup1() {
    echo "====== 第一步：第一组docker挖矿======"
    docker start "${NODE1}" "${NODE2}" "${NODE3}"
    echo "======停止第二组docker ======"
    docker stop "${NODE4}" "${NODE5}" "${NODE6}"

    echo "======开启第一组docker节点挖矿======"
    sleep 3
    waitAllPeerReady "${CLI}" 3
    unlockWallet "${CLI}" ${password}
    setAutomine "${CLI}" 1
}

function switchToDockerGroup2() {
    echo "====== 第一步：第二组docker挖矿======"
    docker start "${NODE4}" "${NODE5}" "${NODE6}"
    echo "======停止第一组docker ======"
    docker stop "${NODE1}" "${NODE2}" "${NODE3}"

    echo "======开启第二组docker节点挖矿======"
    sleep 3
    waitAllPeerReady "${CLI4}" 3
    unlockWallet "${CLI4}" ${password}
    setAutomine "${CLI4}" 1
}

function startAllDockers() {
    docker start "${NODE1}" "${NODE2}" "${NODE3}" "${NODE4}" "${NODE5}" "${NODE6}"
    waitAllPeerReady "${CLI}" 6
}

function initTestEnv() {
    startDockers
    # display cli information
    ${CLI} block last_header
    ${CLI} net info
    ${CLI} net peer_info
    echo "=========== 等待所有节点在线 ==========="
    waitAllPeerReady "${CLI}" 6

    echo "=========== 初始化钱包账号信息以及资金信息 ==========="
    initPrivacyAccount
    displayWalletStatus "${CLI}"
    listAccount "${CLI}"
    listMempoolTxs "${CLI}"
}

function runPrivacyTestType1() {
    echo "===== 初始化测试环境 ====="
    initTestEnv

    echo "===== 开始执行分叉超时测试 ====="
    switchToDockerGroup1
    buildTransactionInGroup1
    switchToDockerGroup2
    buildTransactionInGroup2
    echo "===== 执行分叉超时测试结束 ====="

    startAllDockers
    setAutomine "${CLI}" 0
    setAutomine "${CLI4}" 0
    checkPeersInTheSameHeight

    checkPrivacyRunResult
}

function runPrivacyTest() {
    echo "=========== 运行runPrivacyTest测试 ========== "
    runPrivacyTestType1
}

runPrivacyTest
