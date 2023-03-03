package gBase

import (
	"crypto/rand"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
	"gitlab.vivas.vn/go/grpc_api/api"
	"google.golang.org/protobuf/types/known/anypb"
	"net"
	"runtime"
	"sync"
	"time"
)

type SocketServer struct {
	GServer
	gnet.BuiltinEventEngine
	lis net.Listener
	Done chan struct{}
	chReceiveMsg chan *SocketMessage
	clients sync.Map
	mu sync.Mutex
}
func NewSocket(config ConfigOption, chReceiveRequest chan *Payload) SocketServer {

	b := GServer{
		Config:           &config,
		ChReceiveRequest: chReceiveRequest,
	}
	p := SocketServer{
		GServer: b,
		Done: make(chan struct{}),
		chReceiveMsg: make(chan *SocketMessage),
	}
	return p
}
func (p *SocketServer) Serve() {
	protocol := "tcp"
	if p.Config.Protocol == RequestProtocol_TCP{
		protocol = "tcp://"
	}else if p.Config.Protocol == RequestProtocol_UDP{
		protocol = "udp://"
	}else if p.Config.Protocol == RequestProtocol_UDS{
		protocol = "unix://"
	}
	go gnet.Run(p, protocol+p.Config.Addr, gnet.WithMulticore(true), gnet.WithReusePort(true))
	for i := 0 ;i<runtime.NumCPU()*2;i++{
		go p.receiveMessage()
	}
}
func (p *SocketServer) OnBoot(eng gnet.Engine) gnet.Action {
	p.LogInfo("Listener opened on %s", p.Config.Addr)
	return gnet.None
}
func (p *SocketServer) OnShutdown(eng gnet.Engine) {

}
func (p *SocketServer) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	p. LogInfo ("conn [%v] Open", c.Fd())
	// build msg  hello send to client
	newConn := NewConnection(&ServerConnection{
		DecType: p.Config.EncodeType,
		IsSetupConnection: true,
	},&ClientConnection{
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
		msg := NewMessage(_receive_bin,0, receive.Type, bytes)
		out, _ := msg.Encode(Encryption_NONE, nil)
		(*_c).AsyncWrite(out,nil)
	}(&c)
	return
}
func (p *SocketServer) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	p. LogInfo ("conn [%v] Close", c.Fd())
	p.mu.Lock()
	p.clients.Delete(c.Fd())
	p.mu.Unlock()
	return gnet.Close
}
func (p *SocketServer) OnTraffic(c gnet.Conn) gnet.Action {
	msgs := DecodePacket(p.Config.Logger,c)
	for i := range msgs{
		p.chReceiveMsg <- msgs[i]
	}
	return gnet.None
}
func (p *SocketServer)receiveMessage(){
loop:
	for{
		select {
		case <- p.Done:
			break loop
		case msg := <-p.chReceiveMsg:
			p.LogInfo("Receive Msg from connection %d",msg.Fd)
			p.onReceiveRequest(msg)
		}
	}
}

func (p *SocketServer) onReceiveRequest(msg *SocketMessage) {
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
			msg.Conn.Client.EncType = Encryption_Type(hlRequest.EncodeType)
			p.LogInfo("Client Setup encode type %s",msg.Conn.Client.EncType.String())
			msg.Conn.Client.Platfrom = int32(hlRequest.Platform)
			msg.Conn.Connection_id,_ = uuid.New().MarshalBinary()


			/// BUILD REPLY

			reply := api.Reply{Status: 0}
			hlreply := api.Hello_Reply{ConnectionId: msg.Conn.Connection_id}
			_hlreply,_ := anypb.New(&hlreply)
			reply.Reply = _hlreply
			_reply , _ := proto.Marshal(&reply)
			_msg := SocketMessage{
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

func (p *SocketServer) Close() {
	p.LogInfo("Close")
	close(p.Done)
}

