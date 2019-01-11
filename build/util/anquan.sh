#!/usr/bin/env bash
withdraw=$(../chain33-cli bty withdraw -e "ticket" -n "安全支出费用: $1" -a "$1")
send=$(../chain33-cli bty transfer -a "$1" -n "安全支出费用: $1" -t 1MY4pMgjpS2vWiaSDZasRhN47pcwEire32)
echo "cli wallet sign -d $withdraw -a 1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP >d:/a.txt"
echo "cli wallet sign -d $send -a 1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP >>d:/a.txt"
