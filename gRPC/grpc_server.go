package gRPC

import (
	"context"
	"net"

	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/gserver/gBase"
	"gitlab.vivas.vn/go/internal/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/examples/data"
)

type GRPCServer struct {
	gBase.GServer
	api.UnimplementedAPIServer
}

func NewServer(_addr string, _logger *logger.Logger, _done *chan struct{}, _tls gBase.TLS) *GRPCServer {
	p := &GRPCServer{
		GServer: gBase.GServer{
			Addr:   _addr,
			Logger: _logger,
			Done:   _done,
			Tls:    _tls,
		},
	}
	if _tls.IsTLS {
		p.ServerName = "GRPCS"
	} else {
		p.ServerName = "GRPC"
	}
	return p
}
func (p *GRPCServer) Serve() error {
	p.LogInfo("Start %v server ", p.ServerName)

	if p.Tls.IsTLS {

		lis, err := net.Listen("tcp", p.Addr)
		if err != nil {
			p.LogError("Listen error [%v]", err.Error())
			close(*p.Done)
		} else {
			// Create tls based credential.
			creds, err := credentials.NewServerTLSFromFile(data.Path(p.Tls.Cert), data.Path(p.Tls.Key))
			if err != nil {
				p.LogError("credentials error [%v]", err.Error())
				close(*p.Done)
			} else {
				s := grpc.NewServer(grpc.Creds(creds))
				api.RegisterAPIServer(s, p)
				go s.Serve(lis)
				p.LogInfo("Listener opened on %s", p.Addr)
			}
		}

	} else {
		lis, err := net.Listen("tcp", p.Addr)
		if err != nil {
			p.LogError("Listen error [%v]", err.Error())
			close(*p.Done)
		} else {
			s := grpc.NewServer()
			api.RegisterAPIServer(s, p)
			go s.Serve(lis)

			p.LogInfo("Listener opened on %s", p.Addr)
		}
	}

	return nil
}

func (p GRPCServer) SendRequest(ctx context.Context, request *api.Request) (*api.Reply, error) {
	result := make(chan *gBase.Result)
	var res gBase.Result
	reply := &api.Reply{
		Msg:    "OK",
		Status: 0,
	}
	request.PayloadType = uint32(gBase.ContextType_PROTO)
	request.Protocol = uint32(gBase.RequestFrom_GRPC)
	// send data to handler
	p.HandlerRequest(result, request)
	// wait for return data
	res = *<-result
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
