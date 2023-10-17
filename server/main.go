package main

import (
	"flag"
	"log"
	"net"

	"github.com/fishioon/comchat/proto"
	"google.golang.org/grpc"
)

func main() {
	address := flag.String("host", "127.0.0.1:9981", "comchat server listen address")
	flag.Parse()
	lis, err := net.Listen("tcp", *address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	opts := []grpc.ServerOption{
	}
	s, err := NewServer()
	if err != nil {
		log.Fatalf("faild to new server: %v", err)
	}
	grpcServer := grpc.NewServer(opts...)
	proto.RegisterChatServer(grpcServer, s)
	grpcServer.Serve(lis)
}
