package rpc

import (
	"fmt"
	"time"

	"github.com/dysodeng/rpc/breaker"
	"github.com/dysodeng/rpc/retry"
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
	resolverBuilder resolver.Builder
}

func NewServiceDiscovery(resolverBuilder resolver.Builder) ServiceDiscovery {
	s := &serviceDiscovery{
		resolverBuilder: resolverBuilder,
	}
	return s
}

func (s *serviceDiscovery) ServiceConn(serviceName string, opts ...ServiceDiscoveryOption) (*grpc.ClientConn, error) {
	options := &serviceDiscoveryOption{
		grpcDialOptions: []grpc.DialOption{},
		lb:              RoundRobin,
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

	// 拦截器
	var chain []grpc.UnaryClientInterceptor
	if options.withBreaker { // 熔断器
		chain = append(chain, breaker.Interceptor(options.cb))
	}
	if options.withRetry { // 重试
		chain = append(chain, retry.Interceptor(options.retryPolicy))
	}
	options.grpcDialOptions = append(
		options.grpcDialOptions,
		grpc.WithChainUnaryInterceptor(chain...),
	)

	switch options.lb {
	case RoundRobin, PickFirst:
		options.grpcDialOptions = append(
			options.grpcDialOptions,
			grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingConfig":[{"%s":{}}]}`, options.lb)),
		)
	}

	conn, err := grpc.NewClient(
		fmt.Sprintf("%s:///%s", s.resolverBuilder.Scheme(), serviceName),
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
	withBreaker     bool
	cb              breaker.CircuitBreaker
	withRetry       bool
	retryPolicy     *retry.Policy
}

type ServiceDiscoveryOption func(s *serviceDiscoveryOption)

// WithServiceDiscoveryLB 添加负载均衡选项
func WithServiceDiscoveryLB(lb ServiceDiscoveryLB) ServiceDiscoveryOption {
	return func(s *serviceDiscoveryOption) {
		s.lb = lb
	}
}

// WithServiceDiscoveryKeepalive 添加保活与超时选项
func WithServiceDiscoveryKeepalive(keepaliveTime, timeout time.Duration, backoffConf backoff.Config) ServiceDiscoveryOption {
	return func(s *serviceDiscoveryOption) {
		s.grpcDialOptions = append(
			s.grpcDialOptions,
			grpc.WithConnectParams(grpc.ConnectParams{
				Backoff:           backoffConf,
				MinConnectTimeout: timeout,
			}),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:                keepaliveTime, // 保活时间
				Timeout:             timeout,       // 保活超时
				PermitWithoutStream: true,          // 没有活动流时也保持连接
			}),
		)
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

// WithServiceDiscoveryBreaker 设置熔断器
func WithServiceDiscoveryBreaker(cb breaker.CircuitBreaker) ServiceDiscoveryOption {
	return func(s *serviceDiscoveryOption) {
		s.withBreaker = true
		s.cb = cb
	}
}

// WithServiceDiscoveryRetry 设置重试
func WithServiceDiscoveryRetry(retryPolicy *retry.Policy) ServiceDiscoveryOption {
	return func(s *serviceDiscoveryOption) {
		s.withRetry = true
		s.retryPolicy = retryPolicy
	}
}
