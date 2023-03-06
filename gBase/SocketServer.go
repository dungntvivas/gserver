package gBase

import (
	"bytes"
	"crypto/rand"
	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
	"gitlab.vivas.vn/go/grpc_api/api"
	"google.golang.org/protobuf/proto"
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
			p.onReceiveRequest(msg)
		}
	}
}

func (p *SocketServer) onReceiveRequest(msg *SocketMessage) {

	if msg.MsgGroup == uint32(api.Group_CONNECTION) {
		if(msg.MsgType == uint32(api.TYPE_ID_REQUEST_HELLO)){
			hlRequest := api.Hello_Request{}
			status := uint32(api.ResultType_OK)
			if err := msg.ToRequestProtoModel(&hlRequest);err != nil {
				p.LogError("Request UnmarshalTo Hello_Request %v",err.Error())
				status = uint32(api.ResultType_REQUEST_INVALID)
			}else{
				// process
				msg.Conn.Client.IsSetupConnection = true
				msg.Conn.Client.PKey = hlRequest.PKey
				msg.Conn.Client.EncType = Encryption_Type(hlRequest.EncodeType)
				p.LogInfo("Client %d Setup encode type %s",msg.Conn.Client.Fd,msg.Conn.Client.EncType.String())
				msg.Conn.Client.Platfrom = int32(hlRequest.Platform)
				msg.Conn.Connection_id,_ = uuid.New().MarshalBinary()
			}
			/// BUILD REPLY
			hlreply := api.Hello_Reply{ConnectionId: msg.Conn.Connection_id}
			_buf ,err := GetReplyBuffer(status,msg.MsgType,msg.MsgGroup,msg.MSG_ID,&hlreply,msg.Conn.Client.EncType,msg.Conn.Client.PKey)
			if err != nil {
				p.LogError("Error %v",err.Error())
			}
			if c,o := p.clients.Load(msg.Fd); o{
				(*c.(*gnet.Conn)).AsyncWrite(_buf,nil)
			}
		}else if msg.MsgType == uint32(api.TYPE_ID_REQUEST_KEEPALIVE) {
			p.LogInfo("Receive KeepAlive Request from connection id %d",msg.Conn.Client.Fd)
			hlRequest := api.KeepAlive_Request{}
			status := uint32(api.ResultType_OK)
			if err := msg.ToRequestProtoModel(&hlRequest);err != nil{
				p.LogError("Request UnmarshalTo Hello_Request %v",err.Error())
				status = uint32(api.ResultType_REQUEST_INVALID)
			} else if !msg.Conn.Client.IsSetupConnection {
				p.LogError("Client chưa thiết lập mã hóa kết nối")
				status = uint32(api.ResultType_ENCRYPT_ERROR)

			} else if !bytes.Equal(msg.Conn.Connection_id,hlRequest.ConnectionId) {
				p.LogError("Connection ID Invalid")
				status = uint32(api.KeepAlive_CONNECTION_ID_INVALID)

			}else{
				msg.Conn.UpdateAt = uint64(time.Now().Unix())
			}
			_re := api.KeepAlive_Reply{}
			_buf ,err := GetReplyBuffer(status,msg.MsgType,msg.MsgGroup,msg.MSG_ID,&_re,msg.Conn.Client.EncType,msg.Conn.Client.PKey)
			if err != nil {
				p.LogError("Error %v",err.Error())
				return
			}
			if c,o := p.clients.Load(msg.Fd); o{
				(*c.(*gnet.Conn)).AsyncWrite(_buf,nil)
			}
		}else{
			p.LogInfo("Connection %d send STRANGE_REQUEST id %d",msg.Conn.Client.Fd,msg.MsgType)
			_buf ,err := GetReplyBuffer(uint32(api.ResultType_STRANGE_REQUEST),msg.MsgType,msg.MsgGroup,msg.MSG_ID,nil,msg.Conn.Client.EncType,msg.Conn.Client.PKey)
			if err != nil {
				p.LogError("Error %v",err.Error())
				return
			}
			if c,o := p.clients.Load(msg.Fd); o{
				(*c.(*gnet.Conn)).AsyncWrite(_buf,nil)
			}
		}
	}else{
		/// chuyển tiếp đến host tương ứng
	}



}

func (p *SocketServer) Close() {
	p.LogInfo("Close")
	close(p.Done)
}

