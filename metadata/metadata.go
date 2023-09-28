package metadata

import "sync"

const (
	AppName     = "app_name"     // 应用名称
	ServiceName = "service_name" // 服务名称
	Version     = "version"      // 服务版本
)

const DefaultVersion = "v1.0.0"

// Metadata 服务元数据
type Metadata struct {
	data sync.Map
}

func New(kv map[string]string) *Metadata {
	md := &Metadata{}
	for k, v := range kv {
		md.data.Store(k, v)
	}
	return md
}

func (m *Metadata) Store(k, v string) {
	m.data.Store(k, v)
}

func (m *Metadata) Get(k string) string {
	v, ok := m.data.Load(k)
	if !ok {
		return ""
	}
	return v.(string)
}

func (m *Metadata) GetOrStore(key, defaultValue string) string {
	val, _ := m.data.LoadOrStore(key, defaultValue)
	return val.(string)
}

func (m *Metadata) Metadata() map[string]string {
	data := make(map[string]string)
	m.data.Range(func(k, v interface{}) bool {
		data[k.(string)] = v.(string)
		return true
	})
	return data
}
