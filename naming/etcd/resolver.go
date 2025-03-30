package etcd

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/dysodeng/rpc/metadata"
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

func (r *resolver) watch(ctx context.Context) {
	prefix := fmt.Sprintf("/%s/%s/", r.namespace, r.target.Endpoint())
	rch := r.kv.Watch(ctx, prefix, clientv3.WithPrefix())
	for {
		select {
		case <-r.rn:
			r.resolveNow()
		case <-r.t.C:
			r.ResolveNow(grpcResolver.ResolveNowOptions{})
		case <-r.stopCh:
			return
		case keyWatchCh := <-rch:
			for _, event := range keyWatchCh.Events {
				switch event.Type {
				case mvccpb.PUT:
					var serviceMetadata metadata.ServiceMetadata
					_ = serviceMetadata.Unmarshal(event.Kv.Value)
					if err := serviceMetadata.Unmarshal(event.Kv.Value); err == nil {
						log.Printf("put service: %s, address: %s", serviceMetadata.ServiceName, serviceMetadata.Address)
						r.store.Store(serviceMetadata.Address, serviceMetadata)
						r.updateTargetState()
					}
				case mvccpb.DELETE:
					r.store.Range(func(key, value any) bool {
						if meta, ok := value.(metadata.ServiceMetadata); ok {
							if strings.HasSuffix(string(event.Kv.Key), meta.InstanceID) {
								r.store.Delete(key)
								log.Printf("delete service: %s, address: %s", meta.ServiceName, meta.Address)
								return false
							}
						}
						return true
					})
					r.updateTargetState()
				}
			}
		}
	}
}

func (r *resolver) resolveNow() {
	prefix := fmt.Sprintf("/%s/%s/", r.namespace, r.target.Endpoint())
	resp, err := r.kv.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err == nil {
		for _, kv := range resp.Kvs {
			var serviceMetadata metadata.ServiceMetadata
			err = serviceMetadata.Unmarshal(kv.Value)
			if err != nil {
				log.Printf("resolveNow error: %s", err.Error())
				continue
			}
			if serviceMetadata.Status != metadata.ServiceStatusUp {
				switch serviceMetadata.Status {
				case metadata.ServiceStatusDown:
					log.Printf("resolveNow service %s is down", serviceMetadata.ServiceName)
				case metadata.ServiceStatusStopping:
					log.Printf("resolveNow service %s is stopping", serviceMetadata.ServiceName)
				case metadata.ServiceStatusStarting:
					log.Printf("resolveNow service %s is starting", serviceMetadata.ServiceName)
				}
				r.store.Delete(serviceMetadata.Address)
				continue
			}
			log.Printf("resolveNow serviceMetadata: %s", serviceMetadata.String())
			r.store.Store(serviceMetadata.Address, serviceMetadata)
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
