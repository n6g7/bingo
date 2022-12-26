package registry

type Service struct {
	Name   string
	Domain string
}

type Registry interface {
	Init() error
	ListFabioServices() ([]Service, error)
}
