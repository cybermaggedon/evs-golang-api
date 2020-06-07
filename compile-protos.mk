
all:
	protoc protos/cyberprobe.proto --go_out=plugins=grpc:$$(pwd)

