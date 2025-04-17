package config

type ServerConfig struct {
	ServiceAddr           string
	OtelCollectorEndpoint string // OTEL collector 地址
	EtcdConfig            EtcdConfig
}

type EtcdConfig struct {
	Endpoints   []string
	DialTimeout int
	Namespace   string
}
