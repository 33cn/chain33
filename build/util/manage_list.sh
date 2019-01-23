#!/usr/bin/env bash
guodun1=$(../chain33-cli config config_tx -c token-blacklist -o add -v BTY --paraName "user.p.$2.")
guodun2=$(../chain33-cli config config_tx -c token-finisher -o add -v "$1" --paraName "user.p.$2.")
echo "cli wallet sign -d $guodun1 -a 1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP -e 1h > d:/b.txt"
echo "cli wallet sign -d $guodun2 -a 1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP -e 1h >> d:/b.txt"
