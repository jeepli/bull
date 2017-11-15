package lb

import (
	"sync"
	"sync/atomic"

	"github.com/ikenchina/bull/sd"
)

// NewRoundRobin returns a load balancer that returns services in sequence.
func NewRoundRobin(s sd.EndpointerHolder) (Balancer, error) {
	rr := &roundRobin{
		s: s,
		c: 0,
	}

	endpointers, err := rr.s.Endpointers()
	if err != nil {
		return nil, err
	}
	rr.eps = endpointers

	err = s.RegisterEvent(rr.update)
	if err != nil {
		return nil, err
	}
	return rr, nil
}

type roundRobin struct {
	s sd.EndpointerHolder
	c uint64
	i []uint64
	sync.RWMutex
	eps []sd.Endpointers
}

func (rr *roundRobin) update(ff sd.EndpointerEvent) error {
	rr.Lock()
	defer rr.Unlock()
	endpointers, err := rr.s.Endpointers()
	if err != nil {
		return err
	}
	rr.eps = endpointers

	return nil
}

func (rr *roundRobin) Endpointer(id string, last sd.Endpointer) (sd.Endpointer, error) {
	rr.RLock()

	if len(rr.eps) <= 0 {
		rr.RUnlock()
		return nil, ErrNoEndpointers
	}
	rr.RUnlock()

	old := atomic.AddUint64(&rr.c, 1) - 1
	idx := old % uint64(len(rr.eps))

	rr.Lock()
	if len(rr.eps) != len(rr.i) {
		rr.i = make([]uint64, len(rr.eps))
	}

	rr.Unlock()

	eps := rr.eps[idx]

	old2 := atomic.AddUint64(&rr.i[idx], 1)
	idx2 := old2 % uint64(len(eps))
	return eps[idx2], nil
}
