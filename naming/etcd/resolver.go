package etcd

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	grpcResolver "google.golang.org/grpc/resolver"
)

var num uint64

// etcd resolver implements grpc resolver.Resolver
type resolver struct {
	kv        *clientv3.Client
	target    grpcResolver.Target
	cc        grpcResolver.ClientConn
	store     map[string]struct{}
	namespace string
	storeLock sync.Mutex
	stopCh    chan struct{}
	// rn channel is used by ResolveNow() to force an immediate resolution of the target.
	rn chan struct{}
	t  *time.Ticker
}

func (r *resolver) start(ctx context.Context) {
	num++
	log.Printf("resolver start, num: %d", num)
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
					r.storeLock.Lock()
					r.store[string(ev.Kv.Value)] = struct{}{}
					r.storeLock.Unlock()
					r.updateTargetState()
				case mvccpb.DELETE:
					r.storeLock.Lock()
					delete(r.store, strings.Replace(string(ev.Kv.Key), prefix, "", 1))
					r.storeLock.Unlock()
					r.updateTargetState()
				}
			}
		}
	}
}

func (r *resolver) resolveNow() {
	prefix := "/" + r.namespace + "/" + r.target.Endpoint() + "/"
	log.Printf("resolver resolve now")
	log.Printf("prefix: %s", prefix)
	resp, err := r.kv.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err == nil {
		for _, kv := range resp.Kvs {
			r.storeLock.Lock()
			r.store[string(kv.Value)] = struct{}{}
			r.storeLock.Unlock()
		}
	}

	r.updateTargetState()

	log.Printf("%+v", r.store)
	log.Println(r)
}

func (r *resolver) updateTargetState() {
	addresses := make([]grpcResolver.Address, len(r.store))
	i := 0
	for k := range r.store {
		addresses[i] = grpcResolver.Address{Addr: k}
		i++
	}

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
