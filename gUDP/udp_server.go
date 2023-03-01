package gUDP

import (
	"github.com/panjf2000/gnet/v2"
	"gitlab.vivas.vn/go/gserver/gBase"
	"google.golang.org/grpc"
	"net"
)

type UDPServer struct {
	gBase.GServer
	lis net.Listener
	s *grpc.Server
	gnet.BuiltinEventEngine
}
func New(config gBase.ConfigOption, chReceiveRequest chan *gBase.Payload) *UDPServer {

	b := gBase.GServer{
		Config:           &config,
		ChReceiveRequest: chReceiveRequest,
	}
	p := &UDPServer{
		GServer: b,
	}
	p.Config.ServerName = "UDP"

	return p
}
