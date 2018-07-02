#!/usr/bin/env bash
withdraw=$(./chain33-cli bty withdraw -e "ticket" -n "节点到400，空投$1" -a "$1")
send=$(./chain33-cli bty transfer -a "$1" -n "节点到400，空投$1" -t 1HkEim2QSMnoKVFr9gi9LmaHVvNsQru8u9)
echo "cli wallet sign -d $withdraw -a 1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP" >a.bat
echo "cli wallet sign -d $send -a 1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP" >>a.bat
