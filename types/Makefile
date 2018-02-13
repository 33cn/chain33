export PATH:=${PWD}:${PATH}
all:
	@echo ${PATH}
	protoc --go_out=plugins=grpc:. types.proto
