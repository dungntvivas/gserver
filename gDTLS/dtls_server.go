package gDTLS

import (
	"bytes"
	"context"
	"fmt"
	"github.com/pion/dtls/v2"
	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/gserver/gBase"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"net"
	"runtime"
	"strconv"
	"sync"
	"time"
)
type DTLSServer struct {
	gBase.GServer
	isRunning bool
	ctx context.Context
	cancel context.CancelFunc
	listener net.Listener


	mu      sync.Mutex
	clients sync.Map // ADDRESS ==> Connection ( 1 connection chứa thông tin kết nối ) client_id has connection

	mu_token sync.Mutex
	sessions sync.Map // token ==> FD ( 1 token có nhiều connection ) session_id has client_id

	mu_user sync.Mutex
	users   sync.Map // user ==> TOKEN ( 1 user có nhiều token ) user_id has session_id

	// chan
	chReceiveMsg chan *gBase.SocketMessage
	chClose chan string


}
func New(config gBase.ConfigOption, chReceiveRequest chan *gBase.Payload) *DTLSServer {

	b := gBase.GServer{
		Config:           &config,
		ChReceiveRequest: chReceiveRequest,
	}
	p := &DTLSServer{
		GServer: b,
		chReceiveMsg: make(chan *gBase.SocketMessage, 100),
		chClose: make(chan string),
		isRunning: true,
	}

	var err error
	p.ctx, p.cancel = context.WithCancel(context.Background())
	_config := dtls.Config{
		PSK: func(hint []byte) ([]byte, error) {
			return []byte{0x86, 0x73, 0x86, 0x65, 0x83}, nil
		},
		PSKIdentityHint:      []byte("VIVAS-RDPA DTLS Client"),
		CipherSuites:         []dtls.CipherSuiteID{dtls.TLS_PSK_WITH_AES_256_CCM_8},
		ExtendedMasterSecret: dtls.RequireExtendedMasterSecret,
		ConnectContextMaker: func() (context.Context, func()) {
			return context.WithTimeout(p.ctx, 30*time.Second)
		},
	}
	_, _port, err1 := net.SplitHostPort(p.Config.Addr)
	if err1 != nil {
		p.LogError("SplitHostPort %v",err1.Error())
		return nil
	}
	port ,err2 := strconv.ParseInt(_port,10,0)
	if err2 != nil {
		p.LogError("strconv.ParseInt %v",err2.Error())
		return nil
	}
	addr := &net.UDPAddr{IP: net.ParseIP(p.Config.Addr), Port: int(port)}
	p.listener, err = dtls.Listen("udp", addr, &_config)
	if err != nil {
		p.LogError("dtls listen %v",err.Error())
		return nil
	}



	return p
}

func (p *DTLSServer) Serve() error {
	p.LogInfo("Listening")
	go p.wait_for_new_connection()
	for i := 0; i < runtime.NumCPU(); i++ {
		go p.receiveMsg()
	}
	return nil
}
func (p *DTLSServer)connection_close() {
loop:
	for{
		select {
		case key := <-p.chClose:
			p.mu.Lock()
			defer p.mu.Unlock()
			if _ ,ok := p.clients.Load(key);ok{
				p.clients.Delete(key)
			}
		case <- *p.Config.Done:
			p.isRunning = false
			break loop
		}
	}
}
func (p *DTLSServer)wait_for_new_connection(){
	// Wait for a connection.
	for{
		if !p.isRunning {
			break
		}
		conn, err := p.listener.Accept()
		if err != nil { /// store connection
			p.LogInfo("New Connection Error %v",err.Error())
		}

		// store connection
		_, _port, err := net.SplitHostPort(conn.RemoteAddr().String())
		if err != nil {
			p.LogError("SplitHostPort %v",err.Error())
			continue
		}
		port ,err := strconv.ParseInt(_port,10,0)
		if err != nil {
			p.LogError("strconv.ParseInt %v",err.Error())
			continue
		}

		key := fmt.Sprintf("%s_%s", p.Config.Protocol.String(), _port)
		p.LogInfo("Client %v %v",key,conn.RemoteAddr().Network())
		p.mu.Lock()
		defer p.mu.Unlock()
		newConn := gBase.NewConnection(&gBase.ServerConnection{
			DecType:           p.Config.EncodeType,
		},&gBase.ClientConnection{
			Fd: int(port),
			Conn: &conn,
		})
		p.clients.Store(key,newConn)
		go p.readMsg(newConn)
		// send hello Msg
		p.SendHelloMsg(newConn)
	}
}

