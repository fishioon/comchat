package main

import (
	"context"
	"flag"
	"log"
	"sync"

	"github.com/fishioon/comchat/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile             = flag.String("ca_file", "", "The file containing the CA root cert file")
	serverAddr         = flag.String("addr", "localhost:9981", "The server address in the format of host:port")
	serverHostOverride = flag.String("server_host_override", "x.test.example.com", "The server name used to verify the hostname returned by the TLS handshake")
)

func main() {
	flag.Parse()
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := proto.NewChatClient(conn)

	groups := []*proto.Group{
		{Id: "baidu.com", Seq: ""},
	}
	stream, err := client.Conn(context.TODO(), &proto.ConnReq{Groups: groups})
	if err != nil {
		log.Fatalf("fail to conn: %v", err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
			stream.CloseSend()
		}()
		for {
			msg, err := stream.Recv()
			if err != nil {
				log.Printf("recv msg err: %v", err)
				return
			}
			log.Printf("recv msg success: %v", msg)
		}
	}()
	client.PubMsg(context.TODO(), &proto.PubMsgReq{
		Msg: &proto.Msg{
			Id:      "hello",
			Gid:     "baidu.com",
			Content: "hello, comchat",
		},
	})
	wg.Wait()
}
