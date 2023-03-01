package gRPC

import (
	"context"
	"net"

	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/gserver/gBase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type GRPCServer struct {
	gBase.GServer
	lis net.Listener
	s *grpc.Server
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
				api.RegisterAPIServer(p.s, p)
				go p.s.Serve(p.lis)
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
			api.RegisterAPIServer(p.s, p)
			go p.s.Serve(p.lis)

			p.LogInfo("Listener opened on %s", p.Config.Addr)
		}
	}

	return nil
}
func (p *GRPCServer) Close(){
	p.LogInfo("Close")
	p.s.Stop()
	p.lis.Close()

}

func (p *GRPCServer) SendRequest(ctx context.Context, request *api.Request) (*api.Reply, error) {
	result := make(chan *gBase.Result)
	var res gBase.Result
	reply := &api.Reply{
		Msg:    "OK",
		Status: 0,
	}
	// send data to handler
	p.HandlerRequest(&gBase.Payload{Request: request, ChResult: result})
	// wait for return data
	res = *<-result
	close(result)
	if res.Status >= 1000 {
		reply.Status = uint32(res.Status)
		reply.Msg = api.ResultType(reply.Status).String()
	} else {
		if res.Reply.Status != 0 {
			reply.Msg = res.Reply.Msg
		}
		reply.Status = res.Reply.Status
		reply.Reply = res.Reply.Reply
	}
	return reply, nil

}
