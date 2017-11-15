package client

import (
	"google.golang.org/grpc"

	"github.com/ikenchina/bull/sd"
)

type BaseRpcClient struct {
	node *sd.ServiceNode
	Conn *grpc.ClientConn
}

func (r *BaseRpcClient) InitBase(node *sd.ServiceNode) error {
	conn, err := grpc.Dial(node.Service.Address, grpc.WithInsecure())
	if err != nil {
		return err
	}

	r.Conn = conn
	r.node = node

	return nil
}

func (r *BaseRpcClient) Close() error {
	r.Conn.Close()
	return nil
}

func (r *BaseRpcClient) Node() *sd.ServiceNode {
	return r.node
}
