package gBase

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/gobwas/ws"
	"github.com/panjf2000/gnet/v2"
	"gitlab.vivas.vn/go/grpc_api/api"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type SocketServer struct {
	GServer
	gnet.BuiltinEventEngine
	lis          net.Listener
	Done         chan struct{}
	chReceiveMsg chan *SocketMessage
	clients      sync.Map
	mu           sync.Mutex
}
type APIGenerated struct {
	Type  int    `json:"type"`
	Group string `json:"group"`
}

func NewSocket(config ConfigOption, chReceiveRequest chan *Payload) SocketServer {

	b := GServer{
		Config:           &config,
		ChReceiveRequest: chReceiveRequest,
	}
	p := SocketServer{
		GServer:      b,
		Done:         make(chan struct{}),
		chReceiveMsg: make(chan *SocketMessage, 100),
	}
	return p
}
func (p *SocketServer) Serve() {
	protocol := "tcp"
	if p.Config.Protocol == RequestProtocol_TCP || p.Config.Protocol == RequestProtocol_WS {
		protocol = "tcp://"
	} else if p.Config.Protocol == RequestProtocol_UDP {
		protocol = "udp://"
	} else if p.Config.Protocol == RequestProtocol_UDS {
		protocol = "unix://"
	}
	go gnet.Run(p, protocol+p.Config.Addr, gnet.WithMulticore(true), gnet.WithReusePort(true))
	for i := 0; i < runtime.NumCPU()*2; i++ {
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
	p.LogInfo("conn [%v] Open Connection", c.Fd())
	// build msg  hello send to client

	newConn := NewConnection(&ServerConnection{
		DecType:           p.Config.EncodeType,
		IsSetupConnection: true,
	}, &ClientConnection{
		Fd: c.Fd(),
	})
	newConn.Session_id = ""
	c.SetContext(newConn)
	p.mu.Lock()
	p.clients.Store(c.Fd(), &c)
	p.mu.Unlock()
	if p.Config.Protocol != RequestProtocol_WS {
		go p.SendHelloMsg(newConn, &c)
	}

	return
}

func (p *SocketServer) SendHelloMsg(newConn *Connection, _c *gnet.Conn) {
	/// build hello
	bytes := make([]byte, 5) //generate a random 32 byte key for AES-256
	rand.Read(bytes)
	helloReceive := api.HelloReceive{
		ServerTime:       uint64(time.Now().Unix()),
		PKey:             newConn.Server.PKey,
		ServerEncodeType: api.EncodeType(newConn.Server.DecType),
	}
	receive := api.Receive{
		Type:       uint32(api.TYPE_ID_RECEIVE_HELLO),
		ServerTime: helloReceive.ServerTime,
	}
	_receiveAny, _ := anypb.New(&helloReceive)
	receive.Receive = _receiveAny
	_receive_bin, _ := proto.Marshal(&receive)
	msg := NewMessage(_receive_bin, 0, receive.Type, bytes)
	out, _ := msg.Encode(Encryption_NONE, nil)
	(*_c).AsyncWrite(out, nil)
}

func (p *SocketServer) MarkConnectioIsAuthen(token string, fd int) {
	p.mu.Lock()
	if c, ok := p.clients.Load(fd); ok {
		(*c.(*gnet.Conn)).Context().(*Connection).Client.IsAuthen = true
		(*c.(*gnet.Conn)).Context().(*Connection).Client.IsSetupConnection = true
		(*c.(*gnet.Conn)).Context().(*Connection).Session_id = token
	}
	p.mu.Unlock()
}

func (p *SocketServer) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	p.LogInfo("conn [%v] Close", c.Fd())
	p.mu.Lock()
	p.clients.Delete(c.Fd())
	p.mu.Unlock()
	return gnet.Close
}

