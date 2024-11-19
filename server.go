package rpc

import (
	"fmt"
	"net"
	"reflect"

	"github.com/dysodeng/rpc/metadata"
	"github.com/dysodeng/rpc/naming"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// Server grpc服务
type Server interface {
	RegisterService(serviceMetadata metadata.ServiceRegister, grpcRegister interface{}) error
	Serve() error
	Stop() error
}

type server struct {
	appName     string
	serviceAddr string
	registry    naming.Registry
	grpcServer  *grpc.Server
}

func NewServer(appName, serviceAddr string, registry naming.Registry, opts ...ServerOption) Server {
	options := &serverOption{}
	for _, opt := range opts {
		opt(options)
	}
	s := &server{
		appName:     appName,
		serviceAddr: serviceAddr,
		registry:    registry,
		grpcServer:  grpc.NewServer(options.grpcServerOptions...),
	}
	return s
}

func (s *server) RegisterService(service metadata.ServiceRegister, grpcRegister interface{}) error {
	// grpc 服务注册
	fn := reflect.ValueOf(grpcRegister)
	if fn.Kind() != reflect.Func {
		return errors.New("`grpcRegister` is not a valid grpc registration function")
	}
	params := make([]reflect.Value, 2)
	params[0] = reflect.ValueOf(s.grpcServer)
	params[1] = reflect.ValueOf(service)
	fn.Call(params)

	serviceMetadata := service.RegisterMetadata()

	// 向注册中心注册服务
	serviceName := fmt.Sprintf("%s.%s", s.appName, serviceMetadata.ServiceName)
	err := s.registry.Register(serviceName)
	if err != nil {
		return err
	}

	return nil
}

func (s *server) Serve() error {
	listen, err := net.Listen("tcp", s.serviceAddr)
	if err != nil {
		return err
	}

	err = s.grpcServer.Serve(listen)
	if err != nil {
		return err
	}

	return nil
}

func (s *server) Stop() error {
	err := s.registry.Close()
	if err != nil {
		return err
	}
	s.grpcServer.Stop()
	return nil
}

type serverOption struct {
	grpcServerOptions []grpc.ServerOption
}

type ServerOption func(s *serverOption)

func WithServerGrpcServerOption(opts ...grpc.ServerOption) ServerOption {
	return func(s *serverOption) {
		s.grpcServerOptions = append(s.grpcServerOptions, opts...)
	}
}
