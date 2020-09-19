.PHONY:build

build: service.pb.go

service.pb.go: service.proto
	protoc --go_out=plugins=grpc:. $^