func (p *DTLSServer)readMsg(conn *gBase.Connection){
	b := make([]byte, 8192)
	for {
		if !p.isRunning {
			break
		}
		n, err := (*conn.Client.Conn).Read(b)
		if err != nil {
			//h.unregister(conn)
			key := fmt.Sprintf("%v_%v", p.Config.Protocol.String(), conn.Client.Fd)
			p.chClose <- key
			break
		}
		msgs := gBase.DecodeDTLSPacket(b[:n])
		for i := range msgs {
			msg := msgs[i]
			msg.Conn = conn
			msg.Fd = conn.Client.Fd
			p.chReceiveMsg <- msg
		}
	}
}

func (p *DTLSServer)receiveMsg(){
	loop:
		for {
			select {
			case <-*p.Config.Done:
				break loop
			case msg := <- p.chReceiveMsg:
				if msg.MsgType == uint32(api.TYPE_ID_REQUEST_HELLO) && msg.MsgGroup == uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID) {
					p.onSetupConnection(msg)
				}else if msg.MsgType == uint32(api.TYPE_ID_REQUEST_KEEPALIVE) && msg.MsgGroup == uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID) {
					p.onClientKeepAlive(msg)
				}else if msg.MsgGroup != uint32(api.AUTHEN_GROUP_AUTHEN_GROUP_ID) {
					rq := api.Request{}
					rq.Type = msg.MsgType
					rq.Group = msg.MsgGroup
					rq.BinRequest = msg.Payload
					rq.PayloadType = uint32(msg.TypePayload)
					rq.Protocol = uint32(p.Config.Protocol)
					rq.Session = &api.Session{SessionId: msg.Conn.Session_id}
					result := make(chan *api.Reply)
					_payload := gBase.Payload{Request: &rq, ChReply: result, Connection_id: fmt.Sprintf("%s_%d", p.Config.Protocol.String(), msg.Fd)}
					if msg.Conn.IsOK() {
						_payload.IsAuth = true
						_payload.Session_id = msg.Conn.Session_id
						_payload.User_id = msg.Conn.User_id
					}

					p.HandlerRequest(&_payload)
					res := *<-result
					res.Type = rq.Type
					res.Group = rq.Group
					if res.Status != 0 {
						res.Msg = api.ResultType(res.Status).String()
					}
					if _buf, err := gBase.GetReplyBuffer(msg.MsgType, msg.MsgGroup, msg.MSG_ID, &res, msg.Conn.Client.EncType, msg.Conn.Client.PKey); err == nil {
						if c, o := p.clients.Load(fmt.Sprintf("%s_%d", p.Config.Protocol.String(), msg.Fd)); o {
							(*c.(*net.Conn)).Write(_buf)
						}
					}
				}

			}
		}
}
func (p *DTLSServer) onClientKeepAlive(msg *gBase.SocketMessage) {
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

	conn_id := 0

	conn_id = int(hlRequest.ConnectionId[0]) << 8
	conn_id += int(hlRequest.ConnectionId[1])

	//if hlRequest.ConnectionId

	_re := api.KeepAlive_Reply{}
	_nre := gBase.NewReply(status)
	gBase.PackReply(_nre, &_re)
	_buf, err := gBase.GetReplyBuffer(msg.MsgType, msg.MsgGroup, msg.MSG_ID, _nre, msg.Conn.Client.EncType, msg.Conn.Client.PKey)
	if err != nil {
		p.LogError("Error %v", err.Error())
		return
	}
	if c, o := p.clients.Load(fmt.Sprintf("%s_%d", p.Config.Protocol.String(), msg.Fd)); o {
		conn := c.(*gBase.Connection)
		(*conn.Client).Lock.RLock()
		defer (*conn.Client).Lock.RUnlock()
		(*conn.Client.Conn).Write(_buf)
	}
}
func (p *DTLSServer) onSetupConnection(msg *gBase.SocketMessage) {
	p.LogInfo("onSetupConnection")
	hlRequest := api.Hello_Request{}
	status := uint32(api.ResultType_OK)
	ok := false
	if err := msg.ToRequestProtoModel(&hlRequest); err != nil {
		p.LogError("Binary Request UnmarshalTo Hello_Request %v", err.Error())

	} else {
		ok = true
	}

	if !ok {
		p.LogInfo("ResultType_REQUEST_INVALID")
		status = uint32(api.ResultType_REQUEST_INVALID)
	} else {
		// process
		msg.Conn.Client.IsSetupConnection = true
		msg.Conn.Client.PKey = hlRequest.PKey
		msg.Conn.Client.EncType = gBase.Encryption_Type(hlRequest.EncodeType)

		msg.Conn.Client.Platfrom = int32(hlRequest.Platform)
		msg.Conn.Connection_id = []byte(fmt.Sprintf("%d", msg.Fd))
	}
	/// BUILD REPLY
	hlreply := api.Hello_Reply{
		ConnectionId:     msg.Conn.Connection_id,
		ServerTime:       uint64(time.Now().Unix()),
		ServerEncodeType: api.EncodeType(msg.Conn.Server.DecType),
		PKey:             msg.Conn.Server.PKey,
	}
	_repl := gBase.NewReply(status)
	gBase.PackReply(_repl, &hlreply)
	_buf, err := gBase.GetReplyBuffer(msg.MsgType, msg.MsgGroup, msg.MSG_ID, _repl, msg.Conn.Client.EncType, msg.Conn.Client.PKey)
	if err != nil {
		p.LogError("Error %v", err.Error())
	}
	if c, o := p.clients.Load(fmt.Sprintf("%s_%d", p.Config.Protocol.String(), msg.Fd)); o {
		conn := c.(*gBase.Connection)
		(*conn.Client).Lock.RLock()
		defer (*conn.Client).Lock.RUnlock()
		(*conn.Client.Conn).Write(_buf)
		p.LogInfo("Client %d Setup encode type %s", (*conn.Client).Fd,(*conn.Client).EncType.String())
	}
}


