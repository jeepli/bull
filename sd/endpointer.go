package sd

import (
	"context"

	"github.com/ikenchina/bull/endpoint"
)

type Endpointer interface {
	// execute entrance
	Exec(ep endpoint.Endpoint, ctx context.Context, req interface{}) (interface{}, error)
	Close() error
	Node() *ServiceNode
}

const (
	EPEventTypeUpdate = 0
	EPEventTypeError  = 1
)

type EndpointerEvent struct {
	Type int
	Data interface{}
}

type Endpointers []Endpointer

type EndpointerFactory func(node *ServiceNode) (Endpointers, error)

type EndpointerEventCallBack func(EndpointerEvent) error

type EndpointerHolder interface {
	Endpointers() ([]Endpointers, error)
	Errors() <-chan error
	RegisterEvent(EndpointerEventCallBack) error
}

type EndpointerHolderConfig struct {
	InvalidateOnError bool
}

func NewEndpointerHolders(src Instancer, factory EndpointerFactory, config EndpointerHolderConfig) (EndpointerHolder, error) {
	errCh := make(chan error, 100)
	se := &DefaultEndpointerHolder{
		cache:     newEndpointCache(factory, config, errCh),
		errCh:     errCh,
		instancer: src,
		ch:        make(chan Event, 1), // @todo change to 0?
	}

	ev, err := src.Start()
	if err != nil {
		return nil, err
	}
	se.cache.Update(ev)

	go se.receive()
	src.Register(se.ch)

	return se, nil
}

type DefaultEndpointerHolder struct {
	cache     *endpointCache
	instancer Instancer
	ch        chan Event
	errCh     chan error
}

func (de *DefaultEndpointerHolder) receive() {
	for event := range de.ch {
		de.cache.Update(event)
	}
}

func (de *DefaultEndpointerHolder) Close() {
	de.instancer.Deregister(de.ch)
	close(de.ch)
}

func (de *DefaultEndpointerHolder) Endpointers() ([]Endpointers, error) {
	return de.cache.Endpoints()
}

func (de *DefaultEndpointerHolder) Errors() <-chan error {
	return de.errCh
}

func (de *DefaultEndpointerHolder) RegisterEvent(ff EndpointerEventCallBack) error {
	de.cache.RegisterEvent(ff)
	return nil
}
