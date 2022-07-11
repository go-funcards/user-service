package main

import user "github.com/go-funcards/user-service/cmd"

//go:generate protoc -I proto --go_out=./proto/v1 --go-grpc_out=./proto/v1 proto/v1/user.proto

func main() {
	user.Execute()
}