func (p *SocketServer) OnTraffic(c gnet.Conn) gnet.Action {
	if p.Config.Protocol == RequestProtocol_WS {
		msgs, action := WebsocketDecodePackage(p.Config.Logger,c)
		for i := range msgs {
			p.chReceiveMsg <- msgs[i]
		}
		return action
	} else {
		msgs := DecodePacket(p.Config.Logger, c)
		for i := range msgs {
			p.chReceiveMsg <- msgs[i]
		}
		return gnet.None

	}
}
func (p *SocketServer) receiveMessage() {
loop:
	for {
		select {
		case <-p.Done:
			break loop
		case msg := <-p.chReceiveMsg:
			p.onReceiveRequest(msg)
		}
	}
}

func (p *SocketServer)onSetupConnection(msg *SocketMessage)  {
	hlRequest := api.Hello_Request{}
	status := uint32(api.ResultType_OK)
	if err := msg.ToRequestProtoModel(&hlRequest); err != nil {
		p.LogError("Request UnmarshalTo Hello_Request %v", err.Error())
		status = uint32(api.ResultType_REQUEST_INVALID)
	} else {
		// process
		msg.Conn.Client.IsSetupConnection = true
		msg.Conn.Client.PKey = hlRequest.PKey
		msg.Conn.Client.EncType = Encryption_Type(hlRequest.EncodeType)
		p.LogInfo("Client %d Setup encode type %s", msg.Conn.Client.Fd, msg.Conn.Client.EncType.String())
		msg.Conn.Client.Platfrom = int32(hlRequest.Platform)
		msg.Conn.Connection_id, _ = uuid.New().MarshalBinary()
	}
	/// BUILD REPLY
	hlreply := api.Hello_Reply{
		ConnectionId:     msg.Conn.Connection_id,
		ServerTime:       uint64(time.Now().Unix()),
		ServerEncodeType: api.EncodeType(msg.Conn.Server.DecType),
		PKey: msg.Conn.Server.PKey,
	}
	_buf, err := GetReplyBuffer(status, msg.MsgType, msg.MsgGroup, msg.MSG_ID, &hlreply, msg.Conn.Client.EncType, msg.Conn.Client.PKey)
	if err != nil {
		p.LogError("Error %v", err.Error())
	}
	if c, o := p.clients.Load(msg.Fd); o{
		if p.Config.Protocol == RequestProtocol_WS {
			wsutil.WriteServerMessage((*c.(*gnet.Conn)), ws.OpBinary, _buf)
		}else{
			(*c.(*gnet.Conn)).AsyncWrite(_buf, nil)
		}

	}
}
func (p *SocketServer)onClientKeepAlive(msg *SocketMessage){
	p.LogInfo("Receive KeepAlive Request from connection id %d", msg.Conn.Client.Fd)
	hlRequest := api.KeepAlive_Request{}
	status := uint32(api.ResultType_OK)
	if err := msg.ToRequestProtoModel(&hlRequest); err != nil {
		p.LogError("Request UnmarshalTo Hello_Request %v", err.Error())
		status = uint32(api.ResultType_REQUEST_INVALID)
	} else if !msg.Conn.Client.IsSetupConnection {
		p.LogError("Client chưa thiết lập mã hóa kết nối")
		status = uint32(api.ResultType_ENCRYPT_ERROR)

	} else if !bytes.Equal(msg.Conn.Connection_id, hlRequest.ConnectionId) {
		p.LogError("Connection ID Invalid")
		status = uint32(api.KeepAlive_CONNECTION_ID_INVALID)

	} else {
		msg.Conn.UpdateAt = uint64(time.Now().Unix())
	}
	_re := api.KeepAlive_Reply{}
	_buf, err := GetReplyBuffer(status, msg.MsgType, msg.MsgGroup, msg.MSG_ID, &_re, msg.Conn.Client.EncType, msg.Conn.Client.PKey)
	if err != nil {
		p.LogError("Error %v", err.Error())
		return
	}
	if c, o := p.clients.Load(msg.Fd); o {
		if p.Config.Protocol == RequestProtocol_WS {
			wsutil.WriteServerMessage((*c.(*gnet.Conn)), ws.OpBinary, _buf)
		}else{
			(*c.(*gnet.Conn)).AsyncWrite(_buf, nil)
		}
	}
}




