package client

import (
	"fmt"

	"google.golang.org/grpc"

	pb "yunion.io/x/yunion-kube/pkg/agent/localvolume"
)

type Client struct {
	pb.LocalVolumeClient
}

func NewClient(unixFile string) (*Client, error) {
	conn, err := grpc.Dial(fmt.Sprintf("unix://%s", unixFile), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	c := &Client{
		LocalVolumeClient: pb.NewLocalVolumeClient(conn),
	}
	return c, nil
}
