package rpc

import (
	"context"
	"fmt"
	"time"

	"github.com/dysodeng/rpc/breaker"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
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
		lb:              RoundRobin,
		timeout:         time.Second * 10,
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

	// 超时设置与重试
	options.grpcDialOptions = append(
		options.grpcDialOptions,
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff:           backoff.DefaultConfig,
			MinConnectTimeout: options.timeout,
		}),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second, // 保活时间
			Timeout:             options.timeout,  // 保活超时
			PermitWithoutStream: true,             // 没有活动流时也保持连接
		}),
	)

	switch options.lb {
	case RoundRobin, PickFirst:
		options.grpcDialOptions = append(
			options.grpcDialOptions,
			grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingConfig":[{"%s":{}}]}`, options.lb)),
		)
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
	RoundRobin ServiceDiscoveryLB = "round_robin" // 轮询
)

type serviceDiscoveryOption struct {
	grpcDialOptions []grpc.DialOption
	lb              ServiceDiscoveryLB
	credentials     *credentials.TransportCredentials
	timeout         time.Duration
}

type ServiceDiscoveryOption func(s *serviceDiscoveryOption)

func WithServiceDiscoveryLB(lb ServiceDiscoveryLB) ServiceDiscoveryOption {
	return func(s *serviceDiscoveryOption) {
		s.lb = lb
	}
}

// WithServiceDiscoveryTimeout 添加超时设置选项
func WithServiceDiscoveryTimeout(timeout time.Duration) ServiceDiscoveryOption {
	return func(s *serviceDiscoveryOption) {
		s.timeout = timeout
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

// WithBreaker 设置熔断器
func WithBreaker(cb breaker.CircuitBreaker) ServiceDiscoveryOption {
	return func(s *serviceDiscoveryOption) {
		s.grpcDialOptions = append(s.grpcDialOptions, grpc.WithUnaryInterceptor(
			func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
				return cb.Execute(
					ctx,
					func() error {
						// 执行正常的 RPC 调用
						return invoker(ctx, method, req, reply, cc, opts...)
					},
					func(err error) error {
						// 服务降级处理
						// 这里可以返回默认值、缓存数据或者错误信息
						return err
					},
				)
			}),
		)
	}
}
