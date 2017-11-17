package democlient

import (
	"context"
	"fmt"
	"time"

	"github.com/ikenchina/bull/sd"
	"github.com/ikenchina/bull/sd/consul"
	"github.com/ikenchina/bull/sd/lb"

	"github.com/hashicorp/consul/api"

	pb "github.com/ikenchina/bull/MathService"

	"google.golang.org/grpc"

	grpcclient "github.com/ikenchina/bull/grpc/client"
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

func (c *Client) Build() error {
	cc, _ := api.NewClient(api.DefaultConfig())
	client := consul.NewClient(cc)
	instancer, err := consul.NewInstancer(client, ServiceName, []string{"prod"}, true)
	if err != nil {
		fmt.Println(err)
		return err
	}

	instancerErrCh := instancer.Errors()
	go func() {
		for err := range instancerErrCh {
			fmt.Println("instanceErr : ", err)
		}
	}()
	holder, err := sd.NewEndpointerHolders(instancer, c.NewNode, sd.EndpointerHolderConfig{})
	if err != nil {
		return err
	}

	go func() {
		for err := range holder.Errors() {
			fmt.Println("holderErr : ", err)
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
		fmt.Println(ctx.Value("client").(*RpcClient).Node().Service.Address, ctx.Value("client").(*RpcClient).Node().Service.Port)
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
