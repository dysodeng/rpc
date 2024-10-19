package etcd

import (
	"context"
	"sync"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	grpcResolver "google.golang.org/grpc/resolver"
)

// etcd resolver implements grpc resolver.Resolver
type resolver struct {
	kv        *clientv3.Client
	target    grpcResolver.Target
	cc        grpcResolver.ClientConn
	store     sync.Map
	namespace string
	stopCh    chan struct{}
	rn        chan struct{} // rn channel is used by ResolveNow() to force an immediate resolution of the target.
	t         *time.Ticker
}

func (r *resolver) start(ctx context.Context) {
	prefix := "/" + r.namespace + "/" + r.target.Endpoint() + "/"
	rch := r.kv.Watch(ctx, prefix, clientv3.WithPrefix())
	for {
		select {
		case <-r.rn:
			r.resolveNow()
		case <-r.t.C:
			r.ResolveNow(grpcResolver.ResolveNowOptions{})
		case <-r.stopCh:
			return
		case wresp := <-rch:
			for _, ev := range wresp.Events {
				switch ev.Type {
				case mvccpb.PUT:
					r.store.Store(string(ev.Kv.Value), struct{}{})
					r.updateTargetState()
				case mvccpb.DELETE:
					r.store.Delete(string(ev.Kv.Key))
					r.updateTargetState()
				}
			}
		}
	}
}

func (r *resolver) resolveNow() {
	prefix := "/" + r.namespace + "/" + r.target.Endpoint() + "/"
	resp, err := r.kv.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err == nil {
		for _, kv := range resp.Kvs {
			r.store.Store(string(kv.Value), struct{}{})
		}
	}
	r.updateTargetState()
}

func (r *resolver) updateTargetState() {
	var addresses []grpcResolver.Address
	r.store.Range(func(key, value any) bool {
		addresses = append(addresses, grpcResolver.Address{Addr: key.(string)})
		return true
	})
	_ = r.cc.UpdateState(grpcResolver.State{Addresses: addresses})
}

// ResolveNow will be called by gRPC to try to resolve the target name
// again. It's just a hint, resolver can ignore this if it's not necessary.
func (r *resolver) ResolveNow(opt grpcResolver.ResolveNowOptions) {
	select {
	case r.rn <- struct{}{}:
	default:
	}
}

// Close closes the resolver.
func (r *resolver) Close() {
	r.t.Stop()
	close(r.stopCh)
}
