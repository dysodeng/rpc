package naming

import "github.com/dysodeng/rpc/metadata"

// Registry 服务注册
type Registry interface {
	// Register 服务注册
	// serviceName string 服务名称
	Register(serviceName string, meta metadata.ServiceRegisterMetadata) error

	// Unregister 服务注销
	// serviceName string 服务名称
	Unregister(serviceName string) error

	// Close 关闭服务注册
	Close() error
}
