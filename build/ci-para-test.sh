#!/usr/bin/env bash

PARA_CLI="docker exec ${NODE3} /root/chain33-para-cli"

PARA_CLI2="docker exec ${NODE2} /root/chain33-para-cli"
PARA_CLI1="docker exec ${NODE1} /root/chain33-para-cli"
PARA_CLI0="docker exec ${NODE0} /root/chain33-para-cli"

function para_set_wallet() {
    echo "=========== # para set wallet ============="
    import_key "${PARA_CLI}" "0x6da92a632ab7deb67d38c0f6560bcfed28167998f6496db64c258d5e8393a81b"
    import_key "${PARA_CLI2}" "0x19c069234f9d3e61135fefbeb7791b149cdf6af536f26bebb310d4cd22c3fee4"
    import_key "${PARA_CLI1}" "0x7a80a1f75d7360c6123c32a78ecf978c1ac55636f87892df38d8b85a9aeff115"
    import_key "${PARA_CLI0}" "0xcacb1f5d51700aea07fca2246ab43b0917d70405c65edea9b5063d72eb5c6b71"
}

function para_transfer() {
    echo "=========== # para chain transfer ============="
    para_transfer2account "1Q8hGLfoGe63efeWa8fJ4Pnukhkngt6poK"
    para_transfer2account "1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4"
    para_transfer2account "1JRNjdEqp4LJ5fqycUBm9ayCKSeeskgMKR"
    para_transfer2account "1NLHPEcbTWWxxU3dGUZBhayjrCHD3psX7k"
    para_transfer2account "1MCftFynyvG2F4ED5mdHYgziDxx6vDrScs"
    block_wait "${CLI}" 1

    para_config "${CLI}" "paracross-nodes-user.p.${PARANAME}." "1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4"
    para_config "${CLI}" "paracross-nodes-user.p.${PARANAME}." "1JRNjdEqp4LJ5fqycUBm9ayCKSeeskgMKR"
    para_config "${CLI}" "paracross-nodes-user.p.${PARANAME}." "1NLHPEcbTWWxxU3dGUZBhayjrCHD3psX7k"
    para_config "${CLI}" "paracross-nodes-user.p.${PARANAME}." "1MCftFynyvG2F4ED5mdHYgziDxx6vDrScs"

    para_config "${PARA_CLI}" "token-blacklist" "BTY"

}

function para_transfer2account() {
    echo "${1}"
    hash1=$(${CLI} send coins transfer -a 10 -n test -t "${1}" -k 4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01)
    echo "${hash1}"
}

function para_config() {
    echo "=========== # para chain send config ============="
    echo "${3}"
    tx=$(${1} config config_tx -o add -k "${2}" -v "${3}")
    echo "${tx}"
    sign=$(${CLI} wallet sign -k 0xc34b5d9d44ac7b754806f761d3d4d2c4fe5214f6b074c19f069c4f5c2a29c8cc -d "${tx}")
    echo "${sign}"
    send=$(${CLI} wallet send -d "${sign}")
    echo "${send}"
}

function token_create() {
    echo "=========== # para token test ============="
    hash=$(${1} send token precreate -f 0.001 -i test -n guodunjifen -a 1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4 -p 0 -s GD -t 10000 -k 1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4)
    echo "${hash}"
    block_wait "${1}" 2

    owner=$(${1} tx query -s "${hash}" | jq -r ".receipt.logs[0].log.owner")
    if [ "${owner}" != "1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4" ]; then
        echo "wrong pre create owner"
        exit 1
    fi

    hash=$(${1} send token finish -f 0.001 -a 1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4 -s GD -k 0xc34b5d9d44ac7b754806f761d3d4d2c4fe5214f6b074c19f069c4f5c2a29c8cc)
    echo "${hash}"
    block_wait "${1}" 2

    owner=$(${1} tx query -s "${hash}" | jq -r ".receipt.logs[1].log.owner")
    if [ "${owner}" != "1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4" ]; then
        echo "wrong finish create owner"
        exit 1
    fi

}
function para() {
    echo "=========== # para chain test ============="
    token_create "${PARA_CLI}"
}
