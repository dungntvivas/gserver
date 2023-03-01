package gWebsocket

import (
	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/gserver/gBase"
	"google.golang.org/grpc"
	"net"
)

type WSServer struct {
	gBase.GServer
	lis net.Listener
	s *grpc.Server
	api.UnimplementedAPIServer
}
func New(config gBase.ConfigOption, chReceiveRequest chan *gBase.Payload) *WSServer {

	b := gBase.GServer{
		Config:           &config,
		ChReceiveRequest: chReceiveRequest,
	}
	p := &WSServer{
		GServer: b,
	}
	if p.Config.Tls.IsTLS {
		p.Config.ServerName = "WSS"
	} else {
		p.Config.ServerName = "WS"
	}


	return p
}