func (p *DTLSServer) Close() {
	p.LogInfo("Close")
	p.cancel()
	p.listener.Close()

}

func (p *DTLSServer) MarkConnectioIsAuthen(token string, user_id string, fd string, payload_type gBase.PayloadType) {
	p.LogDebug("Mark Connection %v - %v - %v", fd, token, user_id)
}

func (p *DTLSServer) SendHelloMsg(conn *gBase.Connection) {
	p.LogInfo("Send Msg Hello to conn %v",conn.Client.Fd)
	helloReceive := api.HelloReceive{
		ServerTime:       uint64(time.Now().Unix()),
		PKey:             conn.Server.PKey,
		ServerEncodeType: api.EncodeType(conn.Server.DecType),
	}
	receive := api.Receive{
		Type:       uint32(api.TYPE_ID_RECEIVE_HELLO),
		Group:      uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID),
		ServerTime: helloReceive.ServerTime,
	}
	_receiveAny, _ := anypb.New(&helloReceive)
	receive.Receive = _receiveAny
	_receive_bin, _ := proto.Marshal(&receive)
	msg := gBase.NewMessage(_receive_bin, uint32(receive.Group), receive.Type, []byte{0x86, 0x73, 0x86, 0x65, 0x83})
	out, _ := msg.Encode(gBase.Encryption_NONE, nil, true)
	(*conn.Client).Lock.RLock()
	defer (*conn.Client).Lock.RUnlock()
	(*conn.Client.Conn).Write(out)
}
