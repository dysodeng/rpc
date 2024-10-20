package etcd

import (
	"context"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	grpcResolver "google.golang.org/grpc/resolver"
)

var builderEtcdOnceLock sync.Once

// Builder grpc etcd服务发现
// implements grpc resolver.Builder
type Builder struct {
	namespace          string           // 命名空间
	kv                 *clientv3.Client // etcd客户端连接
	dialTimeout        time.Duration    // etcd连接超时时间
	resolveNowFreqTime time.Duration    // 强制 ResolveNow 间隔时长
}

const (
	defaultResolveNowFreq = time.Hour * 2 // 强制 ResolveNow 默认间隔时长2小时
)

// NewEtcdV3Builder new etcd Builder
func NewEtcdV3Builder(etcdAddress string, opts ...BuilderOption) *Builder {
	builder := &Builder{
		namespace:          defaultNamespace,
		resolveNowFreqTime: defaultResolveNowFreq,
		dialTimeout:        defaultTimeout * time.Second,
	}

	for _, opt := range opts {
		opt(builder)
	}

	conf := clientv3.Config{
		Endpoints:   strings.Split(etcdAddress, ","),
		DialTimeout: builder.dialTimeout,
	}

	var err error
	var client *clientv3.Client

	builderEtcdOnceLock.Do(func() {
		client, err = clientv3.New(conf)
		if err == nil {
			builder.kv = client
		}

		timeoutCtx, cancel := context.WithTimeout(context.Background(), conf.DialTimeout)
		defer cancel()
		_, err = client.Status(timeoutCtx, conf.Endpoints[0])
	})

	grpcResolver.Register(builder)

	return builder
}

// Build creates a new resolver for the given target.
func (d *Builder) Build(target grpcResolver.Target, cc grpcResolver.ClientConn, opts grpcResolver.BuildOptions) (grpcResolver.Resolver, error) {
	r := &resolver{
		kv:        d.kv,
		target:    target,
		cc:        cc,
		namespace: d.namespace,
		stopCh:    make(chan struct{}, 1),
		rn:        make(chan struct{}, 1),
		t:         time.NewTicker(d.resolveNowFreqTime),
	}

	go r.watch(context.Background())
	r.ResolveNow(grpcResolver.ResolveNowOptions{})

	return r, nil
}

// Scheme returns the scheme supported by this resolver.
func (d *Builder) Scheme() string {
	return "etcd"
}

// BuilderOption builder option
type BuilderOption func(builder *Builder)

// WithBuilderNamespace 设置命名空间
func WithBuilderNamespace(namespace string) BuilderOption {
	return func(builder *Builder) {
		builder.namespace = namespace
	}
}

// WithBuilderResolveNowTime 设置强制 ResolveNow 间隔时长
func WithBuilderResolveNowTime(t time.Duration) BuilderOption {
	return func(builder *Builder) {
		builder.resolveNowFreqTime = t
	}
}
