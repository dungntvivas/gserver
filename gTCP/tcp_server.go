package gTCP

import (
	"crypto/rand"
	"github.com/golang/protobuf/proto"
	"github.com/panjf2000/gnet/v2"
	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/gserver/gBase"
	"google.golang.org/protobuf/types/known/anypb"
	"net"
	"runtime"
	"time"
)

/* PAYLOAD

*/


type TCPServer struct {
	gBase.GServer
	gnet.BuiltinEventEngine
	lis net.Listener
	tcpDone chan struct{}
	chReceiveMsg chan *gBase.SocketMessage
}
func New(config gBase.ConfigOption, chReceiveRequest chan *gBase.Payload) *TCPServer {

	b := gBase.GServer{
		Config:           &config,
		ChReceiveRequest: chReceiveRequest,
	}
	p := &TCPServer{
		GServer: b,
		tcpDone: make(chan struct{}),
		chReceiveMsg: make(chan *gBase.SocketMessage),
	}
	p.Config.ServerName = "TCP"

	return p
}
func (p *TCPServer) Serve() {
	go gnet.Run(p, "tcp://"+p.Config.Addr, gnet.WithMulticore(true), gnet.WithTicker(true), gnet.WithReusePort(true))
	for i := 0 ;i<runtime.NumCPU()*2;i++{
		go p.receiveMessage()
	}
}
func (p *TCPServer) OnBoot(eng gnet.Engine) gnet.Action {
	p.LogInfo("Listener opened on %s", p.Config.Addr)
	return gnet.None
}
func (p *TCPServer) OnShutdown(eng gnet.Engine) {

}
func (p *TCPServer) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	p. LogInfo ("conn [%v] Open", c.Fd())
	// build msg  hello send to client
	newConn := gBase.NewConnection(&gBase.ServerConnection{
		DecType: p.Config.EncodeType,
		IsSetupConnection: true,
	})
	c.SetContext(newConn)

	go func(_c *gnet.Conn) {

		/// build hello
		bytes := make([]byte, 5) //generate a random 32 byte key for AES-256
		rand.Read(bytes)
		helloReceive := api.HelloReceive{
			ServerTime:       uint64(time.Now().Unix()),
			PKey:             newConn.Server.PKey,
			ServerEncodeType: api.EncodeType(newConn.Server.DecType),
		}
		receive := api.Receive{
			Type: 100,
			ServerTime: helloReceive.ServerTime,
		}
		_receiveAny,_ := anypb.New(&helloReceive)
		receive.Receive = _receiveAny
		_receive_bin,_ := proto.Marshal(&receive)
		msg := gBase.NewMessage(_receive_bin,0, receive.Type, bytes)
		out, _ := msg.Encode(gBase.Encryption_NONE, nil)
		(*_c).AsyncWrite(out,nil)
	}(&c)
	return
}
func (p *TCPServer) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	p. LogInfo ("conn [%v] Close", c.Fd())
	//if ct := c.Context().(*gBase.Connection);ct != nil{
	//
	//}
	return gnet.Close
}
func (p *TCPServer) OnTraffic(c gnet.Conn) gnet.Action {
	msgs := gBase.DecodePacket(p.Config.Logger,c)
	for i := range msgs{
		p.chReceiveMsg <- msgs[i]
	}
	return gnet.None
}
func (p *TCPServer)receiveMessage(){
	loop:
		for{
			select {
				case <- p.tcpDone:
					break loop
			case msg := <-p.chReceiveMsg:
				p.LogInfo("Receive Msg from connection %d",msg.Fd)
				p.onReceiveRequest(msg)
			}
		}
}

func (p *TCPServer) onReceiveRequest(msg *gBase.SocketMessage) {
	//decode payload if need
	p.LogInfo("id %v",msg.MSG_ID)
	p.LogInfo("group %v",msg.MsgGroup)
	p.LogInfo("type %v",msg.MsgType)
	if msg.MsgGroup == uint32(api.Group_CONNECTION) {
		if(msg.MsgType == 101){

			request := api.Request{}
			msg.ToProtoModel(&request)
			p.LogInfo("payload %v",msg.Payload)

			hlRequest := api.Hello_Request{}
			if err := request.Request.UnmarshalTo(&hlRequest);err != nil{
				p.LogError("Request UnmarshalTo Hello_Request %v",err.Error())
				return
			}
			p.LogInfo("Client Encode Type %v",hlRequest.EncodeType.String())
			p.LogInfo("Platfrom %v",hlRequest.Platform)
		}
	}

}

func (p *TCPServer) Close() {
	p.LogInfo("Close")
	close(p.tcpDone)
}

