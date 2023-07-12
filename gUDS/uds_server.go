package gUDS

import (
	"github.com/DungntVccorp/gserver/gBase"
)

/* PAYLOAD

 */

type UDSServer struct {
	gBase.SocketServer
}

func New(config gBase.ConfigOption, chReceiveRequest chan *gBase.Payload) *UDSServer {
	p := &UDSServer{
		gBase.NewSocket(config, chReceiveRequest),
	}
	return p
}
