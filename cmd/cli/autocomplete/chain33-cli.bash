#!/bin/bash

set +x

function _debug() {
    set -x
    echo "\n debug"
    for var in $@; do
        echo "\t\t$var"
    done
    set +x
}

function _chain33() {

    set +x
    # 这是套路
    local cur prev words cword
    _init_completion || return
    #_debug $cur $prev ${words[*]} $cword

    local commands
    commands=($(${words[0]} 2>&1 | sed '/Available Commands/,/^$/p' -n | grep "  [a-z][a-z]*" -o | xargs))
    #_debug "commands " ${commands[@]}

    local command i
    for ((i = 0; i < ${#words[@]} - 1; i++)); do
        if [[ ${commands[@]} =~ ${words[i]} ]]; then
            command=${words[i]}
            break
        fi
    done

    #_debug $command $i

    if [ "$command" == "" ]; then
        COMPREPLY=($(compgen -W '${commands[@]}' -- "$cur"))
        return 0
    fi

    local sub_commands sub_command sub_idx
    for ((i = 0; i < ${#commands[@]} - 1; i++)); do
        if [[ ${commands[i]} == $command ]]; then
            sub_commands=($(${words[0]} $command 2>&1 | sed '/Available Commands/,/^$/p' -n | grep "  [a-z][2a-z]*" -o | xargs))
            #_debug "sub_commands " ${sub_commands[@]}

            for ((sub_idx = 0; sub_idx < ${#words[@]} - 1; sub_idx++)); do
                if [[ ${sub_commands[@]} =~ ${words[sub_idx]} ]]; then
                    sub_command=${words[sub_idx]}
                    break
                fi
            done
        fi
    done

    if [[ $sub_command == "" ]]; then
        COMPREPLY=($(compgen -W '${sub_commands[@]}' -- "$cur"))
        return 0
    fi

    #set -x
    for ((i = 0; i < ${#sub_commands[@]} - 1; i++)); do
        if [[ ${sub_commands[i]} == $sub_command ]]; then
            sub_opts=($(${words[0]} $command $sub_command 2>&1 | grep "^ " | grep -o " \-[-a-z]*" | sed 's/ //' | xargs))
            if [[ $cur == -* ]]; then
                COMPREPLY=($(compgen -W '${sub_opts[@]}' -- $cur))
                return 0
            else
                COMPREPLY=($(compgen -W '${sub_opts[@]}' -- $cur))
                return 0
            fi
        fi
    done

    COMPREPLY=($(compgen -W '${sub_commands[@]}' -- $cur))
    return 0

}

# 用 _subcmd 补全 chain33-cli
# _subcmd 通过当前光标所在的输入参数过滤可选的子命令
complete -F _chain33 chain33-cli guodun
