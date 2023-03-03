package gUDP

import (
	"github.com/panjf2000/gnet/v2"
	"gitlab.vivas.vn/go/gserver/gBase"
	"net"
)

import (
	"crypto/rand"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"gitlab.vivas.vn/go/grpc_api/api"
	"google.golang.org/protobuf/types/known/anypb"
	"runtime"
	"sync"
	"time"
)

/* PAYLOAD

 */


type UDPServer struct {
	gBase.GServer
	gnet.BuiltinEventEngine
	lis net.Listener
	tcpDone chan struct{}
	chReceiveMsg chan *gBase.SocketMessage
	clients sync.Map
	mu sync.Mutex
}
func New(config gBase.ConfigOption, chReceiveRequest chan *gBase.Payload) *UDPServer {

	b := gBase.GServer{
		Config:           &config,
		ChReceiveRequest: chReceiveRequest,
	}
	p := &UDPServer{
		GServer: b,
		tcpDone: make(chan struct{}),
		chReceiveMsg: make(chan *gBase.SocketMessage),
	}
	return p
}
func (p *UDPServer) Serve() {
	go gnet.Run(p, "udp://"+p.Config.Addr, gnet.WithMulticore(true), gnet.WithReusePort(true))
	for i := 0 ;i<runtime.NumCPU()*2;i++{
		go p.receiveMessage()
	}
}
func (p *UDPServer) OnBoot(eng gnet.Engine) gnet.Action {
	p.LogInfo("Listener opened on %s", p.Config.Addr)
	return gnet.None
}
func (p *UDPServer) OnShutdown(eng gnet.Engine) {

}
func (p *UDPServer) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	p. LogInfo ("conn [%v] Open", c.Fd())
	// build msg  hello send to client
	newConn := gBase.NewConnection(&gBase.ServerConnection{
		DecType: p.Config.EncodeType,
		IsSetupConnection: true,
	},&gBase.ClientConnection{
		Fd: c.Fd(),
	})
	c.SetContext(newConn)
	p.mu.Lock()
	p.clients.Store(c.Fd(),&c)
	p.mu.Unlock()

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
			Type: uint32(api.TYPE_ID_RECEIVE_HELLO),
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
func (p *UDPServer) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	p. LogInfo ("conn [%v] Close", c.Fd())
	p.mu.Lock()
	p.clients.Delete(c.Fd())
	p.mu.Unlock()
	//if ct := c.Context().(*gBase.Connection);ct != nil{
	//
	//}
	return gnet.Close
}
func (p *UDPServer) OnTraffic(c gnet.Conn) gnet.Action {
	msgs := gBase.DecodePacket(p.Config.Logger,c)
	for i := range msgs{
		p.chReceiveMsg <- msgs[i]
	}
	return gnet.None
}
func (p *UDPServer)receiveMessage(){
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

func (p *UDPServer) onReceiveRequest(msg *gBase.SocketMessage) {
	//decode payload if need
	p.LogInfo("id %v",msg.MSG_ID)
	p.LogInfo("group %v",msg.MsgGroup)
	p.LogInfo("type %v",msg.MsgType)
	if msg.MsgGroup == uint32(api.Group_CONNECTION) {
		if(msg.MsgType == uint32(api.TYPE_ID_REQUEST_HELLO)){

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


			msg.Conn.Client.IsSetupConnection = true
			msg.Conn.Client.PKey = hlRequest.PKey
			msg.Conn.Client.EncType = gBase.Encryption_Type(hlRequest.EncodeType)
			p.LogInfo("Client Setup encode type %s",msg.Conn.Client.EncType.String())
			msg.Conn.Client.Platfrom = int32(hlRequest.Platform)
			msg.Conn.Connection_id,_ = uuid.New().MarshalBinary()


			/// BUILD REPLY

			reply := api.Reply{Status: 0}
			hlreply := api.Hello_Reply{ConnectionId: msg.Conn.Connection_id}
			_hlreply,_ := anypb.New(&hlreply)
			reply.Reply = _hlreply
			_reply , _ := proto.Marshal(&reply)
			_msg := gBase.SocketMessage{
				Payload: _reply,
				MsgType: msg.MsgType,
				MsgGroup: msg.MsgGroup,
				MSG_ID: msg.MSG_ID,

			}
			_buf, _ := _msg.Encode(msg.Conn.Client.EncType, msg.Conn.Client.PKey)
			p.LogInfo("reply to client %d encode type %s ",msg.Conn.Client.Fd,msg.Conn.Client.EncType.String())
			if c,o := p.clients.Load(msg.Fd); o{
				(*c.(*gnet.Conn)).AsyncWrite(_buf,nil)
			}
		}
	}

}

func (p *UDPServer) Close() {
	p.LogInfo("Close")
	close(p.tcpDone)
}
