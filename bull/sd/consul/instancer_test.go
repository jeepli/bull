package consul

import (
	"fmt"
	"testing"

	consul "github.com/hashicorp/consul/api"

	"github.com/ikenchina/bull/sd"
)

var _ sd.Instancer = (*Instancer)(nil) // API check

var consulState = []*consul.ServiceEntry{
	{
		Node: &consul.Node{
			Address: "127.0.0.1",
			Node:    "app00.local",
		},
		Service: &consul.AgentService{
			ID:      "search-api-0",
			Port:    8000,
			Service: "search",
			Tags: []string{
				"api",
				"v1",
			},
		},
	},
	{
		Node: &consul.Node{
			Address: "127.0.0.1",
			Node:    "app01.local",
		},
		Service: &consul.AgentService{
			ID:      "search-api-1",
			Port:    8001,
			Service: "search",
			Tags: []string{
				"api",
				"v2",
			},
		},
	},
	{
		Node: &consul.Node{
			Address: "127.0.0.1",
			Node:    "app01.local",
		},
		Service: &consul.AgentService{
			Address: "10.0.0.10",
			ID:      "search-db-0",
			Port:    9000,
			Service: "search",
			Tags: []string{
				"abtest",
			},
		},
	},
}

func TestInstancer(t *testing.T) {
	var (
		client = newTestClient(consulState)
	)

	s, err := NewInstancer(client, "search", []string{"api"}, true)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for dd := range s.Errors() {
			t.Fatal(dd)
		}
	}()
	defer s.Stop()

	ev, err := s.Start()
	if err != nil {
		t.Fatal(err)
	}
	for i, node := range ev.Nodes {
		fmt.Println(i, node.Service.ID)
	}

	if want, have := 2, ev.Nodes; len(have) != want {
		t.Errorf("want %d, have %d", want, have)
	}
}

func TestInstancerNoService(t *testing.T) {
	var (
		client = newTestClient(consulState)
	)

	s, err := NewInstancer(client, "search_test", []string{"api"}, true)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for dd := range s.Errors() {
			t.Fatal(dd)
		}
	}()
	defer s.Stop()

	ev, err := s.Start()
	if err != nil {
		t.Fatal(err)
	}
	if want, have := 0, ev.Nodes; len(have) != want {
		t.Errorf("want %d, have %d", want, have)
	}
}

func TestInstancerWithTags(t *testing.T) {
	var (
		client = newTestClient(consulState)
	)

	s, err := NewInstancer(client, "search", []string{"abtest"}, true)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for dd := range s.Errors() {
			t.Fatal(dd)
		}
	}()
	defer s.Stop()

	ev, err := s.Start()
	if err != nil {
		t.Fatal(err)
	}
	if want, have := 1, ev.Nodes; len(have) != want {
		t.Errorf("want %d, have %d", want, have)
	}
}
