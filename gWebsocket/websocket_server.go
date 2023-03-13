package gWebsocket

import (
	"encoding/json"
	"runtime"
	"sync"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/panjf2000/gnet/v2"
	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/gserver/gBase"
)

type APIGenerated struct {
	Type  int    `json:"type"`
	Group string `json:"group"`
}

type chPayload struct {
	request *api.Request
	conn    *gnet.Conn
}

type WsServer struct {
	gBase.GServer
	gnet.BuiltinEventEngine
	Done         chan struct{}
	chReceiveMsg chan *chPayload
	clients      sync.Map
	mu           sync.Mutex
}

func New(config gBase.ConfigOption, chReceiveRequest chan *gBase.Payload) *WsServer {
	b := gBase.GServer{
		Config:           &config,
		ChReceiveRequest: chReceiveRequest,
	}
	p := WsServer{
		GServer:      b,
		Done:         make(chan struct{}),
		chReceiveMsg: make(chan *chPayload, 100),
	}
	return &p
}
func (p *WsServer) Close() {
	p.LogInfo("Close")
	close(p.Done)
}
func (p *WsServer) Serve() {
	go gnet.Run(p, "tcp://"+p.Config.Addr, gnet.WithMulticore(true), gnet.WithReusePort(true))
	for n := 0; n <= runtime.NumCPU(); n++ {
		go p.receiveMessage()
	}
}

func (p *WsServer) receiveMessage() {
loop:
	for {
		select {
		case <-p.Done:
			break loop
		case msg_payload := <-p.chReceiveMsg:
			var res api.Reply
			result := make(chan *api.Reply)
			p.HandlerRequest(&gBase.Payload{Request: msg_payload.request, ChReply: result, Connection_id: (*msg_payload.conn).Fd()})
			res = *<-result
			if msg_payload.request.PayloadType == uint32(gBase.PayloadType_JSON) {
				wsutil.WriteServerMessage(*msg_payload.conn, ws.OpText, res.BinReply)
			} else {
				wsutil.WriteServerMessage(*msg_payload.conn, ws.OpBinary, res.BinReply)
			}
		}
	}
}
func (p *WsServer) OnBoot(eng gnet.Engine) gnet.Action {
	p.LogInfo("Listener opened on %s", p.Config.Addr)
	return gnet.None
}
func (p *WsServer) OnShutdown(eng gnet.Engine) {

}
func (p *WsServer) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	p.LogInfo("conn [%v] Open Connection", c.Fd())
	c.SetContext(&wsCodec{vAuthorization: ""})
	p.mu.Lock()
	p.clients.Store(c.Fd(), &c)
	p.mu.Unlock()

	return
}
func (p *WsServer) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	p.LogInfo("conn [%v] Close", c.Fd())
	p.mu.Lock()
	p.clients.Delete(c.Fd())
	p.mu.Unlock()
	return gnet.Close
}
func (p *WsServer) MarkConnectioIsAuthen(token string, fd int) {
	p.mu.Lock()
	if c, ok := p.clients.Load(fd); ok {
		p.LogInfo("Save vAuthorization %v", token)
		(*c.(*gnet.Conn)).Context().(*wsCodec).vAuthorization = token
	}
	p.mu.Unlock()
}
func (p *WsServer) OnTraffic(c gnet.Conn) gnet.Action {

	codec := c.Context().(*wsCodec)
	if codec.readBufferBytes(c) == gnet.Close {
		return gnet.Close
	}
	ok, _ := codec.upgrade(c)
	if !ok {
		return gnet.Close
	}

	if codec.buf.Len() <= 0 {
		return gnet.None
	}
	messages, err := codec.Decode(c)
	if err != nil {
		return gnet.Close
	}
	if messages == nil {
		return gnet.None
	}
	for _, message := range messages {
		if message.OpCode == ws.OpText { /// json payload
			rq := APIGenerated{}
			if err := json.Unmarshal(message.Payload, &rq); err != nil {
				p.LogError("json.Unmarshal Error %v", err.Error())
				return gnet.None
			}

			// build request
			request := api.Request{
				PayloadType: uint32(gBase.PayloadType_JSON),
				Protocol:    uint32(gBase.RequestProtocol_WS),
				BinRequest:  message.Payload,
				Type:        uint32(rq.Type),
				Group:       api.Group(api.Group_value[rq.Group]),
				Session:     &api.Session{SessionId: codec.vAuthorization},
			}
			//request.Session = &api.Session{SessionId: vAuthorization}

			p.chReceiveMsg <- &chPayload{
				request: &request,
				conn:    &c,
			}
			return gnet.None
		} else if message.OpCode == ws.OpBinary { // binary request  payload
			// decode use socket payload protocol
			if _msg := gBase.DecodeData(message.Payload, len(message.Payload)); _msg != nil {

				request := api.Request{
					PayloadType: uint32(gBase.PayloadType_BIN),
					Protocol:    uint32(gBase.RequestProtocol_WS),
					BinRequest:  _msg.Payload,
					Type:        _msg.MsgType,
					Group:       api.Group(_msg.MsgGroup),
					Session:     &api.Session{SessionId: codec.vAuthorization},
				}
				p.chReceiveMsg <- &chPayload{
					request: &request,
					conn:    &c,
				}
			} else {
				p.LogInfo("Decode Msg Failure")
			}
			return gnet.None
		}
	}
	return gnet.None
}
