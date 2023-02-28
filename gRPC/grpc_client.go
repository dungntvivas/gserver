package gRPC


import (
	"context"
	"fmt"
	"gitlab.vivas.vn/go/grpc_api/api"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
)

func MakeAuthRoundRobinConnection(scheme string, service_name string) (*grpc.ClientConn, error) {
	return grpc.Dial(
		fmt.Sprintf("%s:///%s", scheme, service_name),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(time.Second*5),
	)
}
func CallRpcService(cc *grpc.ClientConn, request *api.Request, reply *api.Reply) error {
	_cc := api.NewAPIClient(cc)
	_reply, err := _cc.SendRequest(context.TODO(), request)
	if err != nil {
		cc.ResetConnectBackoff()
		return err
	}

	_b, err := proto.Marshal(_reply)
	if err != nil {
		return err
	}
	err = proto.Unmarshal(_b, reply)
	if err != nil {
		return err
	}
	return nil
}

/// Phân giải tên miền cho host

type ResolverBuilder struct {
	SchemeName  string
	ServiceName string
	Addrs       []string
}

func (p *ResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &AddrResolver{
		target: target,
		cc:     cc,
		addrsStore: map[string][]string{
			p.ServiceName: p.Addrs,
		},
	}
	r.start()
	return r, nil
}
func (p *ResolverBuilder) Scheme() string { return p.SchemeName }

type AddrResolver struct {
	target     resolver.Target
	cc         resolver.ClientConn
	addrsStore map[string][]string
}

func (r *AddrResolver) start() {
	addrStrs := r.addrsStore[r.target.Endpoint]
	addrs := make([]resolver.Address, len(addrStrs))
	for i, s := range addrStrs {
		addrs[i] = resolver.Address{Addr: s}
	}
	r.cc.UpdateState(resolver.State{Addresses: addrs})
}
func (*AddrResolver) ResolveNow(o resolver.ResolveNowOptions) {}
func (*AddrResolver) Close()                                  {}
