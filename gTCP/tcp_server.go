package gTCP

import (
	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/gserver/gBase"
	"google.golang.org/grpc"
	"net"
)

type TCPServer struct {
	gBase.GServer
	lis net.Listener
	s *grpc.Server
	api.UnimplementedAPIServer
}
func New(config gBase.ConfigOption, chReceiveRequest chan *gBase.Payload) *TCPServer {

	b := gBase.GServer{
		Config:           &config,
		ChReceiveRequest: chReceiveRequest,
	}
	p := &TCPServer{
		GServer: b,
	}
	p.Config.ServerName = "TCP"


	return p
}