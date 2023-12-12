package grpc

import (
	"context"
	"net"

	auth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"google.golang.org/grpc/reflection"

	recovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"

	grpcValidator "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/validator"

	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"
)

// https://grpc.io
// https://github.com/grpc-ecosystem

// RPCServer is RPC server config.
type RPCServer struct {
	server  *grpc.Server
	addr    string
	network string
}

// New grpc server
func New(opt ...*Options) *RPCServer {
	option := DefaultOptions()

	option.merge(opt)

	keepParams := option.Parameters.getGrpcKeepaliveParams()

	unaryServerInterceptors := make([]grpc.UnaryServerInterceptor, 0)

	unaryServerInterceptors = append(
		unaryServerInterceptors,
		recovery.UnaryServerInterceptor(),
		grpcValidator.UnaryServerInterceptor(), // 参数校验
	)
	if option.Auth != nil {
		unaryServerInterceptors = append(unaryServerInterceptors, auth.UnaryServerInterceptor(option.Auth.Options))
	}

	if len(option.CustomInterceptors) > 0 {
		unaryServerInterceptors = append(unaryServerInterceptors, option.CustomInterceptors...)
	}

	srv := grpc.NewServer(
		keepParams,
		grpc.ChainUnaryInterceptor(unaryServerInterceptors...),
		grpc.ChainStreamInterceptor(
			recovery.StreamServerInterceptor(),
		),
	)

	return &RPCServer{
		server:  srv,
		addr:    option.Grpc.Addr,
		network: option.Grpc.Network,
	}
}

func (s *RPCServer) GetGrpcServer() *grpc.Server {
	return s.server
}

func (s *RPCServer) Serve(ctx context.Context) {
	lis, err := net.Listen(s.network, s.addr)
	if err != nil {
		panic(err)
	}
	reflection.Register(s.server)
	if err = s.server.Serve(lis); err != nil {
		panic(err)
	}
}

func (s *RPCServer) Close(ctx context.Context) {
	s.server.GracefulStop()
}
