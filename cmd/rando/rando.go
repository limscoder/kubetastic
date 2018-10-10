package main

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/limscoder/kubetastic/pkg/randopb"
)

type randoServer struct{}

func (s *randoServer) GetRand(ctx context.Context, req *randopb.GetRandRequest) (*randopb.GetRandResponse, error) {
	r := rand.New(rand.NewSource(int64(req.Seed)))
	val := r.Intn(int(req.Max))
	fmt.Printf("got a value: %v", val)
	fmt.Println(randopb.GetRandResponse{Value: int32(val)})
	return &randopb.GetRandResponse{Value: int32(val)}, nil
}
