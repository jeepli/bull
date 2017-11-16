package lb

import (
	"errors"
	"fmt"
	"sync"

	"github.com/stathat/consistent"

	"github.com/ikenchina/bull/sd"
)

var (
	ErrInvalidEndpointer = errors.New("invalid endpointer")
)

// NewconsistentHash returns a load balancer that returns services in sequence.
func NewConsistentHash(s sd.EndpointerHolder) (Balancer, error) {
	rr := &consistentHash{
		s:    s,
		hash: consistent.New(),
		eps:  make(map[string]sd.Endpointer),
	}

	endpointers, err := rr.s.Endpointers()
	if err != nil {
		return nil, err
	}
	rr.updateEndpointers(endpointers)

	err = s.RegisterEvent(rr.update)
	if err != nil {
		return nil, err
	}
	return rr, nil
}

type consistentHash struct {
	hash *consistent.Consistent
	s    sd.EndpointerHolder
	sync.RWMutex
	eps map[string]sd.Endpointer
}

func (rr *consistentHash) update(ff sd.EndpointerEvent) error {
	rr.Lock()
	defer rr.Unlock()
	endpointers, err := rr.s.Endpointers()
	if err != nil {
		return err
	}

	rr.updateEndpointers(endpointers)

	return nil
}

func (rr *consistentHash) updateEndpointers(endpointers []sd.Endpointers) error {
	for _, ep := range endpointers {
		for i, e := range ep {
			if e.Node() == nil || e.Node().Service == nil {
				return ErrInvalidEndpointer
			}
			key := fmt.Sprintf("%d+%s:%d", i, e.Node().Service.Address, e.Node().Service.Port)
			rr.hash.Add(key)
			rr.eps[key] = e
		}
	}
	return nil
}

func (rr *consistentHash) Endpointer(id string, last sd.Endpointer) (sd.Endpointer, error) {
	rr.RLock()
	defer rr.RUnlock()
	if len(rr.eps) <= 0 {
		return nil, ErrNoEndpointers
	}

	key, err := rr.hash.Get(id)
	if err != nil {
		return nil, err
	}

	ep := rr.eps[key]

	return ep, nil
}
