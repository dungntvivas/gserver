package gTCP

import (
	"github.com/panjf2000/gnet/v2"
	"gitlab.vivas.vn/go/gserver/gBase"
	"google.golang.org/grpc"
	"net"
)

type TCPServer struct {
	gBase.GServer
	lis net.Listener
	s *grpc.Server
	gnet.BuiltinEventEngine
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
func (p *TCPServer) Serve() error {
	return gnet.Run(p, "tcp://"+p.Config.Addr, gnet.WithMulticore(true), gnet.WithTicker(true), gnet.WithReusePort(true))
}
func (p *TCPServer) OnBoot(eng gnet.Engine) gnet.Action {
	p.LogInfo("Listener opened on %s", p.Config.Addr)
	return gnet.None
}
func (p *TCPServer) OnShutdown(eng gnet.Engine) {}
func (p *TCPServer) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	return
}
func (p *TCPServer) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	return gnet.None
}
func (p *TCPServer) OnTraffic(c gnet.Conn) gnet.Action {
	return gnet.None
}
