package gRPC

import (
	"context"
	"fmt"
	"time"

	"gitlab.vivas.vn/go/grpc_api/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/resolver/manual"
)

var serviceConfig = `{
	"loadBalancingPolicy": "round_robin",
	"healthCheckConfig": {
		"serviceName": ""
	}
}`

func NewClientConn(scheme string, service_name string, addrs ...string) (api.APIClient, error) {
	r := manual.NewBuilderWithScheme(scheme)
	var rAddress []resolver.Address
	for _, ad := range addrs {
		rAddress = append(rAddress, resolver.Address{
			Addr: ad,
		})
	}
	r.InitialState(resolver.State{Addresses: rAddress})
	address := fmt.Sprintf("%s:///%s", r.Scheme(), service_name)
	fmt.Printf("%s\n", address)

	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithResolvers(r),
		grpc.WithDefaultServiceConfig(serviceConfig),
	}

	conn, err := grpc.Dial(address, options...)

	if err != nil {
		return nil, err
	}

	cc := api.NewAPIClient(conn)
	fmt.Printf("New API Client")

	return cc, nil
}

func MakeRpcRequest(cc api.APIClient, request *api.Request) (*api.Reply, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := cc.SendRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	return r, nil
}
