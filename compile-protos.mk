
all:
	protoc cyberprobe.proto --go_out=plugins=grpc:$$(pwd)

