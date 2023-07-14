//go:build !windows
// +build !windows

//
package gUDP

import (
	"github.com/DungntVccorp/gserver/gBase"
)

/*
PAYLOAD
*/
type UDPServer struct {
	gBase.SocketServer
}

func New(config gBase.ConfigOption, chReceiveRequest chan *gBase.Payload) *UDPServer {
	p := &UDPServer{
		gBase.NewSocket(config, chReceiveRequest),
	}
	return p
}
