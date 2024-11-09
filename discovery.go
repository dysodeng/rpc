package rpc

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
)

// ServiceDiscovery 服务发现
type ServiceDiscovery interface {
	// ServiceConn 获取服务连接信息
	ServiceConn(serviceName string, opts ...ServiceDiscoveryOption) (*grpc.ClientConn, error)
}

type serviceDiscovery struct {
	appName         string
	resolverBuilder resolver.Builder
}

func NewServiceDiscovery(appName string, resolverBuilder resolver.Builder) ServiceDiscovery {
	s := &serviceDiscovery{
		appName:         appName,
		resolverBuilder: resolverBuilder,
	}
	return s
}

func (s *serviceDiscovery) ServiceConn(serviceName string, opts ...ServiceDiscoveryOption) (*grpc.ClientConn, error) {
	options := &serviceDiscoveryOption{
		grpcDialOptions: []grpc.DialOption{},
	}
	for _, opt := range opts {
		opt(options)
	}

	if options.credentials == nil {
		options.grpcDialOptions = append(
			options.grpcDialOptions,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	}

	switch options.lb {
	case RoundRobin, PickFirst:
		options.grpcDialOptions = append(
			options.grpcDialOptions,
			grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingConfig":[{"%s":{}}]}`, options.lb)),
		)
		break
	}

	conn, err := grpc.NewClient(
		fmt.Sprintf("%s:///%s.%s", s.resolverBuilder.Scheme(), s.appName, serviceName),
		options.grpcDialOptions...,
	)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// ServiceDiscoveryLB 负载均衡
type ServiceDiscoveryLB string

const (
	PickFirst  ServiceDiscoveryLB = "pick_first"  // 选择第一个
	RoundRobin ServiceDiscoveryLB = "round_robin" // 轮训
)

type serviceDiscoveryOption struct {
	grpcDialOptions []grpc.DialOption
	lb              ServiceDiscoveryLB
	credentials     *credentials.TransportCredentials
}

type ServiceDiscoveryOption func(s *serviceDiscoveryOption)

func WithServiceDiscoveryLB(lb ServiceDiscoveryLB) ServiceDiscoveryOption {
	return func(s *serviceDiscoveryOption) {
		s.lb = lb
	}
}

func WithServiceDiscoveryGrpcDialTransportCredentials(credentials *credentials.TransportCredentials) ServiceDiscoveryOption {
	return func(s *serviceDiscoveryOption) {
		s.credentials = credentials
	}
}

func WithServiceDiscoveryGrpcDialOption(opts ...grpc.DialOption) ServiceDiscoveryOption {
	return func(s *serviceDiscoveryOption) {
		s.grpcDialOptions = append(s.grpcDialOptions, opts...)
	}
}
