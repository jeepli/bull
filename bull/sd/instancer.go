package sd

import (
	"runtime"
)

type Node struct {
	ID              string
	Node            string
	Address         string
	TaggedAddresses map[string]string
	Meta            map[string]string
}
type Service struct {
	ID                string
	Service           string
	Tags              []string
	Port              int
	Address           string
	EnableTagOverride bool
}
type HealthCheck struct {
	Node        string
	CheckID     string
	Name        string
	Status      string
	Notes       string
	Output      string
	ServiceID   string
	ServiceName string
}

// HealthChecks is a collection of HealthCheck structs.
type HealthChecks []*HealthCheck

type ServiceMetric struct {
	CPU struct {
		Num int
	}
	Memory *runtime.MemStats
}

type ServiceNode struct {
	Node    *Node
	Service *Service
	Checks  HealthChecks
	Metric  *ServiceMetric
}
type Event struct {
	Nodes []*ServiceNode
	Err   error
}
type Instancer interface {
	Register(chan<- Event)
	Deregister(chan<- Event)
	Start() (Event, error)
	Stop()
}
