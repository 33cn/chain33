# secp256k1-go

golang secp256k1 library

Implements cryptographic operations for the secp256k1 ECDSA curve used by Bitcoin.

## Installing

You need to have installed [gmp](https://gmplib.org/) in your system.

	sudo apt-get install gmp-dev

### Ubuntu 12.04

	sudo apt-get update
	sudo apt-get install libgmp-dev

### OSX 10.9

	curl -O https://ftp.gnu.org/gnu/gmp/gmp-6.0.0a.tar.xz
	tar xfvz gmp-6.0.0a.tar.xz
	cd gmp-6.0.0a.tar.xz
	./configure
	make
	sudo make install
	make check # just in case

## Test

To run tests do

	go test
