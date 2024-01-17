package proxy

type Service struct {
	Name   string
	Domain string
}

type Proxy interface {
	Init() error
	ListServices() ([]Service, error)
	GetTarget(sourceDomain string) string
	IsValidTarget(target string) bool
}
