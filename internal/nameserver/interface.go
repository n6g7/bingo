package nameserver

type Record struct {
	Name  string
	Cname string
}

type Nameserver interface {
	Init() error
	ListRecords() ([]Record, error)
	RemoveRecord(name string) error
	AddRecord(name, cname string) error
}
