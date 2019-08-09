module github.com/33cn/chain33

go 1.12

replace (
	cloud.google.com/go => github.com/googleapis/google-cloud-go v0.43.1-0.20190808215159-84f66600e42d
	golang.org/x/crypto => github.com/golang/crypto v0.0.0-20190313024323-a1f597ede03a
	golang.org/x/exp => github.com/golang/exp v0.0.0-20190731235908-ec7cb31e5a56
	golang.org/x/image => github.com/golang/image v0.0.0-20190227222117-0694c2d4d067
	golang.org/x/lint => github.com/golang/lint v0.0.0-20190409202823-959b441ac422
	golang.org/x/mobile => github.com/golang/mobile v0.0.0-20190312151609-d3739f865fa6
	golang.org/x/net => github.com/golang/net v0.0.0-20190318221613-d196dffd7c2b
	golang.org/x/oauth2 => github.com/golang/oauth2 v0.0.0-20190523182746-aaccbc9213b0
	golang.org/x/sync => github.com/golang/sync v0.0.0-20190227155943-e225da77a7e6
	golang.org/x/sys => github.com/golang/sys v0.0.0-20190318195719-6c81ef8f67ca
	golang.org/x/text => github.com/golang/text v0.3.0
	golang.org/x/time => github.com/golang/time v0.0.0-20190308202827-9d24e82272b4
	golang.org/x/tools => github.com/golang/tools v0.0.0-20190529010454-aa71c3f32488
	google.golang.org/api => github.com/googleapis/google-api-go-client v0.7.0
	google.golang.org/appengine => github.com/golang/appengine v1.6.1-0.20190515044707-311d3c5cf937
	google.golang.org/genproto => github.com/google/go-genproto v0.0.0-20190522204451-c2c4e71fbf69
	google.golang.org/grpc => github.com/grpc/grpc-go v1.21.0
)

require (
	github.com/AndreasBriese/bbloom v0.0.0-20180913140656-343706a395b7 // indirect
	github.com/BurntSushi/toml v0.3.1
	github.com/NebulousLabs/Sia v1.3.7
	github.com/NebulousLabs/entropy-mnemonics v0.0.0-20170316012907-7b01a644a636 // indirect
	github.com/NebulousLabs/errors v0.0.0-20171229012116-7ead97ef90b8 // indirect
	github.com/NebulousLabs/fastrand v0.0.0-20180208210444-3cf7173006a0 // indirect
	github.com/NebulousLabs/merkletree v0.0.0-20181025040823-2a1d1d1dc33c // indirect
	github.com/XiaoMi/pegasus-go-client v0.0.0-20181029071519-9400942c5d1c
	github.com/apache/thrift v0.0.0-20171203172758-327ebb6c2b6d // indirect
	github.com/boltdb/bolt v1.3.1 // indirect
	github.com/btcsuite/btcd v0.0.0-20181013004428-67e573d211ac
	github.com/coreos/bbolt v1.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dchest/blake256 v1.0.0 // indirect
	github.com/decred/base58 v1.0.0
	github.com/dgraph-io/badger v1.5.4
	github.com/dgryski/go-farm v0.0.0-20180109070241-2de33835d102
	github.com/fortytw2/leaktest v1.3.0 // indirect
	github.com/go-stack/stack v1.8.0
	github.com/golang/protobuf v1.3.2
	github.com/golang/snappy v0.0.0-20180518054509-2e65f85255db // indirect
	github.com/haltingstate/secp256k1-go v0.0.0-20151224084235-572209b26df6
	github.com/hashicorp/golang-lru v0.5.1
	github.com/huin/goupnp v1.0.0
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jackpal/go-nat-pmp v1.0.1
	github.com/kr/pretty v0.1.0 // indirect
	github.com/mattn/go-colorable v0.0.9
	github.com/mattn/go-isatty v0.0.4 // indirect
	github.com/mr-tron/base58 v1.1.0
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/pkg/errors v0.8.0
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rs/cors v1.6.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3 // indirect
	github.com/stretchr/objx v0.1.1 // indirect
	github.com/stretchr/testify v1.2.2
	github.com/syndtr/goleveldb v0.0.0-20181105012736-f9080354173f
	github.com/tjfoc/gmsm v0.0.0-20171124023159-98aa888b79d8
	golang.org/x/crypto v0.0.0-20190510104115-cbcb75029529
	golang.org/x/net v0.0.0-20190620200207-3b0461eec859
	golang.org/x/sys v0.0.0-20190624142023-c5567b49c5d0
	google.golang.org/grpc v1.21.1
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127
	gopkg.in/go-playground/webhooks.v5 v5.2.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0-20170531160350-a96e63847dc3
	gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637 // indirect
	gopkg.in/yaml.v2 v2.2.2 // indirect
)
