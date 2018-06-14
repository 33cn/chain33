package main

/*
go build -o cryptogen
./cryptogen generate
默认采用当前目录下new.json作为配置文件，生成证书及密钥文件，生成位置为cmd/chain33/authdir/crypto
如果需要修改配置文件以及生成位置，可以用如下命令
./cryptogen generate --config=.... --output=
*/

/*
PAY ATTENTION:
配置里用户及组织名需要与chain33.toml对应，否则无法签名及验证
*/
