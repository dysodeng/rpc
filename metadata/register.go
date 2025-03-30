package metadata

type ServiceRegister interface {
	RegisterMetadata() ServiceRegisterMetadata
	UnsafeServiceRegister
}

type ServiceRegisterMetadata struct {
	AppName     string
	ServiceName string
	Version     string
	Env         string
	Tags        []string
}

type UnsafeServiceRegister interface {
	mustEmbedUnimplementedServiceRegister()
}

type UnimplementedServiceRegister struct{}

func (UnimplementedServiceRegister) RegisterMetadata() ServiceRegisterMetadata {
	panic("Implement the RegisterMetadata method for the ServiceRegister interface")
}

func (UnimplementedServiceRegister) mustEmbedUnimplementedServiceRegister() {}
