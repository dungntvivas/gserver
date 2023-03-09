package gTCP

import (
	"gitlab.vivas.vn/go/gserver/gBase"
)

/* PAYLOAD

 */

type TCPServer struct {
	gBase.SocketServer
}

func New(config gBase.ConfigOption, chReceiveRequest chan *gBase.Payload) *TCPServer {
	p := &TCPServer{
		gBase.NewSocket(config, chReceiveRequest),
	}
	return p
}
