package etcd

import (
	"context"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	grpcResolver "google.golang.org/grpc/resolver"
)

// Builder grpc etcd服务发现
// implements grpc resolver.Builder
type Builder struct {
	// etcd客户端连接
	kv *clientv3.Client
	// 命名空间
	namespace string
	// 强制 ResolveNow 间隔时长
	resolveNowFreqTime time.Duration
}

const (
	defaultResolveNowFreq = time.Hour * 2 // 强制 ResolveNow 默认间隔时长(小时)
)

// NewEtcdV3Builder new etcd Builder
func NewEtcdV3Builder(etcdClient *clientv3.Client) *Builder {
	builder := &Builder{
		kv:                 etcdClient,
		namespace:          defaultNamespace,
		resolveNowFreqTime: defaultResolveNowFreq,
	}
	grpcResolver.Register(builder)
	return builder
}

// Build creates a new resolver for the given target.
func (d *Builder) Build(target grpcResolver.Target, cc grpcResolver.ClientConn, opts grpcResolver.BuildOptions) (grpcResolver.Resolver, error) {
	r := &resolver{
		kv:        d.kv,
		target:    target,
		cc:        cc,
		store:     make(map[string]struct{}),
		namespace: d.namespace,
		stopCh:    make(chan struct{}, 1),
		rn:        make(chan struct{}, 1),
		t:         time.NewTicker(d.resolveNowFreqTime),
	}

	go r.start(context.Background())
	r.ResolveNow(grpcResolver.ResolveNowOptions{})
	return r, nil
}

// Scheme returns the scheme supported by this resolver.
func (d *Builder) Scheme() string {
	return "etcd"
}
