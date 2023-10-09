package gRPC

import (
	"context"
	"net"

	"github.com/DungntVccorp/grpc_api/api"
	"github.com/DungntVccorp/gserver/gBase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

type GRPCServer struct {
	gBase.GServer
	lis net.Listener
	s   *grpc.Server
	api.UnimplementedAPIServer
}

func New(config gBase.ConfigOption, chReceiveRequest chan *gBase.Payload) *GRPCServer {
	b := gBase.GServer{
		Config:           &config,
		ChReceiveRequest: chReceiveRequest,
	}
	p := &GRPCServer{
		GServer: b,
	}
	if p.Config.Tls.IsTLS {
		p.Config.ServerName = "GRPCS"
	} else {
		p.Config.ServerName = "GRPC"
	}
	return p
}
func (p *GRPCServer) Serve() error {
	p.LogInfo("Start %v server ", p.Config.ServerName)
	healthcheck := health.NewServer()
	if p.Config.Tls.IsTLS {
		var err error
		p.lis, err = net.Listen("tcp", p.Config.Addr)
		if err != nil {
			p.LogError("Listen error [%v]", err.Error())
			close(*p.Config.Done)
		} else {
			// Create tls based credential.
			creds, err := credentials.NewServerTLSFromFile(p.Config.Tls.Cert, p.Config.Tls.Key)
			if err != nil {
				p.LogError("credentials error [%v]", err.Error())
				close(*p.Config.Done)
			} else {
				p.s = grpc.NewServer(grpc.Creds(creds))
				healthgrpc.RegisterHealthServer(p.s, healthcheck)

				api.RegisterAPIServer(p.s, p)
				go p.s.Serve(p.lis)
				next := healthgrpc.HealthCheckResponse_SERVING
				healthcheck.SetServingStatus("", next)
				p.LogInfo("Listener opened on %s", p.Config.Addr)
			}
		}

	} else {
		var err error
		p.lis, err = net.Listen("tcp", p.Config.Addr)
		if err != nil {
			p.LogError("Listen error [%v]", err.Error())
			close(*p.Config.Done)
		} else {
			p.s = grpc.NewServer()

			healthgrpc.RegisterHealthServer(p.s, healthcheck)
			api.RegisterAPIServer(p.s, p)
			go p.s.Serve(p.lis)
			next := healthgrpc.HealthCheckResponse_SERVING
			healthcheck.SetServingStatus("", next)

			p.LogInfo("Listener opened on %s", p.Config.Addr)
		}
	}
	return nil
}

func (p *GRPCServer) Close() {
	p.LogInfo("Close")
	p.s.Stop()
	p.lis.Close()

}

func (p *GRPCServer) SendRequest(ctx context.Context, request *api.Request) (*api.Reply, error) {
	chReply := make(chan *api.Reply)
	//request.PayloadType = uint32(gBase.PayloadType_BIN)
	request.Protocol = uint32(gBase.RequestProtocol_GRPC)
	// send data to handler
	p.HandlerRequest(&gBase.Payload{Request: request, ChReply: chReply})
	// wait for return data
	res := <-chReply
	close(chReply)
	return res, nil

}
