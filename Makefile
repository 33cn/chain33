all:
	go build -i
	./chain33
clean:
	rm -rf datadir/addrbook
	rm -rf datadir/blockchain.db
	rm -rf datadir/mavltree
