package main

import (
	"context"
	"math/rand"

	"google.golang.org/grpc/codes"

	"github.com/limscoder/kubetastic/pkg/randopb"
	"google.golang.org/grpc"
)

type randoServer struct {
	throttleRatio int
}

func (s *randoServer) GetRand(ctx context.Context, req *randopb.GetRandRequest) (*randopb.GetRandResponse, error) {
	r := rand.New(rand.NewSource(req.Seed))
	throttle := r.Intn(100)
	if throttle <= s.throttleRatio {
		return nil, grpc.Errorf(codes.ResourceExhausted, "request throttled")
	}

	val := r.Intn(int(req.Max))
	return &randopb.GetRandResponse{Value: int32(val)}, nil
}
