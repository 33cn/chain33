#!/usr/bin/env bash
fzmtest="$(../chain33-cli --paraName user.p.fzmtest. config config_tx -k lottery-creator -o add -v 15nn4D2ppUj8tmyHfdm8g4EvtqpBYUS7DM)"
echo "cli wallet sign -d $fzmtest -a 1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP -e 1h >> d:/b.txt"
