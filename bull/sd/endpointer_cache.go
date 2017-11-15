package sd

import (
	"errors"
	"sync"
)

var (
	ErrInvalidEvent = errors.New("invalid event")
)

// endpointCache collects the most recent set of instances from a service discovery
// system, creates endpoints for them using a factory function, and makes
// them available to consumers.
type endpointCache struct {
	config    EndpointerHolderConfig
	mtx       sync.RWMutex
	factory   EndpointerFactory
	cache     map[string]Endpointers
	err       error
	endpoints []Endpointers
	errCh     chan error
	eventFunc EndpointerEventCallBack
}

func newEndpointCache(factory EndpointerFactory, config EndpointerHolderConfig, errCh chan error) *endpointCache {
	return &endpointCache{
		config:  config,
		factory: factory,
		cache:   map[string]Endpointers{},
		errCh:   errCh,
	}
}

// Update should be invoked by clients with a complete set of current instance
// strings whenever that set changes. The cache manufactures new endpoints via
// the factory, closes old endpoints when they disappear, and persists existing
// endpoints if they survive through an update.
func (c *endpointCache) Update(event Event) {

	if event.Err == nil {
		c.updateCache(event)
		c.err = nil
		return
	}

	// @todo cleanup
	// Sad path. Something's gone wrong in sd.
	c.errCh <- event.Err
	if !c.config.InvalidateOnError {
		return // keep returning the last known endpoints on error
	}
	if c.err != nil {
		return // already in the error state, do nothing & keep original error
	}
	c.err = event.Err
	// set new deadline to invalidate Endpoints unless non-error Event is received
	//c.invalidateDeadline = c.timeNow().Add(c.config.invalidateTimeout)
	if c.eventFunc != nil {
		err := c.eventFunc(EndpointerEvent{Type: EPEventTypeError, Data: event.Err})
		if err != nil {
			c.errCh <- err
		}
	}

	return
}

// @todo check status of health check
// @notify load balancer when any changing of services
func (c *endpointCache) updateCache(event Event) {
	update := false

	// Deterministic order (for later).
	instances := make([]string, 0)
	for _, node := range event.Nodes {
		if node.Service == nil {
			c.errCh <- ErrInvalidEvent
		} else {
			instances = append(instances, node.Service.ID)
		}
	}

	// Produce the current set of services.
	cache := make(map[string]Endpointers, len(instances))
	for _, node := range event.Nodes {
		if node.Service == nil {
			continue
		}
		// If it already exists, just copy it over.
		if sc, ok := c.cache[node.Service.ID]; ok {
			cache[node.Service.ID] = sc
			delete(c.cache, node.Service.ID)
			continue
		}

		service, err := c.factory(node)
		if err != nil {
			c.errCh <- err
			continue
		}
		update = true
		cache[node.Service.ID] = service
	}

	// Close any leftover endpoints.
	for _, sc := range c.cache {
		for _, ss := range sc {
			err := ss.Close()
			update = true
			if err != nil {
				c.errCh <- err
			}
		}
	}

	endpoints := make([]Endpointers, 0, len(cache))
	for _, instance := range instances {
		if _, ok := cache[instance]; !ok {
			continue
		}
		endpoints = append(endpoints, cache[instance])
	}

	// Swap and trigger GC for old copies.
	c.mtx.Lock()
	c.endpoints = endpoints
	c.cache = cache
	c.mtx.Unlock()

	if update && c.eventFunc != nil {
		err := c.eventFunc(EndpointerEvent{Type: EPEventTypeUpdate})
		if err != nil {
			c.errCh <- err
		}
	}
}

// Endpoints yields the current set of (presumably identical) endpoints, ordered
// lexicographically by the corresponding instance string.
func (c *endpointCache) Endpoints() ([]Endpointers, error) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	return c.endpoints, nil
}

func (c *endpointCache) RegisterEvent(ff EndpointerEventCallBack) error {
	c.eventFunc = ff
	return nil
}