func (p *SocketServer) onReceiveRequest(msg *SocketMessage) {

	if msg.MsgType == uint32(api.TYPE_ID_REQUEST_HELLO) && msg.MsgGroup == uint32(api.Group_CONNECTION){
		p.onSetupConnection(msg)
		return
	}
	if (msg.MSG_encode_decode_type != msg.Conn.Server.DecType || !msg.Conn.Client.IsSetupConnection) && msg.TypePayload == PayloadType_BIN {
		p.LogError("Connection Decode Invalid [server %v - payload %v]",msg.Conn.Server.DecType,msg.MSG_encode_decode_type)
		return
	}

	if msg.MsgType == uint32(api.TYPE_ID_REQUEST_KEEPALIVE) && msg.MsgGroup == uint32(api.Group_CONNECTION){
		p.onClientKeepAlive(msg)
		return
	}
	if msg.MsgGroup != uint32(api.Group_AUTHEN){
		// check authen
		if !msg.Conn.isOK() {
			p.LogError("Connection %d - isAuthen %v -- group %v -- type %v",msg.Fd,msg.Conn.Client.IsAuthen,msg.MsgGroup,msg.MsgType)
			if _buf, err := GetReplyBuffer(uint32(api.ResultType_REQUEST_INVALID), msg.MsgType, msg.MsgGroup, msg.MSG_ID, nil, Encryption_NONE, nil); err == nil {
				if c, o := p.clients.Load(msg.Fd); o {
					if p.Config.Protocol == RequestProtocol_WS {
						if(msg.TypePayload == PayloadType_JSON){
							wsutil.WriteServerMessage((*c.(*gnet.Conn)), ws.OpText, []byte(fmt.Sprintf("{\"status\":%d, \"msg\":\"%s\"}", uint32(api.ResultType_REQUEST_INVALID), "REQUEST_INVALID")))
						}else{
							wsutil.WriteServerMessage((*c.(*gnet.Conn)), ws.OpBinary, _buf)
						}

					}else{
						(*c.(*gnet.Conn)).AsyncWrite(_buf, nil)
					}
				}
			}

			return
		}
	}
	rq := api.Request{}
	rq.Type = msg.MsgType
	rq.Group = api.Group(msg.MsgGroup)
	rq.BinRequest = msg.Payload
	rq.PayloadType = uint32(msg.TypePayload)
	rq.Protocol = uint32(p.Config.Protocol)
	rq.Session = &api.Session{SessionId: msg.Conn.Session_id}
	result := make(chan *api.Reply)
	p.HandlerRequest(&Payload{Request: &rq, ChReply: result, Connection_id: msg.Fd})
	res := *<-result
	if res.Status != 0 {
		res.Msg = api.ResultType(res.Status).String()
	}

	if _buf, err := GetReplyBuffer(res.Status , msg.MsgType, msg.MsgGroup, msg.MSG_ID, &res, msg.Conn.Client.EncType, msg.Conn.Client.PKey); err == nil {
		if c, o := p.clients.Load(msg.Fd); o {
			if p.Config.Protocol == RequestProtocol_WS {
				if(msg.TypePayload == PayloadType_JSON){
					wsutil.WriteServerMessage((*c.(*gnet.Conn)), ws.OpText, res.BinReply)
				}else{
					wsutil.WriteServerMessage((*c.(*gnet.Conn)), ws.OpBinary, _buf)
				}
			}else{
				(*c.(*gnet.Conn)).AsyncWrite(_buf, nil)
			}
		}
	}
}

func (p *SocketServer) Close() {
	p.LogInfo("Close")
	close(p.Done)
}
