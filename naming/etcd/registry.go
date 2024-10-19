package etcd

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/dysodeng/rpc/naming"
	"github.com/pkg/errors"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// etcd 实现注册中心
type etcd struct {
	serviceAddress string
	namespace      string
	lease          int64
	services       sync.Map
	kv             *clientv3.Client
	stopChan       chan struct{}
}

var initOnceLock sync.Once

const (
	defaultLease     = 5      // 默认服务租约时长(秒)
	defaultTimeout   = 5      // 默认etcd连接超时时长(秒)
	defaultNamespace = "grpc" // 默认服务命名空间
)

func NewEtcdRegistry(serviceListenAddress, etcdAddress string) (naming.Registry, error) {
	etcdRegistry := &etcd{
		serviceAddress: serviceListenAddress,
		namespace:      defaultNamespace,
		lease:          defaultLease,
	}

	var err error
	var cli *clientv3.Client

	conf := clientv3.Config{
		Endpoints:   strings.Split(etcdAddress, ","),
		DialTimeout: defaultTimeout * time.Second,
	}

	initOnceLock.Do(func() {
		cli, err = clientv3.New(conf)
		if err == nil {
			etcdRegistry.kv = cli
		}

		timeoutCtx, cancel := context.WithTimeout(context.Background(), defaultTimeout*time.Second)
		defer cancel()
		_, err = cli.Status(timeoutCtx, conf.Endpoints[0])
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not connect to etcd")
	}

	// etcd健康检查与服务端断连重连机制
	go func() {
		checkHealth := func() error {
			ctx, chCancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer chCancel()

			_, err = etcdRegistry.kv.Maintenance.Status(ctx, conf.Endpoints[0])
			if err != nil {
				return err
			}
			return nil
		}

		// 每5秒检查一次etcd服务状态
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err = checkHealth(); err != nil {
					log.Println("etcd server health check failed.")
					// 尝试重连etcd
					cli, err = clientv3.New(conf)
					if err == nil {
						log.Println("reconnect to etcd successfully")
						// 重新注册服务
						etcdRegistry.kv = cli
						etcdRegistry.services.Range(func(key, value any) bool {
							_ = etcdRegistry.Register(key.(string))
							return true
						})
					}
				}
			}
		}
	}()

	return etcdRegistry, nil
}

func (registry *etcd) Register(serviceName string) error {
	serviceKey := fmt.Sprintf("/%s/%s/%s", registry.namespace, serviceName, registry.serviceAddress)

	// 设置租约时间
	ctx, cancel := context.WithCancel(context.Background())
	resp, err := registry.kv.Grant(ctx, registry.lease)
	if err != nil {
		cancel()
		return err
	}

	// 注册服务并绑定租约
	_, err = registry.kv.Put(ctx, serviceKey, registry.serviceAddress, clientv3.WithLease(resp.ID))
	if err != nil {
		cancel()
		return err
	}

	// 设置续租 并定期发送续租请求(心跳)
	leaseRespChan, err := registry.kv.KeepAlive(ctx, resp.ID)
	if err != nil {
		cancel()
		return err
	}

	go func() {
		for {
			select {
			case _, ok := <-leaseRespChan:
				if !ok {
					cancel()
					return
				}
			}
		}
	}()

	// 本地存储服务
	registry.services.Store(serviceName, resp.ID)

	log.Printf("register gRPC service: %s", serviceName)

	return nil
}

func (registry *etcd) Unregister(serviceName string) error {
	leaseID, ok := registry.services.Load(serviceName)
	if ok {
		// 撤销租约
		if _, err := registry.kv.Revoke(context.Background(), leaseID.(clientv3.LeaseID)); err != nil {
			return err
		}
	}

	log.Printf("unregister gRPC service: %s", serviceName)

	return nil
}

func (registry *etcd) Close() error {
	// 注销所有服务
	registry.services.Range(func(key, value any) bool {
		_ = registry.Unregister(key.(string))
		return true
	})
	return registry.kv.Close()
}

func (registry *etcd) serviceList() map[string]interface{} {
	list := make(map[string]interface{})
	registry.services.Range(func(key, value any) bool {
		list[key.(string)] = value.(clientv3.LeaseID)
		return true
	})
	return list
}
