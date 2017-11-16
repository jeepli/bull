package lb

import (
	"errors"

	"github.com/ikenchina/bull/sd"
)

// Balancer yields endpoints according to some heuristic.
type Balancer interface {
	Endpointer(id string, last sd.Endpointer) (sd.Endpointer, error)
}

// ErrNoEndpoints is returned when no qualifying endpoints are available.
var ErrNoEndpointers = errors.New("no endpointers available")
