package config

type ServerConfig struct {
	AppName               string
	ServiceAddr           string
	OtelCollectorEndpoint string // OTEL collector 地址
	EtcdConfig            EtcdConfig
}

type EtcdConfig struct {
	Endpoints   []string
	DialTimeout int
	Namespace   string
}
