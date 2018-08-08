#!/usr/bin/env bash
/root/chain33 -f /root/chain33.toml &
sleep 120
/root/chain33 -f "$PARAFILE"
