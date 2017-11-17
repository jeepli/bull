package democlient

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/hashicorp/consul/api"
	pb "github.com/ikenchina/bull/MathService"
	"github.com/ikenchina/bull/sd/consul"
	"google.golang.org/grpc"
)

type server struct{}

func (s *server) Add(ctx context.Context, in *pb.AddRequest) (*pb.AddReply, error) {
	return &pb.AddReply{X: in.A + in.B}, nil
}

func TestServer(address string, port int) {
	listenAdd := fmt.Sprintf("%s:%d", address, port)

	lis, err := net.Listen("tcp", listenAdd)
	if err != nil {
		log.Fatal("failed to listen: %v", err)
	}

	//
	serviceRegister := &api.AgentServiceRegistration{
		ID:      listenAdd,
		Name:    ServiceName,
		Tags:    []string{"prod"},
		Address: address,
		Port:    port,
	}

	cc, _ := api.NewClient(api.DefaultConfig())
	client := consul.NewClient(cc)

	client.Register(serviceRegister)
	defer client.Deregister(serviceRegister)

	s := grpc.NewServer()
	pb.RegisterMathServiceServer(s, &server{})
	s.Serve(lis)
}

const (
	ServiceName = "testgrpc"
)
