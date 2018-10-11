package main

import (
	"flag"
	"log"
	"net"
	"strconv"

	"github.com/limscoder/kubetastic/pkg/randopb"

	"google.golang.org/grpc"
)

var serviceAddr = flag.String("server", ":9001", "host:port for server")
var throttle = flag.String("throttle", "0", "number of incoming requests to throttle (out of 100)")

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", *serviceAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	throttleRatio, err := strconv.Atoi(*throttle)
	if err != nil {
		log.Fatalf("failed to parse throttle flag: %v", err)
	}

	s := grpc.NewServer()
	randopb.RegisterRandoServer(s, &randoServer{throttleRatio: throttleRatio})
	s.Serve(lis)
}
