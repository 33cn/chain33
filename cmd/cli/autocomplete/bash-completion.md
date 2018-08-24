# 命令行自动补全

  目前在ubuntu下用git 等软件你会发现按tab键有提示或补全的功能， 这个功能依赖一个软件包， 叫bash-completion。bash-completion 提供的能力是bash shell 的可编程补全。而chain33-cli 功能也越来越多，每加一步都看下帮助， 效率不高。 把这个能力加到chain33-cli里， 应该能提高不少效率。

## bash-completion

bash 自带命令行补全功能，但 由于 bash-completion 可编程，他可以带来更好的补全效果。 

不同的版本安装位置可能不一样， 但工作原理一致。

工作原理：
  1. 功能入口一个名为 bash_completion 的脚本， 在shell 初始化时加载。
  1. bash_completion脚本会加载 completions 目录下补全脚本。 
     1. completions 目录也可能有不一样的名字 如 bash_completion.d
     1. 目录下一个文件对应一个命令， 和 /etc/init.d/ /etc/rsyslog.d/ 等一样一样的
  
### 熟悉下环境

 先跟踪下我机器上的配置位置，熟悉下环境
``` 
# ~/.bashrc:                                # 在shell 初始化时加载
if ! shopt -oq posix; then
  if [ -f /usr/share/bash-completion/bash_completion ]; then
    . /usr/share/bash-completion/bash_completion
  elif [ -f /etc/bash_completion ]; then
    . /etc/bash_completion
  fi
fi

# /usr/share/bash-completion/               # 补全脚本目录
$ find /usr/share/bash-completion/ 
/usr/share/bash-completion/                 # 配置和脚本独立目录
/usr/share/bash-completion/bash_completion  # 入口脚本
/usr/share/bash-completion/completions      # 配置脚本目录
/usr/share/bash-completion/completions/man  # 一个个配置脚本， 我挑了几个看上去熟悉的
/usr/share/bash-completion/completions/git
/usr/share/bash-completion/completions/xhost
/usr/share/bash-completion/completions/route
/usr/share/bash-completion/completions/java
/usr/share/bash-completion/completions/docker-compose
/usr/share/bash-completion/completions/xxd
```

### 补全命令

bash 自带补全命令， 主要有下面一个命令提供支持 (man bash 可以看到说明)
 1. compgen： 从候选的单词列表中选出匹配的单词， 补全命令
 1. complete: 补全参数
 1. compopt:  修改每个名称指定的补全选项

compgen 演示
```
# 感觉好简单
$ compgen -W "z1 z2 z3 fz1 fz2 f3" z
z1
z2
z3

# 一下子不好了
compgen: 用法: compgen [-abcdefgjksuv] [-o 选项]  [-A 动作] [-G 全局模式] [-W 词语列表]  [-F 函数] [-C 命令] [-X 过滤模式] [-P 前缀] [-S 后缀] [词语]
```

complete 演示
```
# 是不是很简单
$ complete -W "account block bty trade token" chain33-cli
linj@linj-TM1701:~$ chain33-cli t
token  trade  

# 再次打击
complete: 用法: complete [-abcdefgjksuv] [-pr] [-DE] [-o 选项] [-A 动作] [-G 全局模式] [-W 词语列表]  [-F 函数] [-C 命令] [-X 过滤模式] [-P 前缀] [-S 后缀] [名称 ...]
```

### bash 支持补全相关的内置变量

用过c的应该对 __FILE__ __LINE__ 熟悉， bash 为了支持补全同样内置了相关变量

```
前缀 COMP, 看后面部分表示变量意思

补全函数的输出
 ${COMPREPLY}   	一个数组变量，bash从这个变量中读取可编程补全所调用的shell函数生成的补全条目。

补全函数的输入
 ${COMP_WORDS}          一个数组变量，包含当前命令行的每个单词 
 ${COMP_CWORD}          光标位置的单词“${COMP_WORDS}”中的下标. 为什么是光标， 输入命令可以可以回到前面修改命令

补全函数的其他变量
 ${COMP_LINE}		当前命令行. 切分成 WORDS前的原始命令行
 ${COMP_POINT}          当前光标位置相对于当前命令行开头的下标。 命令行里的第N个字符

补全模式
 ${COMP_KEY}            触发当前补全函数的键，或键序列中的最后一个键。 一直是 9
 ${COMP_TYPE}           一个整数值，与触发调用补全函数时试图进行补全的类型相对应。 一个tab 9, 多个tab 63

分割符
 ${COMP_WORDBREAKS}     “readline”库进行单词补全时用作单词分隔的字符，如果没有设置这个变量，即使以后进行重置，它也会失去特殊作用。
```

```
linj@linj-TM1701:~$ chain33-cli account 

declare -a COMP_WORDS='([0]="chain33-cli" [1]="account" [2]="")'
declare -- COMP_CWORD="2"
declare -- COMP_LINE="chain33-cli account "
declare -- COMP_WORDBREAKS=" 	
\"'><=;|&(:"
declare -- COMP_KEY="9"
declare -- COMP_POINT="20"
declare -- COMP_TYPE="9"
```

### 演示程序1 子命令补全

chain33-cli 参数补全
```
#!/bin/bash
# 通过chain33-cli 的help 找到一级的子命令
subcmd_list=("account" "block" "bty" "close" "coins" "config" "evm" "exec" "hashlock" "help" "mempool" "net" "privacy" "relay" "retrieve" "seed" "send" "stat" "ticket" "token" "trade" "tx" "version" "wallet")
#
function _subcmd() {
  local cur
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  COMPREPLY=( $(compgen -W "${subcmd_list[*]}" -- ${cur}) )
  return 0
}

# 用 _subcmd 补全 chain33-cli
# _subcmd 通过当前光标所在的输入参数过滤可选的子命令
complete -F _subcmd chain33-cli
```

试用一下
```
linj@linj-TM1701:~$ . subcmd.bash  
linj@linj-TM1701:~$ ./chain33/chain33-cli 
account   bty       coins     evm       hashlock  mempool   privacy   retrieve  send      ticket    trade     version   
block     close     config    exec      help      net       relay     seed      stat      token     tx        wallet    
linj@linj-TM1701:~$ ./chain33/chain33-cli t
ticket  token   trade   tx      
```

### 生效

```
linj@linj-TM1701:~$ sudo install subcmd.bash  /usr/share/bash-completion/completions/chain33-cli
# 重新开个窗口就有用了
linj@linj-TM1701:~$ ./chain33/chain33-cli 
account   bty       coins     evm       hashlock  mempool   privacy   retrieve  send      ticket    trade     version   
block     close     config    exec      help      net       relay     seed      stat      token     tx        wallet    
```

### 给chain33-cli 做个 bash-completion


 地址: https://gitlab.33.cn/linj/chain33-cli-completion

演示
```
linj@linj-TM1701:~$ ./chain33/chain33-cli 
account   bty       coins     evm       hashlock  mempool   privacy   retrieve  send      ticket    trade     version   
block     close     config    exec      help      net       relay     seed      stat      token     tx        wallet    
linj@linj-TM1701:~$ ./chain33/chain33-cli b
block  bty    
linj@linj-TM1701:~$ ./chain33/chain33-cli bty 
priv2priv  priv2pub   pub2priv   send       transfer   txgroup    withdraw   
linj@linj-TM1701:~$ ./chain33/chain33-cli bty t
transfer  txgroup   
linj@linj-TM1701:~$ ./chain33/chain33-cli bty transfer -
-a        --amount  -h        --help    -n        --note    --para    --rpc     -t        --to  
```
