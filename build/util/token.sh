#!/usr/bin/env bash
percreate="$(../chain33-cli token precreate -s SCTC -n "Spider Chain" -i "让世界人人共享的区块链" -p 0.000001 -t 1000000000 -a 1Jdp8WYr4qXVyLseftY6SubSEgPTTEzaF -f 0.001)"
finish="$(../chain33-cli token finish -f 0.001 -a 1Jdp8WYr4qXVyLseftY6SubSEgPTTEzaF -s SCTC)"
echo "cli wallet sign -d $percreate -a 1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP  -e 1h > d:/a.txt"
echo "cli wallet sign -d $finish -a 1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP -e 1h >> d:/a.txt"
