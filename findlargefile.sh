#!/bin/bash
#set -x

# Shows you the largest objects in your repo's pack file.
# Written for osx.
#
# @see https://stubbisms.wordpress.com/2009/07/10/git-script-to-show-largest-pack-objects-and-trim-your-waist-line/
# @author Antony Stubbs

# set the internal field separator to line break, so that we can iterate easily over the verify-pack output
IFS=$'\n'

# list all objects including their size, sort by size, take top 10
objects=$(git verify-pack -v .git/objects/pack/pack-*.idx | grep -v chain | sort -k3nr | head -n 15)

echo "All sizes are in kB's. The pack column is the size of the object, compressed, inside the pack file."

# 39526  18752  2c8c0fa0a853bfa135f03f1bd30c905336425caa  cmd/chain33/test
# 30702  9463   240191b5561e737e50283045972befb4a13275d4  |/home/hugo/dev/src/github.com/33cn/chain33/build/cli
# 26721  8842   33064554018a49d4f0ff9db680241efaa67e5b7f  cmd/cli/cli
# 11979  4918   0c300a7b61218271e666f3246f36e8f1915b0d57  |/home/hugo/dev/src/github.com/33cn/chain33/build/chain33
# 4834   1113   9ec4f3d49403e8b9dd46885031a92e23af3828b9  vendor/golang.org/x/text/collate/tables.go
# 3563   1034   eb297e33f34b7ba69cdbf1fd720a97a74fa8aed9  vendor/golang.org/x/text/language/display/tables.go
# 3561   1033   6493357efae3b213d709cfb5f50a7e037e5972ca  vendor/golang.org/x/text/language/display/tables.go
# 3383   592    3b28d8f0e670f425f3248e315e23984942b100f6  vendor/golang.org/x/tools/third_party/typescript/typescript.js
# 2584   865    4079adb3cb96475e4499ab150abbc3fe193ee0ef  vendor/github.com/haltingstate/secp256k1-go/secp256k1-go2/z_consts.go
# 1161   879    1b1b8179e167f5e2d1d5d2c189b9bfec70c387a4  vendor/github.com/btcsuite/btcd/btcec/secp256k1.go
# 1105   234    99c4926d451e549bc1be6a78946f1ad0988358be  vendor/golang.org/x/text/unicode/runenames/tables.go

history="2c8c0fa0a853bfa135f03f1bd30c905336425caa 240191b5561e737e50283045972befb4a13275d4 33064554018a49d4f0ff9db680241efaa67e5b7f \
         0c300a7b61218271e666f3246f36e8f1915b0d57 9ec4f3d49403e8b9dd46885031a92e23af3828b9 eb297e33f34b7ba69cdbf1fd720a97a74fa8aed9  \
         6493357efae3b213d709cfb5f50a7e037e5972ca 3b28d8f0e670f425f3248e315e23984942b100f6 4079adb3cb96475e4499ab150abbc3fe193ee0ef \
         1b1b8179e167f5e2d1d5d2c189b9bfec70c387a4 99c4926d451e549bc1be6a78946f1ad0988358be"

oversize="false"
limitsize=2000 # limit 2M

output="size,pack,SHA,location"
allObjects=$(git rev-list --all --objects)
for y in $objects; do
    # extract the size in bytes
    size=$(($(echo "$y" | cut -f 5 -d ' ') / 1024))
    # extract the compressed size in bytes
    compressedSize=$(($(echo "$y" | cut -f 6 -d ' ') / 1024))
    # extract the SHA
    sha=$(echo "$y" | cut -f 1 -d ' ')
    if [[ ! $history =~ $sha ]]; then
        if [ $size -ge $limitsize ]; then
            echo "over size file = $sha"
            oversize="true"
        fi
    fi
    # find the objects location in the repository tree
    other=$(echo "${allObjects}" | grep "$sha")
    #lineBreak=`echo -e "\n"`
    output="${output}\\n${size},${compressedSize},${other}"
done

echo -e "$output" | column -t -s ', '
if [ "$oversize" == "true" ]; then
    echo "there are files over 2M size committed!!!"
    exit 1
fi
