package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ikenchina/bull/sd"
	"github.com/ikenchina/bull/sd/consul"
	"github.com/ikenchina/bull/sd/lb"

	"github.com/hashicorp/consul/api"

	pb "testgrpc/MathService"

	"google.golang.org/grpc"

	grpcclient "testsd/grpc/client"
)

type Client struct {
	balancer lb.Balancer
}

func (c *Client) NewNode(node *sd.ServiceNode) (sd.Endpointers, error) {
	s, err := NewRpcClient(node)
	if err != nil {
		return nil, err
	}
	ss := make([]sd.Endpointer, 1)
	ss[0] = s

	return ss, nil
}

func (c *Client) build() error {
	cc, _ := api.NewClient(api.DefaultConfig())
	client := consul.NewClient(cc)
	instancer, err := consul.NewInstancer(client, "testgrpc", []string{"prod"}, true)
	if err != nil {
		fmt.Println(err)
		return err
	}

	instancerErrCh := instancer.Errors()
	go func() {
		for err := range instancerErrCh {
			fmt.Println(err)
		}
	}()
	holder, err := sd.NewEndpointerHolders(instancer, c.NewNode, sd.EndpointerHolderConfig{})
	if err != nil {
		return err
	}

	go func() {
		for err := range holder.Errors() {
			fmt.Println(err)
		}
	}()
	l, err := lb.NewRoundRobin(holder)
	if err != nil {
		return err
	}
	c.balancer = l
	return nil
}

func (c *Client) Add(a, b int32) (int32, error) {
	retry := lb.Retry(4, 4*time.Second, c.balancer, "", func(ctx context.Context, request interface{}) (response interface{}, err error) {
		fmt.Println("Client : ", ctx.Value("client").(*RpcClient).Node().Service.Address)

		xx, err := ctx.Value("client").(*RpcClient).Add(context.Background(), &pb.AddRequest{A: a, B: b}, grpc.FailFast(false))
		return xx.GetX(), err

	})
	ret, err := retry(context.Background(), nil)
	if err != nil {
		return 0, err
	}

	return ret.(int32), err
}

////////// rpc client
type RpcClient struct {
	grpcclient.BaseRpcClient
	pb.MathServiceClient
}

func NewRpcClient(node *sd.ServiceNode) (*RpcClient, error) {
	rr := &RpcClient{}
	err := rr.InitBase(node)
	if err != nil {
		return nil, err
	}
	rr.MathServiceClient = pb.NewMathServiceClient(rr.Conn)
	return rr, nil
}

func (r *RpcClient) Exec(ep sd.Endpoint, ctx context.Context, req interface{}) (interface{}, error) {
	ctx2 := context.WithValue(ctx, "client", r)
	return ep(ctx2, req)
}

//////////
//////////
//////////

func main() {
	cc := &Client{}
	err := cc.build()
	fmt.Println(err)

	for i := 0; i < 1000; i++ {
		ret, err := cc.Add(int32(i), 2)
		fmt.Println(ret, err)
		time.Sleep(2 * time.Second)
	}
}
