#!/usr/bin/env bash
/root/chain33 -f "$1" &
sleep 120
/root/chain33 -f "$2"
