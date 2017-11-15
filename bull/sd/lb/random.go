package lb

import (
	"math/rand"
	"sync"

	"github.com/ikenchina/bull/sd"
)

// NewRandom returns a load balancer that selects services randomly.
func NewRandom(s sd.EndpointerHolder, seed int64) (Balancer, error) {
	rr := &random{
		s: s,
		r: rand.New(rand.NewSource(seed)),
	}
	err := s.RegisterEvent(rr.update)
	if err != nil {
		return nil, err
	}
	return rr, nil
}

type random struct {
	sync.RWMutex
	s   sd.EndpointerHolder
	r   *rand.Rand
	eps []sd.Endpointers
}

func (rr *random) update(sd.EndpointerEvent) error {
	rr.Lock()
	defer rr.Unlock()
	endpointers, err := rr.s.Endpointers()
	if err != nil {
		return err
	}
	rr.eps = endpointers
	return nil
}

func (r *random) Endpointer(id string, last sd.Endpointer) (sd.Endpointer, error) {
	r.RLock()
	defer r.RUnlock()

	if len(r.eps) <= 0 {
		return nil, ErrNoEndpointers
	}
	eps := r.eps[r.r.Intn(len(r.eps))]
	return eps[r.r.Intn(len(eps))], nil
}
