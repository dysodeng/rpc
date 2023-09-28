package contracts

import (
	"github.com/dysodeng/rpc/metadata"
)

// Registry 服务注册
type Registry interface {
	// Register 服务注册
	// serviceName string 服务名称
	// metadata metadata.Metadata 服务元数据
	Register(serviceName string, metadata *metadata.Metadata) error

	// UpdateService 更新服务信息
	// serviceName string 服务名称
	// metadata metadata.Metadata 服务元数据
	UpdateService(serviceName string, metadata *metadata.Metadata) error

	// Unregister 服务注销
	// serviceName string 服务名称
	Unregister(serviceName string) error
}
