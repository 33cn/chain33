# crypto

## 功能
* 支持 ed25519, secp256k1, sm2
* 统一的 PrivKey，Pubkey, Signature 接口, 详见 `crypto.go`

## 依赖
* sm2 编译依赖 gmssl 2.0 版本
* 安装：`bash ./deps/install_gmssl.sh`
