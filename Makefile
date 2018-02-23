all:
	go build -i
	./chain33
ticket:
	./chain33 -f chain33.test.toml
clean:
	rm -rf datadir/addrbook
	rm -rf datadir/blockchain.db
	rm -rf datadir/mavltree
	rm -rf chain33.log
