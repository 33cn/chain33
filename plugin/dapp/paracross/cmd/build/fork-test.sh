#!/usr/bin/env bash

# shellcheck source=/dev/null
source testcase.sh

#================fork-test============================
# shellcheck disable=SC2154
function checkParaBlockHashfun() {
    echo "====== syn para blockchain ======"

    height=0
    hash=""
    height1=$($PARA_CLI block last_header | jq ".height")
    sleep 1
    height2=$($PARA_CLI4 block last_header | jq ".height")

    if [ "${height2}" -ge "${height1}" ]; then
        height=$height2
        printf "主链为 $PARA_CLI 当前最大高度 %d \\n" "${height}"
        sleep 1
        hash=$($CLI block hash -t "${height}" | jq ".hash")
    else
        height=$height1
        printf "主链为 $PARA_CLI4 当前最大高度 %d \\n" "${height}"
        sleep 1
        hash=$($CLI4 block hash -t "${height}" | jq ".hash")
    fi

    for ((j = 0; j < $1; j++)); do
        for ((k = 0; k < ${#forkParaContainers[*]}; k++)); do
            sleep 1
            height0[$k]=$(${forkParaContainers[$k]} block last_header | jq ".height")
            if [ "${height0[$k]}" -ge "${height}" ]; then
                sleep 1
                hash0[$k]=$(${forkParaContainers[$k]} block hash -t "${height}" | jq ".hash")
            else
                hash0[$k]="${forkParaContainers[$k]}"
            fi
        done

        if [ "${hash0[0]}" = "${hash}" ] && [ "${hash0[1]}" = "${hash}" ] && [ "${hash0[2]}" = "${hash}" ] && [ "${hash0[3]}" = "${hash}" ]; then
            echo "syn para blockchain success break"
            break
        else
            if [ "${hash0[1]}" = "${hash0[0]}" ] && [ "${hash0[2]}" = "${hash0[0]}" ] && [ "${hash0[3]}" = "${hash0[0]}" ]; then
                echo "syn para blockchain success break"
                break
            fi
        fi

        printf '第 %d 次，10s后查询\n' $j
        sleep 10
        #检查是否超过了最大检测次数
        var=$(($1 - 1))
        if [ $j -ge "${var}" ]; then
            echo "====== syn para blockchain fail======"
            exit 1
        fi
    done
    echo "====== syn para blockchain success======"
}
