package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/limscoder/kubetastic/pkg/randopb"

	"google.golang.org/grpc"
)

var serviceAddr = flag.String("server", ":9001", "host:port for server")

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", *serviceAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Printf("listening at %s", *serviceAddr)
	s := grpc.NewServer()
	randopb.RegisterRandoServer(s, &randoServer{})
	s.Serve(lis)
}
