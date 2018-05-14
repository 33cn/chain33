#!/bin/bash
# 根据自己情况，是否需要调试，使用下面两个命令
#./chain33 -f test.chain33.toml
dlv --headless=true --listen=:2345 --api-version=2 exec ./chain33 -- -f test.chain33.toml
