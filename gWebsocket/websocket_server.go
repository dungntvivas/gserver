package gWebsocket

import (
	"github.com/DungntVccorp/gserver/gBase"
)

type WSServer struct {
	gBase.SocketServer
}

func New(config gBase.ConfigOption, chReceiveRequest chan *gBase.Payload) *WSServer {
	p := &WSServer{
		gBase.NewSocket(config, chReceiveRequest),
	}
	return p
}
