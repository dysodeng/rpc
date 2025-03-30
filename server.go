package rpc

import (
	"fmt"
	"net"
	"reflect"

	"github.com/dysodeng/rpc/config"
	"github.com/dysodeng/rpc/health"
	"github.com/dysodeng/rpc/logger"
	"github.com/dysodeng/rpc/metadata"
	"github.com/dysodeng/rpc/middleware"
	"github.com/dysodeng/rpc/naming"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
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
	config      *config.ServerConfig
	shutdown    chan struct{}
}

func NewServer(conf *config.ServerConfig, registry naming.Registry, opts ...ServerOption) Server {
	options := &serverOption{}
	// 添加默认中间件
	options.grpcServerOptions = append(options.grpcServerOptions,
		grpc.UnaryInterceptor(middleware.Chain(
			middleware.Recovery(),
			middleware.Logging(logger.Logger()),
			middleware.Metrics(),
		)),
	)
	for _, opt := range opts {
		opt(options)
	}

	s := &server{
		appName:     conf.AppName,
		serviceAddr: conf.ServiceAddr,
		registry:    registry,
		config:      conf,
		grpcServer:  grpc.NewServer(options.grpcServerOptions...),
		shutdown:    make(chan struct{}),
	}

	// 注册健康检查服务
	grpc_health_v1.RegisterHealthServer(s.grpcServer, &health.Server{})

	return s
}

func (s *server) RegisterService(service metadata.ServiceRegister, grpcRegister interface{}) error {
	// grpc 服务注册
	fn := reflect.ValueOf(grpcRegister)
	if fn.Kind() != reflect.Func {
		return errors.Errorf("grpcRegister must be a function, got %T", grpcRegister)
	}

	params := make([]reflect.Value, 2)
	params[0] = reflect.ValueOf(s.grpcServer)
	params[1] = reflect.ValueOf(service)
	fn.Call(params)

	serviceMetadata := service.RegisterMetadata()

	// 向注册中心注册服务
	serviceName := fmt.Sprintf("%s.%s", s.appName, serviceMetadata.ServiceName)
	err := s.registry.Register(serviceName, serviceMetadata)
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
	s.grpcServer.GracefulStop()
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
