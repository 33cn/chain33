changelog

# [1.70.0](https://github.com/33cn/chain33/compare/v1.69.0...v1.70.0) (2026-09-02)


### Bug Fixes

* **account:** add safeAdd overflow protection to exec account balance additions ([9119063](https://github.com/33cn/chain33/commit/91190630427a99113dd907ea824211da36d4169a))
* **account:** reject non-positive amount in GenesisInit ([76a6573](https://github.com/33cn/chain33/commit/76a65737da7528d004931e034731d36ec0aa33a6))
* **account:** reject self-transfer across eth address case variants in ExecTransfer ([f777ab9](https://github.com/33cn/chain33/commit/f777ab9c35740d94998ca31fd133ab1467d7547e))
* **account:** reject self-transfer across eth address case variants in ExecTransferFrozen ([6ea076f](https://github.com/33cn/chain33/commit/6ea076fd1cdd9bea01bf72b75b792a04f7e45275))
* **account:** stop sharing key buffer in accountReadKey ([85c65a9](https://github.com/33cn/chain33/commit/85c65a96ba7b0d995b27926fa0a3a62fc5947b19))
* add integration build tag to testnode tests, fix IsRecordFaultErr test ([a6b998f](https://github.com/33cn/chain33/commit/a6b998f37f8852918342c5436fc6e36769ab1cd8))
* address 20 bugs from codebase review (security, correctness, stability) ([7198f44](https://github.com/33cn/chain33/commit/7198f44b3a32a67e5170fde126cd1bda6c42a376))
* address check_fmt gosec and goimports failures ([23342cd](https://github.com/33cn/chain33/commit/23342cdf2b2b270df27d4d84af43c900d0ab2fa8)), closes [#nosec](https://github.com/33cn/chain33/issues/nosec)
* address CI fmt/vet failures ([bb3fb41](https://github.com/33cn/chain33/commit/bb3fb418d4f8cb66d73c7840928481014249f2f7))
* CBC 随机 IV 后修正下游消费者与测试 ([b707573](https://github.com/33cn/chain33/commit/b70757355f324d77bcc0f745769ebe36a06e7dbd))
* **ci:** exclude gg18 integration test from coverage to stop flaky timeouts ([337a855](https://github.com/33cn/chain33/commit/337a855bd25b0596ee9bd6140dca6cbdfd84f2ac)), closes [#1378](https://github.com/33cn/chain33/issues/1378)
* **ci:** switch semantic-release preset from jshint to angular ([a22cff9](https://github.com/33cn/chain33/commit/a22cff9b6d9ff82ce50f1a96de9348cb7bd7e4b1))
* **ci:** use admin GH_TOKEN to bypass protected branch ([2d8612f](https://github.com/33cn/chain33/commit/2d8612fdefc8d2ae67a6571dc13214d8f6b566d9))
* **ci:** use built-in GITHUB_TOKEN for semantic-release ([95adb16](https://github.com/33cn/chain33/commit/95adb16c9fea16a752db4c93cc8d0723853e3ece))
* **cli:** exit non-zero on RPC failure so scripts can detect errors ([250680c](https://github.com/33cn/chain33/commit/250680cf563bcccc44244127b4c59163aa69e8f6))
* **coins:** add missing ExecDelLocal_Genesis to rollback genesis receiver stats ([aab7c97](https://github.com/33cn/chain33/commit/aab7c9738f7922022318387647d8a6a162c48152))
* **coins:** reject non-positive amount in genesis tx at executor entry ([8121769](https://github.com/33cn/chain33/commit/812176982f0834e5b5d03e210febc4110a4ea6dc))
* drop toolchain directive that breaks lint job ([62c9a20](https://github.com/33cn/chain33/commit/62c9a200c129d41483c27536ff6f6b90260b6e26))
* **ethrpc:** add address field to support fetching logs from called contracts ([#1344](https://github.com/33cn/chain33/issues/1344)) ([2a05aa3](https://github.com/33cn/chain33/commit/2a05aa3e0b57c677b4a0cc013d975a0a3393c664))
* **ethrpc:** add fallback address for evm logs ([f6402bc](https://github.com/33cn/chain33/commit/f6402bc6ffd6da085008b07f4cabc0e7158378a9))
* **ethrpc:** backfill historical logs with missing address field ([404c9d7](https://github.com/33cn/chain33/commit/404c9d75d3849b6c895d1b8b7e8ec1177535b031))
* **ethrpc:** checkIPWhitelist treat '*' as allow-all, consistent with InitIPWhitelist ([528b6ef](https://github.com/33cn/chain33/commit/528b6ef7df0463074903ceadd14563e40d5aeb61))
* **ethrpc:** guard nil deref and empty-slice index in eth handlers ([95622fd](https://github.com/33cn/chain33/commit/95622fd12d1aa5c6c5fe969034b63e87c7acca4c))
* **ethrpc:** prevent uint64→int64 overflow in AssembleChain33Tx ([cbeb5f8](https://github.com/33cn/chain33/commit/cbeb5f831d2be72bb22d7b08de76263cb6a556c9))
* **ethrpc:** reject value overflowing uint64 in EstimateGas ([16f95ae](https://github.com/33cn/chain33/commit/16f95ae8cbb33d4e26cfe17dd393acb775b58fec))
* ForkParaFee default to MaxHeight to avoid breaking para consensus ([ac7af31](https://github.com/33cn/chain33/commit/ac7af31a9c610c79629ff74adfbe455d5222ea6d))
* gofmt formatting for p2p, rpc, and gg18 test files ([3e52242](https://github.com/33cn/chain33/commit/3e5224299cccba7e324e2ed0273f762d0122648a))
* gofmt formatting for types/evm_event.pb.go comment spacing ([4926ac0](https://github.com/33cn/chain33/commit/4926ac035f75b9f973c3f68a495cf1deb355c8ee))
* gofmt whitespace formatting ([3862ddd](https://github.com/33cn/chain33/commit/3862ddd4c900ebaacad5fdf4072e203cb94ecfb4))
* increase TSS integration test timeout for CI ([b6b4a46](https://github.com/33cn/chain33/commit/b6b4a46e43c1d33ccaa3b8bc5af9cf3c2f81a993))
* make gg18 integration test deterministic, remove flaky timeout ([aaf65cf](https://github.com/33cn/chain33/commit/aaf65cf7db85d600a9bc0a05700f303c8f46a1c8)), closes [#1361](https://github.com/33cn/chain33/issues/1361)
* make TestConnManager deterministic, poll for routing-table convergence ([f2b22f7](https://github.com/33cn/chain33/commit/f2b22f7a409766f864cc97ad286fdd36e5fa232c))
* make TimeCache.Has() and List() respect expiration time ([d22471f](https://github.com/33cn/chain33/commit/d22471f44f16e3e230be0d06ee27cf8cf9eb3dad))
* **p2p/dht:** move taskGroup.Add out of goroutines to fix data race ([90e3844](https://github.com/33cn/chain33/commit/90e3844434c27e06d50fdb5030ac2c03ec5e8431))
* **queue:** ensure proper channel closure by associating done channels with subscriptions ([3df23fc](https://github.com/33cn/chain33/commit/3df23fcadcc8f2fb996a1a2d9283d23653da7637))
* **queue:** prevent deadlock and improve channel closing in queue management ([c529bfe](https://github.com/33cn/chain33/commit/c529bfe41fd6f91e900bfa0bc9319cc8600bcd0e))
* read task fields to satisfy structcheck ([9eabcd5](https://github.com/33cn/chain33/commit/9eabcd5d79c8dbfe4a0ab79993406e02f40a64f5))
* remove flaky TestWaitGroupAdd_AfterGoroutine_Race ([5d04a19](https://github.com/33cn/chain33/commit/5d04a191ecac193f2aecc17800e7b01c1d2f7b44))
* replace broken codecov with shields.io badge, remove codecov CI step ([1d3ff58](https://github.com/33cn/chain33/commit/1d3ff58fe6a189067e64872f059e7f22b86dc4f9))
* TestConnManager re-adds peer each poll to beat DHT eviction ([40a62e4](https://github.com/33cn/chain33/commit/40a62e4b16bcb22a0822dfd3e07131e93aafcd3d))
* **tss:** handle nil configuration for TSS service initialization ([9e2053c](https://github.com/33cn/chain33/commit/9e2053c4be8a5b8fda7c6fe841b3692f3f5ad0b6))
* **types:** close chainID replay loopholes on txgroup and zero-minfee paths ([dc480c0](https://github.com/33cn/chain33/commit/dc480c01005082413dcd76e8d67306035476e10a))
* **wallet:** CBCDecrypterPrivkey support 64-byte ed25519 keys with random IV ([7986213](https://github.com/33cn/chain33/commit/79862130543d4765432d7cea7169b375d8b6afe3))
* **wallet:** derive encryption key via pbkdf2 instead of raw password ([98ac33e](https://github.com/33cn/chain33/commit/98ac33ee5da0f23157e134bd25fc6f4e7fe5a7c0))


### Features

* add [blacklist] toml config for account blacklist ([3f8f145](https://github.com/33cn/chain33/commit/3f8f145b215ecc1c67b0eda8a5c37aceb9ea9187))
* add account k2addr CLI command ([2e58d0b](https://github.com/33cn/chain33/commit/2e58d0bb5a54799dd8ad97e25d718a71c19e75d0))
* add Chain33Config.SetFork for programmatic system fork override ([1edb5ac](https://github.com/33cn/chain33/commit/1edb5ac8d4fd34aaa9a5a7887b4d90468dd6dc22))
* add consensus-level account blacklist to block attacker addresses ([aa71469](https://github.com/33cn/chain33/commit/aa71469c0974ea20d69dc01396d7b46d50d3ed40))
* TSS threshold signing (GG18) and DHT integration ([#1341](https://github.com/33cn/chain33/issues/1341)) ([3086d49](https://github.com/33cn/chain33/commit/3086d49f35ec5189f8b948d61240249c9be3cc1c))


### Reverts

* Revert "fix: ForkParaFee default to MaxHeight to avoid breaking para consensus" ([b38fbb3](https://github.com/33cn/chain33/commit/b38fbb311a320c064d4da3eb23e6739b6b0ad2b9))

<a name="1.69.0"></a>
# [1.69.0](https://github.com/33cn/chain33/compare/v1.68.2...v1.69.0) (2024-05-31)


### Features

* import avalanche snowman engine ([25b1b0c](https://github.com/33cn/chain33/commit/25b1b0c))

<a name="1.68.2"></a>
## [1.68.2](https://github.com/33cn/chain33/compare/v1.68.1...v1.68.2) (2024-04-12)


### Bug Fixes

* add format eth address fork ([ee8af86](https://github.com/33cn/chain33/commit/ee8af86))

<a name="1.68.1"></a>
## [1.68.1](https://github.com/33cn/chain33/compare/v1.68.0...v1.68.1) (2023-09-12)


### Bug Fixes

* update go-libp2p dependency to v0.23.4 ([88be560](https://github.com/33cn/chain33/commit/88be560))

<a name="1.68.0"></a>
# [1.68.0](https://github.com/33cn/chain33/compare/v1.67.5...v1.68.0) (2023-02-17)


### Features

* add state committer interface in consensus module(#1268) ([8d907b2](https://github.com/33cn/chain33/commit/8d907b2)), closes [#1268](https://github.com/33cn/chain33/issues/1268)

<a name="1.67.5"></a>
## [1.67.5](https://github.com/33cn/chain33/compare/v1.67.4...v1.67.5) (2022-12-15)


### Bug Fixes

* fix issue #1279 ([f76cfda](https://github.com/33cn/chain33/commit/f76cfda)), closes [#1279](https://github.com/33cn/chain33/issues/1279)

<a name="1.67.4"></a>
## [1.67.4](https://github.com/33cn/chain33/compare/v1.67.3...v1.67.4) (2022-10-11)


### Bug Fixes

* update wallet recover script doc ([ee44bf0](https://github.com/33cn/chain33/commit/ee44bf0))

<a name="1.67.3"></a>
## [1.67.3](https://github.com/33cn/chain33/compare/v1.67.2...v1.67.3) (2022-05-27)


### Bug Fixes

* add address key format fork ([3a3af20](https://github.com/33cn/chain33/commit/3a3af20))

<a name="1.67.2"></a>
## [1.67.2](https://github.com/33cn/chain33/compare/v1.67.1...v1.67.2) (2022-04-18)


### Bug Fixes

* add fork chain detection(#1236) ([192058e](https://github.com/33cn/chain33/commit/192058e)), closes [#1236](https://github.com/33cn/chain33/issues/1236)

<a name="1.67.1"></a>
## [1.67.1](https://github.com/33cn/chain33/compare/v1.67.0...v1.67.1) (2022-03-28)


### Bug Fixes

* fix 32 bit machine build ([a494d18](https://github.com/33cn/chain33/commit/a494d18))
* update patch version ([c0c2d0b](https://github.com/33cn/chain33/commit/c0c2d0b))

<a name="1.67.0"></a>
# [1.67.0](https://github.com/33cn/chain33/compare/v1.66.5...v1.67.0) (2022-03-21)


### Features

* add multiple address format support(#1181) ([659f342](https://github.com/33cn/chain33/commit/659f342)), closes [#1181](https://github.com/33cn/chain33/issues/1181)

<a name="1.66.5"></a>
## [1.66.5](https://github.com/33cn/chain33/compare/v1.66.4...v1.66.5) (2022-02-18)


### Bug Fixes

* fix reset transaction ([96eccb0](https://github.com/33cn/chain33/commit/96eccb0))

<a name="1.66.4"></a>
## [1.66.4](https://github.com/33cn/chain33/compare/v1.66.3...v1.66.4) (2022-01-19)


### Bug Fixes

* fix merge iterator reverse list(#1211) ([fd5b2dd](https://github.com/33cn/chain33/commit/fd5b2dd)), closes [#1211](https://github.com/33cn/chain33/issues/1211)





## [1.66.4](https://github.com/33cn/chain33/compare/v1.66.3...v1.66.4) (2022-01-19)

<a name="1.66.3"></a>
## [1.66.3](https://github.com/33cn/chain33/compare/v1.66.2...v1.66.3) (2022-01-14)


### Bug Fixes

* fix list table primary key(#1203) ([b82008a](https://github.com/33cn/chain33/commit/b82008a)), closes [#1203](https://github.com/33cn/chain33/issues/1203)





## [1.66.3](https://github.com/33cn/chain33/compare/v1.66.2...v1.66.3) (2022-01-14)

<a name="1.66.2"></a>
## [1.66.2](https://github.com/33cn/chain33/compare/v1.66.1...v1.66.2) (2022-01-10)


### Bug Fixes

* fix localdb and statedb del key ([087cdaa](https://github.com/33cn/chain33/commit/087cdaa))





## [1.66.2](https://github.com/33cn/chain33/compare/v1.66.1...v1.66.2) (2022-01-10)

<a name="1.66.1"></a>
## [1.66.1](https://github.com/33cn/chain33/compare/v1.66.0...v1.66.1) (2021-12-28)


### Bug Fixes

* close a closed queue ([971ba67](https://github.com/33cn/chain33/commit/971ba67))





## [1.66.1](https://github.com/33cn/chain33/compare/v1.66.0...v1.66.1) (2021-12-28)

<a name="1.66.0"></a>
# [1.66.0](https://github.com/33cn/chain33/compare/v1.65.5...v1.66.0) (2021-12-28)


### Features

* release 1.66.0 ([edefa66](https://github.com/33cn/chain33/commit/edefa66))





# [1.66.0](https://github.com/33cn/chain33/compare/v1.65.5...v1.66.0) (2021-12-28)


### Performance Improvements

* update key generator ([5c5d864](https://github.com/33cn/chain33/commit/5c5d86450bafd3edcad471591e11afead7fad0fe))

## [1.65.5](https://github.com/33cn/chain33/compare/v1.65.4...v1.65.5) (2021-10-19)


### Bug Fixes

* Adjust github action ([9246b83](https://github.com/33cn/chain33/commit/9246b830a84d9a52ae140d09754ed71291c81548))

## [1.65.4](https://github.com/33cn/chain33/compare/v1.65.3...v1.65.4) (2021-10-15)


### Bug Fixes

* **doc:** release 1.65.4 ([4f53148](https://github.com/33cn/chain33/commit/4f531488049a79640121ba5d950166939dedaebd))

# [1.66.0](https://github.com/33cn/chain33/compare/v1.65.2...v1.66.0) (2021-10-15)


### Bug Fixes

* 🐛version control: Add github action for auto publish release and tag version ([22642e1](https://github.com/33cn/chain33/commit/22642e187aecaa21d5904c5d82e459fc6a0f72c4))
* **chain:** add ticker stop method ([aac09d4](https://github.com/33cn/chain33/commit/aac09d45e0ee64f77e81cc36da569444da511ccd))
* chunk key when delete ([f559759](https://github.com/33cn/chain33/commit/f5597596f5f20e02c29eb699d3cefa53cd42b95b))
* chunkStatusCacheMutex unlock bug ([cd7bdc8](https://github.com/33cn/chain33/commit/cd7bdc8111538c4ac1926f54ce85f78071624fe5))
* close pubsub ([aca60b8](https://github.com/33cn/chain33/commit/aca60b86d6e5d8b613d10e15f20181e3b445c2eb))
* close stream without delay after reading ([678ff0a](https://github.com/33cn/chain33/commit/678ff0a78e32d5be98cd123717f1f396467dac4b))
* compact block body in localdb ([6c6a0aa](https://github.com/33cn/chain33/commit/6c6a0aab49ca885ffb60a25ac79c2667ef3f573e))
* compact db ([c704b6a](https://github.com/33cn/chain33/commit/c704b6a3fcb501f546a19a204247cbbc6db69d3f))
* compact db after deleting chunk ([6371fcf](https://github.com/33cn/chain33/commit/6371fcf336241e5430c30a94c303c1dad84c2ac2))
* dht unit test ([1566a58](https://github.com/33cn/chain33/commit/1566a58369fe4c8f4307dc2fcd156840151ddb6d))
* do not push searched peers into peer channel ([1523267](https://github.com/33cn/chain33/commit/152326733cbaba07e0af7d894b2036481db1b58c))
* fetch chunk routine bug ([1a5b5be](https://github.com/33cn/chain33/commit/1a5b5bef855d1966e7944b146a4d2ad456651619))
* fix ci and add manually auto publish release ([28febf7](https://github.com/33cn/chain33/commit/28febf7face3b8842641c72751150a5bd550017a))
* ignore peers without addr when saving peers ([7a15e22](https://github.com/33cn/chain33/commit/7a15e22a5dcf62d426c304dabbdedd6d8ded0264))
* index bug ([fd6a114](https://github.com/33cn/chain33/commit/fd6a114b69aa169f542f1288d532aeb40cd2f4b1))
* libp2p stream leak ([53988d7](https://github.com/33cn/chain33/commit/53988d78a8335ff4ae95a588ea3a970f38acb80c))
* mustFetchChunk context ([c00ce7d](https://github.com/33cn/chain33/commit/c00ce7d738e38b0d7563975f46b9bca404ef558e))
* push to channel unblockedly ([4d884f1](https://github.com/33cn/chain33/commit/4d884f1bf525ffa9296f097763f775cf2a7378e5))
* query chunk records from peers in routing tableif there is no given peer ([5362c0a](https://github.com/33cn/chain33/commit/5362c0a273b4ffc001343f1ab80ba0779b188d07))
* query public ip ([a73dd61](https://github.com/33cn/chain33/commit/a73dd61ebd6ef96d7c67beb7862e10e8ad96b006))
* refresh peer info ([7e95977](https://github.com/33cn/chain33/commit/7e95977ac73d028d776b4556c42b00c559cf5ecb))
* retry to exec block when error occurs in chunk download mode ([330f8b2](https://github.com/33cn/chain33/commit/330f8b2a03a92782314bf18a412d883f039338ed))
* set dht to server mode ([8de2fb9](https://github.com/33cn/chain33/commit/8de2fb98df00900e3d9bc8bbd9b84d35c53cd070))
* unit test ([2269a15](https://github.com/33cn/chain33/commit/2269a1596766ff0bc95e862467a1b657a2bebf93))
* use peerlist instead of best peers ([198f580](https://github.com/33cn/chain33/commit/198f5800ff818b03e19d5d37360655e07f7eacbc))


### Features

* **deps:** bump github.com/decred/base58 from 1.0.2 to 1.0.3 ([cfbde5e](https://github.com/33cn/chain33/commit/cfbde5ef9e4acca23a7e82a47c777f87779969c2))
* **deps:** bump github.com/influxdata/influxdb from 1.7.9 to 1.9.5 ([162a75c](https://github.com/33cn/chain33/commit/162a75c2457d09a7fb1e99a0ae0bd8c29d7a83a0))
* **deps:** bump github.com/multiformats/go-multiaddr ([44b7c10](https://github.com/33cn/chain33/commit/44b7c10f9c6632b188e85ee0b646e06443547c4f))

# [1.66.0](https://github.com/33cn/chain33/compare/v1.65.2...v1.66.0) (2021-10-15)


### Bug Fixes

* 🐛version control: Add github action for auto publish release and tag version ([22642e1](https://github.com/33cn/chain33/commit/22642e187aecaa21d5904c5d82e459fc6a0f72c4))
* **chain:** add ticker stop method ([aac09d4](https://github.com/33cn/chain33/commit/aac09d45e0ee64f77e81cc36da569444da511ccd))
* chunk key when delete ([f559759](https://github.com/33cn/chain33/commit/f5597596f5f20e02c29eb699d3cefa53cd42b95b))
* chunkStatusCacheMutex unlock bug ([cd7bdc8](https://github.com/33cn/chain33/commit/cd7bdc8111538c4ac1926f54ce85f78071624fe5))
* close pubsub ([aca60b8](https://github.com/33cn/chain33/commit/aca60b86d6e5d8b613d10e15f20181e3b445c2eb))
* close stream without delay after reading ([678ff0a](https://github.com/33cn/chain33/commit/678ff0a78e32d5be98cd123717f1f396467dac4b))
* compact block body in localdb ([6c6a0aa](https://github.com/33cn/chain33/commit/6c6a0aab49ca885ffb60a25ac79c2667ef3f573e))
* compact db ([c704b6a](https://github.com/33cn/chain33/commit/c704b6a3fcb501f546a19a204247cbbc6db69d3f))
* compact db after deleting chunk ([6371fcf](https://github.com/33cn/chain33/commit/6371fcf336241e5430c30a94c303c1dad84c2ac2))
* dht unit test ([1566a58](https://github.com/33cn/chain33/commit/1566a58369fe4c8f4307dc2fcd156840151ddb6d))
* do not push searched peers into peer channel ([1523267](https://github.com/33cn/chain33/commit/152326733cbaba07e0af7d894b2036481db1b58c))
* fetch chunk routine bug ([1a5b5be](https://github.com/33cn/chain33/commit/1a5b5bef855d1966e7944b146a4d2ad456651619))
* fix ci and add manually auto publish release ([28febf7](https://github.com/33cn/chain33/commit/28febf7face3b8842641c72751150a5bd550017a))
* ignore peers without addr when saving peers ([7a15e22](https://github.com/33cn/chain33/commit/7a15e22a5dcf62d426c304dabbdedd6d8ded0264))
* index bug ([fd6a114](https://github.com/33cn/chain33/commit/fd6a114b69aa169f542f1288d532aeb40cd2f4b1))
* libp2p stream leak ([53988d7](https://github.com/33cn/chain33/commit/53988d78a8335ff4ae95a588ea3a970f38acb80c))
* mustFetchChunk context ([c00ce7d](https://github.com/33cn/chain33/commit/c00ce7d738e38b0d7563975f46b9bca404ef558e))
* push to channel unblockedly ([4d884f1](https://github.com/33cn/chain33/commit/4d884f1bf525ffa9296f097763f775cf2a7378e5))
* query chunk records from peers in routing tableif there is no given peer ([5362c0a](https://github.com/33cn/chain33/commit/5362c0a273b4ffc001343f1ab80ba0779b188d07))
* query public ip ([a73dd61](https://github.com/33cn/chain33/commit/a73dd61ebd6ef96d7c67beb7862e10e8ad96b006))
* refresh peer info ([7e95977](https://github.com/33cn/chain33/commit/7e95977ac73d028d776b4556c42b00c559cf5ecb))
* retry to exec block when error occurs in chunk download mode ([330f8b2](https://github.com/33cn/chain33/commit/330f8b2a03a92782314bf18a412d883f039338ed))
* set dht to server mode ([8de2fb9](https://github.com/33cn/chain33/commit/8de2fb98df00900e3d9bc8bbd9b84d35c53cd070))
* unit test ([2269a15](https://github.com/33cn/chain33/commit/2269a1596766ff0bc95e862467a1b657a2bebf93))
* use peerlist instead of best peers ([198f580](https://github.com/33cn/chain33/commit/198f5800ff818b03e19d5d37360655e07f7eacbc))


### Features

* **deps:** bump github.com/decred/base58 from 1.0.2 to 1.0.3 ([cfbde5e](https://github.com/33cn/chain33/commit/cfbde5ef9e4acca23a7e82a47c777f87779969c2))
* **deps:** bump github.com/influxdata/influxdb from 1.7.9 to 1.9.5 ([162a75c](https://github.com/33cn/chain33/commit/162a75c2457d09a7fb1e99a0ae0bd8c29d7a83a0))
* **deps:** bump github.com/multiformats/go-multiaddr ([44b7c10](https://github.com/33cn/chain33/commit/44b7c10f9c6632b188e85ee0b646e06443547c4f))

# [6.4.0](https://github.com/33cn/chain33/compare/v6.3.0...v6.4.0) (2021-10-15)


### Bug Fixes

* 🐛version control: Add github action for auto publish release and tag version ([22642e1](https://github.com/33cn/chain33/commit/22642e187aecaa21d5904c5d82e459fc6a0f72c4))
* **chain:** add ticker stop method ([aac09d4](https://github.com/33cn/chain33/commit/aac09d45e0ee64f77e81cc36da569444da511ccd))
* chunk key when delete ([f559759](https://github.com/33cn/chain33/commit/f5597596f5f20e02c29eb699d3cefa53cd42b95b))
* chunkStatusCacheMutex unlock bug ([cd7bdc8](https://github.com/33cn/chain33/commit/cd7bdc8111538c4ac1926f54ce85f78071624fe5))
* close pubsub ([aca60b8](https://github.com/33cn/chain33/commit/aca60b86d6e5d8b613d10e15f20181e3b445c2eb))
* close stream without delay after reading ([678ff0a](https://github.com/33cn/chain33/commit/678ff0a78e32d5be98cd123717f1f396467dac4b))
* compact block body in localdb ([6c6a0aa](https://github.com/33cn/chain33/commit/6c6a0aab49ca885ffb60a25ac79c2667ef3f573e))
* compact db ([c704b6a](https://github.com/33cn/chain33/commit/c704b6a3fcb501f546a19a204247cbbc6db69d3f))
* compact db after deleting chunk ([6371fcf](https://github.com/33cn/chain33/commit/6371fcf336241e5430c30a94c303c1dad84c2ac2))
* dht unit test ([1566a58](https://github.com/33cn/chain33/commit/1566a58369fe4c8f4307dc2fcd156840151ddb6d))
* do not push searched peers into peer channel ([1523267](https://github.com/33cn/chain33/commit/152326733cbaba07e0af7d894b2036481db1b58c))
* fetch chunk routine bug ([1a5b5be](https://github.com/33cn/chain33/commit/1a5b5bef855d1966e7944b146a4d2ad456651619))
* fix ci and add manually auto publish release ([28febf7](https://github.com/33cn/chain33/commit/28febf7face3b8842641c72751150a5bd550017a))
* ignore peers without addr when saving peers ([7a15e22](https://github.com/33cn/chain33/commit/7a15e22a5dcf62d426c304dabbdedd6d8ded0264))
* index bug ([fd6a114](https://github.com/33cn/chain33/commit/fd6a114b69aa169f542f1288d532aeb40cd2f4b1))
* libp2p stream leak ([53988d7](https://github.com/33cn/chain33/commit/53988d78a8335ff4ae95a588ea3a970f38acb80c))
* mustFetchChunk context ([c00ce7d](https://github.com/33cn/chain33/commit/c00ce7d738e38b0d7563975f46b9bca404ef558e))
* push to channel unblockedly ([4d884f1](https://github.com/33cn/chain33/commit/4d884f1bf525ffa9296f097763f775cf2a7378e5))
* query chunk records from peers in routing tableif there is no given peer ([5362c0a](https://github.com/33cn/chain33/commit/5362c0a273b4ffc001343f1ab80ba0779b188d07))
* query public ip ([a73dd61](https://github.com/33cn/chain33/commit/a73dd61ebd6ef96d7c67beb7862e10e8ad96b006))
* refresh peer info ([7e95977](https://github.com/33cn/chain33/commit/7e95977ac73d028d776b4556c42b00c559cf5ecb))
* retry to exec block when error occurs in chunk download mode ([330f8b2](https://github.com/33cn/chain33/commit/330f8b2a03a92782314bf18a412d883f039338ed))
* return nil when no result for list result ([75bf83a](https://github.com/33cn/chain33/commit/75bf83a484677951f9ef6bf5aee56938340a0f2a))
* set dht to server mode ([8de2fb9](https://github.com/33cn/chain33/commit/8de2fb98df00900e3d9bc8bbd9b84d35c53cd070))
* unit test ([2269a15](https://github.com/33cn/chain33/commit/2269a1596766ff0bc95e862467a1b657a2bebf93))
* use peerlist instead of best peers ([198f580](https://github.com/33cn/chain33/commit/198f5800ff818b03e19d5d37360655e07f7eacbc))
* 在没有新区块时, 完成历史数据的推送 ([cf8ba79](https://github.com/33cn/chain33/commit/cf8ba79582617e796185056cc858de0b84835ccc))


### Features

* **deps:** bump github.com/decred/base58 from 1.0.2 to 1.0.3 ([cfbde5e](https://github.com/33cn/chain33/commit/cfbde5ef9e4acca23a7e82a47c777f87779969c2))
* **deps:** bump github.com/influxdata/influxdb from 1.7.9 to 1.9.5 ([162a75c](https://github.com/33cn/chain33/commit/162a75c2457d09a7fb1e99a0ae0bd8c29d7a83a0))
* **deps:** bump github.com/multiformats/go-multiaddr ([44b7c10](https://github.com/33cn/chain33/commit/44b7c10f9c6632b188e85ee0b646e06443547c4f))

# Changelog
All notable changes to this project will be documented in this file.

## [6.0.2]
### Changed
- changed cli version cmd return json format and added title app localdb version info
- change cli seed generate -l 0 from json format to only string

## [6.0.1]
### Changed
- Update configuration files from p2p/verMix to p2p/verMin from[@libangzhu](https://github.com/libangzhu).
- Update p2p version Logic(if you do not fill in the range of p2p version,then verMin=version,verMax=verMin+1) from[@libangzhu](https://github.com/libangzhu).

# [6.2]
### Changed
- Update dapp coins command line name, 'bty' not supported any more, 'coins' recommended
