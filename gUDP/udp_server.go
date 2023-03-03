package gUDP

import (
	"gitlab.vivas.vn/go/gserver/gBase"
)

/* PAYLOAD

 */
type UDPServer struct {
	gBase.SocketServer
}
func New(config gBase.ConfigOption, chReceiveRequest chan *gBase.Payload) *UDPServer {
	p := &UDPServer{
		gBase.NewSocket(config,chReceiveRequest),
	}
	return  p
}