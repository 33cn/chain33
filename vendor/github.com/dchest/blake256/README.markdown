Package blake256
=====================

	import "github.com/dchest/blake256"

Package blake256 implements BLAKE-256 and BLAKE-224 hash functions (SHA-3
candidate).

Public domain.


Constants
---------

``` go
const BlockSize = 64
```
The block size of the hash algorithm in bytes.

``` go
const Size = 32
```
The size of BLAKE-256 hash in bytes.

``` go
const Size224 = 28
```
The size of BLAKE-224 hash in bytes.


Functions
---------

### func New

	func New() hash.Hash

New returns a new hash.Hash computing the BLAKE-256 checksum.

### func New224

	func New224() hash.Hash

New224 returns a new hash.Hash computing the BLAKE-224 checksum.

### func New224Salt

	func New224Salt(salt []byte) hash.Hash

New224Salt is like New224 but initializes salt with the given 16-byte slice.

### func NewSalt

	func NewSalt(salt []byte) hash.Hash

NewSalt is like New but initializes salt with the given 16-byte slice.
