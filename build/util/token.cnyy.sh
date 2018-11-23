#!/usr/bin/env bash
percreate="$(../chain33-cli token precreate -s CNYY -n "CNY Coins" -i "" -p 0.000001 -t 1000000000 -a 1MY4pMgjpS2vWiaSDZasRhN47pcwEire32 -f 0.001)"
finish="$(../chain33-cli token finish -f 0.001 -a 1MY4pMgjpS2vWiaSDZasRhN47pcwEire32 -s CNYY)"
echo "cli wallet sign -d $percreate -a 1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP  -e 1h > d:/a.txt"
echo "cli wallet sign -d $finish -a 1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP -e 1h >> d:/a.txt"
