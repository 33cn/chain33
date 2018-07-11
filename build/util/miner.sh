#!/usr/bin/env bash
ansible miner -m shell -a "$@" --key-file=~/Downloads/chain33.pem
