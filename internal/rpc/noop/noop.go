package noop

import (
	"context"
	"syscall"

	"github.com/containers/buildah/internal/rpc/noop/pb"
	"google.golang.org/grpc"
)

type noopServer struct {
	pb.UnimplementedNoopServer
}

func (n *noopServer) Noop(_ context.Context, req *pb.NoopRequest) (*pb.NoopResponse, error) {
	if req == nil {
		return nil, syscall.EINVAL
	}
	resp := &pb.NoopResponse{}
	resp.Ignored = req.Ignored
	return resp, nil
}

func Register(s grpc.ServiceRegistrar) {
	pb.RegisterNoopServer(s, &noopServer{})
}
