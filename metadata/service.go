package metadata

import "encoding/json"

type ServiceStatus string

const (
	ServiceStatusUp       ServiceStatus = "up"
	ServiceStatusDown     ServiceStatus = "down"
	ServiceStatusStarting ServiceStatus = "starting"
	ServiceStatusStopping ServiceStatus = "stopping"
)

// ServiceMetadata 服务元数据
type ServiceMetadata struct {
	ServiceName  string            `json:"service_name"`  // 服务名称
	Version      string            `json:"version"`       // 服务版本
	Address      string            `json:"address"`       // 服务地址
	Env          string            `json:"env"`           // 服务环境
	Weight       int               `json:"weight"`        // 服务权重，用于负载均衡
	Tags         []string          `json:"tags"`          // 服务标签
	Status       ServiceStatus     `json:"status"`        // 服务状态：up, down, starting, stopping
	RegisterTime int64             `json:"register_time"` // 注册时间
	InstanceID   string            `json:"instance_id"`   // 服务实例ID
	Properties   map[string]string `json:"properties"`    // 额外属性
}

func (s *ServiceMetadata) Unmarshal(value []byte) error {
	if err := json.Unmarshal(value, s); err != nil {
		return err
	}
	return nil
}

func (s *ServiceMetadata) String() string {
	data, _ := json.Marshal(s)
	return string(data)
}
