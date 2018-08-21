#!/usr/bin/env bash
ansible miner2 -m shell -a "$@" --key-file=~/Downloads/chain33.pem
