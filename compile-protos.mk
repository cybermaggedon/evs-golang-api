
# This is used by maintainer to create generatedgolang code from protobuf

all:
	protoc cyberprobe.proto --go_out=plugins=grpc:$$(pwd)

