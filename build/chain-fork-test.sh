#!/usr/bin/env bash

chain30Fee=0
chain33Fee=0


function checkMineHeight()
{
    result=$(sudo docker exec -it build_chain30_1 ./chain33-cli wallet auto_mine -f 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "stop wallet1 mine fail"
        return 1
    fi
    sleep 3
    result=$(sudo docker exec -it build_chain33_1 ./chain33-cli wallet auto_mine -f 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "stop wallet2 mine fail"
        return 1
    fi

    echo "====== stop all wallet mine success ======"

    echo "====== sleep 100s syn blockchain ======"
    sleep 100

    height=0
    height1=$(sudo docker exec -it build_chain30_1 ./chain33-cli block last_header | jq ".height")
    sleep 3
    height2=$(sudo docker exec -it build_chain33_1 ./chain33-cli block last_header | jq ".height")
    if [ $height2 -ge $height1 ]
    then
        height=$height2
        printf "当前最大高度 %d \n" $height
    else
        height=$height1
        printf "当前最大高度 %d \n" $height
    fi

    if [ $height -eq 0 ]
    then
       echo "获取当前最大高度失败"
       return 1
    fi
    array=("build_chain28_1" "build_chain29_1" "build_chain31_1" "build_chain32_1")
    k=0
    for i in ${array[*]}; do
        for((j=0;j<10;j++)); do
            height1=$(sudo docker exec -it ${array[$k]} ./chain33-cli block last_header | jq ".height")
	    if [ $height1 -gt $height ]   #如果大于说明区块还没有完全产生完，替换期望高度
	    then
                height=$height1
                printf "查询 %s 目前区块最高高度为第 %d \n" ${array[$k]} $height
            elif [ $height1 -eq $height ] #找到目标高度
            then 
                break
            else
                printf "查询 %s 第 %d 次，当前高度 %d, 需要高度%d, 同步中，sleep 200s 后查询\n" ${array[$k]} $j $height1 $height
                sleep 200
            fi
            #检查是否超过了最大检测次数
            if [ $j -ge 9 ]
            then
               echo "====== syn blockchain fail======"
               return 1
            fi
        done
        k=`expr $k + 1`
    done

    return 0
}

function peersCount()
{
   name=$1
   time=$2
   needCount=$3
   
    for((i=0;i<$time;i++)); do
       peersCount=$(sudo docker exec -it $name ./chain33-cli net peer_info | jq '.[] | length')
       printf "查询节点 %s ,所需节点数 %d ,当前节点数 %s\n" $name $needCount ${peersCount}
       if [ "${peersCount}" = "$needCount" ];then
           echo "============= 符合节点数要求 ============="
           return 0
       else 
          echo "============= 休眠 60s 继续查询 ============="
          sleep 60
       fi
    done

    return 1
}

function optDockerfun()
{
    echo "====== sleep 100s 区块生成中 ======"

    name="build_chain30_1"
    fromAdd="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
    execAdd="1FeyE6VDZ4FYgpK1n2okWMDAtPkwBuooQd"
    note="test"
    amount=300
    SendToPrivacyExec $name $fromAdd $execAdd $note $amount

    sleep 3

    name="build_chain33_1"
    fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
    execAdd="1FeyE6VDZ4FYgpK1n2okWMDAtPkwBuooQd"
    note="test"
    amount=300
    SendToPrivacyExec $name $fromAdd $execAdd $note $amount

    sleep 200

    sleep 3

    name="build_chain30_1"
    fromAdd="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
    showPrivacyExec $name $fromAdd

    sleep 3

    name="build_chain33_1"
    fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
    showPrivacyExec $name $fromAdd


    for((i=0;i<20;i++)); do
        name="build_chain33_1"
        time=2
        needCount=6
        peersCount $name $time $needCount
        peerStatus=$?
        if [ $peerStatus -eq 1 ];then
            name="build_chain30_1"
            peersCount $name $time $needCount
            peerStatus=$?
            if [ $peerStatus -eq 0 ];then
               break
            fi
        else
            break
        fi
        #检查是否超过了最大检测次数
        if [ $i -ge 19 ]
        then
            echo "====== peers not enough ======"
            exit 1
        fi
    done

    checkMineHeight
    status=$?
    echo $status
    if [ $status -eq 0 ];then
       echo "====== All peers is the same height ======"
    else
       echo "====== All peers is the different height, syn blockchain fail======"
       exit 1
    fi

    echo "==================================="
    echo "====== 第一步：第一组docker挖矿======"
    echo "==================================="

    echo "======停止第二组docker ======"
    sudo docker stop build_chain31_1
    sleep 3
    sudo docker stop build_chain32_1
    sleep 3
    sudo docker stop build_chain33_1


    echo "======开启第一组docker节点挖矿======"
    sleep 3
    result=$(sudo docker exec -it build_chain30_1 ./chain33-cli wallet auto_mine -f 1 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "start wallet2 mine fail"
        exit 1
    fi


    name="build_chain30_1"
    time=60
    needCount=3

    peersCount $name $time $needCount
    peerStatus=$?
    echo $peerStatus
    if [ $peerStatus -eq 1 ];then
       echo "====== peers not enough ======"
       exit 1
    fi

#    echo "======sleep 150s 发送交易======"
#    for((i=0;i<50;i++)); do
#       name="build_chain30_1"
#       to="1KcCVZLSQYRUwE5EXTsAoQs9LuJW6xwfQa"
#       From="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
#       note=1
#       amount=0.2
#       sendTx $name $to $From $note $amount
#       sleep 3
#       height=$(sudo docker exec -it build_chain33_1 ./chain33-cli block last_header | jq ".height")
#       printf "%s 发送第 %d 笔交易 当前高度 %d \n" $name $i $height
#    done

    echo "====== 发送公对私交易 ======"
    for((i=0;i<3;i++)); do
       name="build_chain30_1"
       fromAdd="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
       priAdd="069fdcd7a2d7cf30dfc87df6f277ae451a78cae6720a6bb05514a4a43e0622d55c854169cc63b6353234c3e65db75e7b205878b1bd94e9f698c7043b27fa162b"
       note="pub2priv_test"
       amount=5
       pub2priv $name $fromAdd $priAdd $note $amount

       sleep 3
       height=$(sudo docker exec -it build_chain30_1 ./chain33-cli block last_header | jq ".height")
       printf "%s 发送公对私第 %d 笔交易 当前高度 %d \n" $name $i $height
    done

    echo "======sleep 150s======"
    sleep 150

    echo "====== 发送私对私交易 ======"
    for((i=0;i<3;i++)); do
       name="build_chain30_1"
       fromAdd="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
       priAdd="d5672eeafbcdf53c8fc27969a5d9797083bb64fb4848bd391cd9b3919c4a1d3cb8534f12e09de3cc541eaaf45acccacaf808a6804fd10a976804397e9ecaf96f"
       note="priv2priv_test"
       amount=2
       mixcount=0
       priv2priv $name $fromAdd $priAdd $note $amount $mixcount

       sleep 3
       height=$(sudo docker exec -it build_chain30_1 ./chain33-cli block last_header | jq ".height")
       printf "%s 发送私对私第 %d 笔交易 当前高度 %d \n" $name $i $height
       chain30Fee=`expr $chain30Fee + 1`
    done

    echo "======停止第一组docker节点挖矿======"
    result=$(sudo docker exec -it build_chain30_1 ./chain33-cli wallet auto_mine -f 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "stop wallet2 mine fail"
        return 1
    fi

    echo "====== sleep 200s 第一组内部同步 ======"
    sleep 200

    echo "=============查询当前隐私余额============="
    name="build_chain30_1"
    fromAdd="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
    showPrivacyBalance $name $fromAdd

    name="build_chain30_1"
    fromAdd="1KcCVZLSQYRUwE5EXTsAoQs9LuJW6xwfQa"
    showPrivacyBalance $name $fromAdd


    echo "======================================="
    echo "======== 第二步：第二组docker挖矿 ======="
    echo "======================================="


    echo "======停止第一组docker======"
    sleep 3
    sudo docker stop build_chain28_1
    sleep 3
    sudo docker stop build_chain29_1
    sleep 3
    sudo docker stop build_chain30_1

    echo "======sleep 60s======"
    sleep 60

    echo "======启动第二组docker======"
    sleep 3
    sudo docker start build_chain31_1
    sleep 3
    sudo docker start build_chain32_1
    sleep 3
    sudo docker start build_chain33_1

    echo "======sleep 20s======"
    sleep 20
    result=$(sudo docker exec -it build_chain33_1 ./chain33-cli wallet unlock -p 1314 -t 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "wallet1 unlock fail"
        exit 1
    fi

    name="build_chain33_1"
    time=60
    needCount=3

    peersCount $name $time $needCount
    peerStatus=$?
    echo $peerStatus
    if [ $peerStatus -eq 1 ];then
       echo "====== peers not enough ======"
       exit 1
    fi

    echo "======开启第二组docker节点挖矿======"
    sleep 3
    result=$(sudo docker exec -it build_chain33_1 ./chain33-cli wallet auto_mine -f 1 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "start wallet1 mine fail"
        exit 1
    fi


#    echo "======sleep 150s 发送交易======"
#    for((i=0;i<50;i++)); do
#       name="build_chain33_1"
#       to="1KcCVZLSQYRUwE5EXTsAoQs9LuJW6xwfQa"
#       From="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
#       note=1
#       amount=0.2
#       sendTx $name $to $From $note $amount
#       sleep 3
#       height=$(sudo docker exec -it build_chain33_1 ./chain33-cli block last_header | jq ".height")
#       printf "%s 发送第 %d 笔交易 当前高度 %d \n" $name $i $height
#    done

    echo "====== 发送公对私交易 ======"
    for((i=0;i<3;i++)); do
       name="build_chain33_1"
       fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
       priAdd="0a9d212b2505aefaa8da370319088bbccfac097b007f52ed71d8133456c8185823c8eac43c5e937953d7b6c8e68b0db1f4f03df4946a29f524875118960a35fb"
       note="pub2priv_test"
       amount=5
       pub2priv $name $fromAdd $priAdd $note $amount

       sleep 3
       height=$(sudo docker exec -it build_chain33_1 ./chain33-cli block last_header | jq ".height")
       printf "%s 发送公对私第 %d 笔交易 当前高度 %d \n" $name $i $height
    done

    echo "======sleep 60s======"
    sleep 60

    echo "====== 发送私对私交易 ======"
    for((i=0;i<3;i++)); do
       name="build_chain33_1"
       fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
       priAdd="fcbb75f2b96b6d41f301f2d1abc853d697818427819f412f8e4b4e12cacc0814d2c3914b27bea9151b8968ed1732bd241c8788a332b295b731aee8d39a060388"
       note="priv2priv_test"
       amount=3
       mixcount=0
       priv2priv $name $fromAdd $priAdd $note $amount $mixcount

       sleep 3
       height=$(sudo docker exec -it build_chain33_1 ./chain33-cli block last_header | jq ".height")
       printf "%s 发送私对私第 %d 笔交易 当前高度 %d \n" $name $i $height
       chain33Fee=`expr $chain33Fee + 1`
    done

    echo "====== sleep 200s 第二组内部同步 ======"
    sleep 200

    echo "=============查询当前隐私余额============="
    name="build_chain33_1"
    fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
    showPrivacyBalance $name $fromAdd

    name="build_chain33_1"
    fromAdd="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
    showPrivacyBalance $name $fromAdd

    echo "======================================="
    echo "====== 第三步：两组docker共同挖矿 ======="
    echo "======================================="

    echo "======启动第一组docker======"
    sleep 3
    sudo docker start build_chain28_1
    sleep 3
    sudo docker start build_chain29_1
    sleep 3
    sudo docker start build_chain30_1

    echo "======sleep 20s======"
    sleep 20
    result=$(sudo docker exec -it build_chain30_1 ./chain33-cli wallet unlock -p 1314 -t 0 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "wallet2 unlock fail"
        exit 1
    fi
    echo "======开启第一组docker节点挖矿======"
    sleep 3
    result=$(sudo docker exec -it build_chain30_1 ./chain33-cli wallet auto_mine -f 1 | jq ".isok")
    if [ "${result}" = "false" ]; then
        echo "start wallet2 mine fail"
        exit 1
    fi
}

function checkBlockHashfun()
{
    echo "====== sleep 100s syn blockchain ======"
    sleep 100

    height=0
    hash=""
    sleep 3
    height1=$(sudo docker exec -it build_chain30_1 ./chain33-cli block last_header | jq ".height")
    sleep 3
    height2=$(sudo docker exec -it build_chain33_1 ./chain33-cli block last_header | jq ".height")
    if [ $height2 -ge $height1 ]
    then
        height=$height2
        printf "主链为 build_chain33_1 当前最大高度 %d \n" $height
        sleep 3
        hash=$(sudo docker exec -it build_chain33_1 ./chain33-cli block hash -t $height | jq ".hash")
    else
        height=$height1
        printf "主链为 build_chain30_1 当前最大高度 %d \n" $height
        sleep 3
        hash=$(sudo docker exec -it build_chain30_1 ./chain33-cli block hash -t $height | jq ".hash")
    fi

    echo $hash

    array=("build_chain28_1" "build_chain29_1" "build_chain30_1" "build_chain31_1" "build_chain32_1" "build_chain33_1")

    
    for((j=0;j<$1;j++)); do
        k=0
        for i in ${array[*]}; do
            sleep 3
            height0[$k]=$(sudo docker exec -it ${array[$k]} ./chain33-cli block last_header | jq ".height")
	    if [ ${height0[$k]} -ge $height ]
	    then
                sleep 3
                hash0[$k]=$(sudo docker exec -it ${array[$k]} ./chain33-cli block hash -t $height | jq ".hash")
            else
                hash0[$k]=${array[$k]}
            fi
            k=`expr $k + 1`
        done
            
        if [ ${hash0[0]} = $hash ]&&[ ${hash0[1]} = $hash ]&&[ ${hash0[2]} = $hash ]&&[ ${hash0[3]} = $hash ]&&[ ${hash0[4]} = $hash ]&&[ ${hash0[5]} = $hash ]
        then
            echo "syn blockchain success break"
            break
        else
            if [ ${hash0[1]} = ${hash0[0]} ]&&[ ${hash0[2]} = ${hash0[0]} ]&&[ ${hash0[3]} = ${hash0[0]} ]&&[ ${hash0[4]} = ${hash0[0]} ]&&[ ${hash0[5]} = ${hash0[0]} ]
            then
                echo "syn blockchain success break"
                break
            fi
        fi
        printf "第 %d 次，未查询到网络同步，sleep 100s 后查询\n" $j
        sleep 100
        #检查是否超过了最大检测次数
        var=`expr $1 - 1`
        if [ $j -ge $var ]
        then
            echo "====== syn blockchain fail======"
            exit 1
        fi
    done
    echo "====== syn blockchain success======"

    echo "====== sleep 150s====="
    sleep 150


    name="build_chain30_1"
    fromAdd="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
    showPrivacyExec $name $fromAdd

    name="build_chain30_1"
    fromAdd="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
    showPrivacyBalance $name $fromAdd

    sleep 3

    name="build_chain30_1"
    fromAdd="1KcCVZLSQYRUwE5EXTsAoQs9LuJW6xwfQa"
    showPrivacyBalance $name $fromAdd

    printf "中间交易费为%d \n" $chain30Fee

    sleep 3

    name="build_chain33_1"
    fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
    showPrivacyExec $name $fromAdd

    sleep 3

    name="build_chain33_1"
    fromAdd="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
    showPrivacyBalance $name $fromAdd

    sleep 3

    name="build_chain33_1"
    fromAdd="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
    showPrivacyBalance $name $fromAdd

    printf "中间交易费为%d \n" $chain33Fee
}



function sendTx()
{
    #hash=$(sudo docker exec -it $1 ./chain33-cli bty transfer -t $2 -a $5 -n $4 | tr '\r' ' ')
    #echo $hash

    #sign=$(sudo docker exec -it $1 ./chain33-cli wallet sign -a $3 -d $hash | tr '\r' ' ')
    #echo $sign

    #sudo docker exec -it $1 ./chain33-cli wallet send -d $sign

    sudo docker exec -it $1 ./chain33-cli send bty transfer -a $5 -n $4 -t $2 -k $From
}

# $1 name
# $2 fromAdd
# $3 execAdd
# $4 note
# $5 amount
function SendToPrivacyExec()
{
    name=$1
    fromAdd=$2
    execAdd=$3
    note=$4
    amount=$5

    sudo docker exec -it $name ./chain33-cli send bty transfer -k $fromAdd -t $execAdd -n $note -a $amount
}

# $1 name
# $2 fromAdd
# $3 priAdd
# $4 note
# $5 amount
function pub2priv()
{
    name=$1
    fromAdd=$2
    priAdd=$3
    note=$4
    amount=$5
    sudo docker exec -it $name ./chain33-cli privacy pub2priv -f $fromAdd -p $priAdd -a $amount -n $note
}

# $1 name
# $2 fromAdd
# $3 priAdd
# $4 note
# $5 amount
# $6 mixcount
function priv2priv()
{
    name=$1
    fromAdd=$2
    priAdd=$3
    note=$4
    amount=$5
    mixcount=$6
    #sudo docker exec -it $name ./chain33-cli privacy priv2priv -f $fromAdd -p $priAdd -a $amount -n $note -m $mixcount
    sudo docker exec -it $name ./chain33-cli privacy priv2priv -f $fromAdd -p $priAdd -a $amount -n $note
}

# $1 name
# $2 fromAdd
# $3 toAdd
# $4 note
# $5 amount
# $6 mixcount
function priv2pub()
{
    name=$1
    fromAdd=$2
    toAdd=$3
    note=$4
    amount=$5
    mixcount=$6
    sudo docker exec -it $name ./chain33-cli privacy priv2pub -f $fromAdd -t $toAdd -a $amount -n $note -m $mixcount
}


# $1 name
# $2 fromAdd
function showPrivacyExec()
{
    name=$1
    fromAdd=$2
    printf "==========showPrivacyExec name=%s addr=%s==========\n" $name $fromAdd
    sudo docker exec -it $name ./chain33-cli account balance -e privacy -a $fromAdd
}


# $1 name
# $2 fromAdd
function showPrivacyBalance()
{
    name=$1
    fromAdd=$2
    printf "==========showPrivacyBalance name=%s addr=%s==========\n" $name $fromAdd
    #result=$(sudo docker exec -it $name ./chain33-cli privacy showpb -a $fromAdd | jq ".PrivacyBalance")
    #printf "Balance %s \n" $result
    #sudo docker exec -it $name ./chain33-cli privacy showpb -a $fromAdd
    sudo docker exec -it $name ./chain33-cli privacy showpai -a $fromAdd -d 0
}

optDockerfun
loopCount=600 #循环次数，每次循环休眠时间5分钟
checkBlockHashfun $loopCount


