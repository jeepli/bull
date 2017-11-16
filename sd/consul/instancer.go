package consul

import (
	"io"
	"unsafe"

	consul "github.com/hashicorp/consul/api"

	"github.com/ikenchina/bull/sd"
)

const defaultIndex = 0

// Instancer yields instances for a service in Consul.
type Instancer struct {
	client      Client
	errChan     chan error
	service     string
	tags        []string
	passingOnly bool
	quitc       chan struct{}
	notify      map[chan<- sd.Event]struct{}
}

func NewInstancer(client Client, service string, tags []string, passingOnly bool) (*Instancer, error) {
	errChan := make(chan error, 100)
	s := &Instancer{
		client:      client,
		service:     service,
		errChan:     errChan,
		tags:        tags,
		passingOnly: passingOnly,
		quitc:       make(chan struct{}),
		notify:      make(map[chan<- sd.Event]struct{}, 0),
	}

	return s, nil
}

// @todo set registrator channel to 0 or start return event
func (s *Instancer) Start() (sd.Event, error) {
	event, index, err := s.getEvent(defaultIndex, nil)
	if err != nil {
		return sd.Event{}, err
	}

	go s.loop(index)

	return event, nil
}

// Stop terminates the instancer.
func (s *Instancer) Stop() {
	close(s.quitc)
}

func (s *Instancer) Errors() <-chan error {
	return s.errChan
}

func (s *Instancer) loop(lastIndex uint64) {
	var (
		event sd.Event
		err   error
	)
	for {
		event, lastIndex, err = s.getEvent(lastIndex, s.quitc)
		switch {
		case err == io.EOF:
			return
		case err != nil:
			s.errChan <- err
			s.notifyEvent(sd.Event{
				Err: err,
			})
		default:
			s.notifyEvent(event)
		}
	}
}

func (s *Instancer) notifyEvent(event sd.Event) {
	for ch := range s.notify {
		ch <- event
	}
}

func (s *Instancer) getEvent(lastIndex uint64, interruptc chan struct{}) (sd.Event, uint64, error) {
	tag := ""
	if len(s.tags) > 0 {
		tag = s.tags[0]
	}

	// Consul doesn't support more than one tag in its service query method.
	// https://github.com/hashicorp/consul/issues/294
	// Hashi suggest prepared queries, but they don't support blocking.
	// https://www.consul.io/docs/agent/http/query.html#execute
	// If we want blocking for efficiency, we must filter tags manually.

	type response struct {
		event sd.Event
		index uint64
	}

	var (
		errc = make(chan error, 1)
		resc = make(chan response, 1)
	)

	go func() {
		entries, meta, err := s.client.Service(s.service, tag, s.passingOnly, &consul.QueryOptions{
			WaitIndex: lastIndex,
		})
		if err != nil {
			errc <- err
			return
		}
		if len(s.tags) > 1 {
			entries = filterEntries(entries, s.tags[1:]...)
		}

		resc <- response{
			event: makeEvent(entries),
			index: meta.LastIndex,
		}
	}()

	select {
	case err := <-errc:
		return sd.Event{}, 0, err
	case res := <-resc:
		return res.event, res.index, nil
	case <-interruptc:
		return sd.Event{}, 0, io.EOF
	}
}

// Register implements Instancer.
func (s *Instancer) Register(ch chan<- sd.Event) {
	s.notify[ch] = struct{}{}
}

// Deregister implements Instancer.
func (s *Instancer) Deregister(ch chan<- sd.Event) {
	delete(s.notify, ch)
}

func filterEntries(entries []*consul.ServiceEntry, tags ...string) []*consul.ServiceEntry {
	var es []*consul.ServiceEntry

ENTRIES:
	for _, entry := range entries {
		ts := make(map[string]struct{}, len(entry.Service.Tags))
		for _, tag := range entry.Service.Tags {
			ts[tag] = struct{}{}
		}

		for _, tag := range tags {
			if _, ok := ts[tag]; !ok {
				continue ENTRIES
			}
		}
		es = append(es, entry)
	}

	return es
}

func makeEvent(entries []*consul.ServiceEntry) sd.Event {
	event := sd.Event{}
	for _, entry := range entries {
		event.Nodes = append(event.Nodes, &sd.ServiceNode{
			Node:    makeNode(entry.Node),
			Service: makeService(entry.Service),
			Checks:  makeHealth(entry.Checks),
			// @todo missing metrics
		})
	}
	return event
}

func makeNode(node *consul.Node) *sd.Node {
	return (*sd.Node)(unsafe.Pointer(node))
}

func makeService(sv *consul.AgentService) *sd.Service {
	return (*sd.Service)(unsafe.Pointer(sv))
}
func makeHealth(hc consul.HealthChecks) sd.HealthChecks {
	ret := make(sd.HealthChecks, len(hc))
	for i := 0; i < len(ret); i++ {
		ret[i] = (*sd.HealthCheck)(unsafe.Pointer(hc[i]))
	}
	return ret
}
