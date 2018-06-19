withdraw="`./chain33-cli bty withdraw -n "空投币20000，withdraw到合约地址" -a 20000 -e "ticket"`"
send="`./chain33-cli bty transfer -n "空投币20000，发送到空投地址" -a 20000 -t 1HkEim2QSMnoKVFr9gi9LmaHVvNsQru8u9`"
echo "cli wallet sign -d $withdraw -a 1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP -e 1h > d:/b.txt"
echo "cli wallet sign -d $send -a 1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP -e 1h >> d:/b.txt"
