#!/usr/bin/env bash
guodun="$(../chain33-cli config config_tx -k token-blacklist -o add -v BTY --paraName "user.p.guodun.")"
fzmtest="$(../chain33-cli config config_tx -k token-blacklist -o add -v BTY --paraName "user.p.fzmtest.")"
echo "cli wallet sign -d $guodun -a 1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP -e 1h > d:/b.txt"
echo "cli wallet sign -d $fzmtest -a 1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP -e 1h >> d:/b.txt"
