#!/bin/sh

signAndSend() {
    signed=`./cli wallet sign -d $unsigned -e 2m -k $1`
    length=${#signed}
    let length=length-2
    ./cli tx send -d `echo ${signed:1:length}`
}

if [ $1 == "help" ]
then
    echo "transfer        : create transfer transaction and send it"
    echo "withdraw        : create withdraw transaction and send it"
    echo "token_transfer  : create token transfer transaction and send it"
    echo "token_withdraw  : create token withdraw transaction and send it"
    echo "token_precreate : create token precreate transaction and send it"
    echo "token_finish    : create token finish transaction and send it"
    echo "token_revoke    : create token revoke transaction and send it"
    echo "trade_sell      : create tokem sell transaction and send it"
    echo "trade_buy       : create token buy transaction and send it"
    echo "trade_revoke    : create token buying revoke transaction and send it"
elif [ $1 == "transfer" ]
then
    read -t 120 -p "please input to_addr amount note priv_key: " to_addr amount note priv_key
    unsigned=`./cli bty transfer -t $to_addr -a $amount -n $note`
    signAndSend $priv_key
elif [ $1 == "withdraw" ]
then
    read -t 120 -p "please input executor amount note priv_key: " exec amount note priv_key
    unsigned=`./cli bty withdraw -e $exec -a $amount -n $note`
    signAndSend $priv_key
elif [ $1 == "token_transfer" ]
then
    read -t 120 -p "please input to_addr amount note token_symbol priv_key: " to_addr amount note symbol priv_key
    unsigned=`./cli token transfer -t $to_addr -a $amount -n $note -s $symbol`
    signAndSend $priv_key
elif [ $1 == "token_withdraw" ]
then
    read -t 120 -p "please input executor amount note token_symbol priv_key: " exec amount note symbol priv_key
    unsigned=`./cli token withdraw -e $exec -a $amount -n $note -s $symbol`
    signAndSend $priv_key
elif [ $1 == "token_precreate" ]
then
    read -t 120 -p "please input token_name token_symbol introduction owner_addr token_price total_number fee priv_key: " name symbol intro addr price total fee priv_key
    unsigned=`./cli token precreate -n $name -s $symbol -i $intro -a $addr -p $price -t $total -f $fee`
    signAndSend $priv_key
elif [ $1 == "token_finish" ]
then
    read -t 120 -p "please input owner_addr token_symbol fee priv_key: " addr symbol fee priv_key
    unsigned=`./cli token finish -a $addr -s $symbol -f $fee`
    signAndSend $priv_key
elif [ $1 == "token_revoke" ]
then
    read -t 120 -p "please input owner_addr token_symbol fee priv_key: " addr symbol fee priv_key
    unsigned=`./cli token revoke -a $addr -s $symbol -f $fee`
    signAndSend $priv_key
elif [ $1 == "trade_sell" ]
then
    read -t 120 -p "please input token_symbol amount min price total fee priv_key: " symbol amount min price total fee priv_key
    unsigned=`./cli trade sell -s $symbol -a $amount -m $min -p $price -t $total -f $fee`
    signAndSend $priv_key
elif [ $1 == "trade_buy" ]
then
    read -t 120 -p "please input sell_id count fee priv_key: " sell_id count fee priv_key
    unsigned=`./cli trade buy -s $sell_id -c $count -f $fee`
    signAndSend $priv_key
elif [ $1 == "trade_revoke" ]
then
    read -t 120 -p "please input sell_id fee priv_key: " sell_id fee priv_key
    unsigned=`./cli trade revoke -s $sell_id -f $fee`
    signAndSend $priv_key
fi
