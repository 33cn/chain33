#!/usr/bin/env bash
guodun="$(../chain33-cli config  config_tx  -k token-finisher -o add -v 13KDST7ndBmzunGYAkCRabooWGRc95hM3W --paraName "user.p.guodun.")"
fzmtest="$(../chain33-cli config  config_tx  -k token-finisher -o add -v 16ReZHzMCGtPt8B7XbnZQ2jeXsPG9wEufh --paraName "user.p.fzmtest.")"
echo "cli wallet sign -d $guodun -a 1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP -e 1h > d:/b.txt"
echo "cli wallet sign -d $fzmtest -a 1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP -e 1h >> d:/b.txt"
