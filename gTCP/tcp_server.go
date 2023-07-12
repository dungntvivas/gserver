package gTCP

import (
	"github.com/DungntVccorp/gserver/gBase"
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
