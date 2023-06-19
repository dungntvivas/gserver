package gService

import (
	"os"
	"os/signal"
	"runtime"
	"sync"

	"google.golang.org/protobuf/types/known/anypb"

	"github.com/golang/protobuf/jsonpb"
	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/gserver/gBase"
	"gitlab.vivas.vn/go/gserver/gRPC"
	"gitlab.vivas.vn/go/libinternal/logger"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type HandlerRequest func(request *api.Request, reply *api.Reply) uint32

type Service struct {
	Done           chan struct{}
	interrupt      chan os.Signal
	Logger         *logger.Logger
	receiveRequest chan *gBase.Payload
	SvName         string
	cb             HandlerRequest
	Operator       sync.Map
	grpc_server    *gRPC.GRPCServer
	gw_server      api.APIClient
	gw_enable      bool
}

func NewService(SvName string, _log *logger.Logger, config gBase.ConfigOption) *Service {
	p := &Service{
		Logger:         _log,
		Done:           make(chan struct{}),
		interrupt:      make(chan os.Signal, 1),
		receiveRequest: make(chan *gBase.Payload, runtime.NumCPU()*2),
		SvName:         SvName,
	}
	signal.Notify(p.interrupt, os.Interrupt)
	// config server http grpc ...
	config.Logger = _log
	p.grpc_server = gRPC.New(config, p.receiveRequest)

	return p
}
func (p *Service) PushMessage(pType gBase.Push_Type, receiver []string, ignore_Type gBase.Push_Type, ignore_receiver string, msg_type uint32, msg *api.Receive) bool {
	if !p.gw_enable {
		return false
	}
	_rq := api.Request{
		Type:  uint32(api.TYPE_INTERNAL_ID_PUSH_RECEIVE),
		Group: uint32(api.INTERNAL_GROUP_INTERNAL_GROUP_ID),
	}
	_rq_push := api.PushReceive_Request{
		PushType:       pType.Push_Type_to_proto_type(),
		Receiver:       receiver,
		ReceiveType:    msg_type,
		Ignore_Type:    ignore_Type.Push_Type_to_proto_type(),
		IgnoreReceiver: ignore_receiver,
		RcType:         msg.Type,
		RcGroup:        uint32(msg.Group),
	}
	var err error
	_rq_push.Receive, err = gBase.MsgToByte(msg)
	if err != nil {
		p.LogError("Send Push Error %v", err.Error())
		return false
	}
	_rq_push.ReceiveJson, err = protojson.Marshal(msg)
	if err != nil {
		p.LogError("Send Push Error %v", err.Error())
		return false
	}
	_rq.Request, err = anypb.New(&_rq_push)
	if err != nil {
		p.LogError("Send Push Error %v", err.Error())
		return false
	}
	_, err = gRPC.MakeRpcRequest(p.gw_server, &_rq)
	if err != nil {
		p.LogError("Send Push Error %v", err.Error())
		return false
	}

	return true
}
func (p *Service) SetCallBackRequest(cb HandlerRequest) {
	p.cb = cb
}
func (p *Service) LogInfo(format string, args ...interface{}) {
	p.Logger.Log(logger.Info, "["+p.SvName+"] "+format, args...)
}
func (p *Service) LogDebug(format string, args ...interface{}) {
	p.Logger.Log(logger.Debug, "["+p.SvName+"] "+format, args...)
}
func (p *Service) LogError(format string, args ...interface{}) {
	p.Logger.Log(logger.Error, "["+p.SvName+"] "+format, args...)
}
func (p *Service) Start() {
	/// start server
	if p.grpc_server != nil {
		p.grpc_server.Serve()
	}

	if p.grpc_server.Config.GW != nil && len(p.grpc_server.Config.GW) != 0 {
		var err error
		p.gw_server, err = gRPC.NewClientConn("gw", "gw.com.vivas.vn", p.grpc_server.Config.GW...)
		if err != nil {
			p.LogError("Start GRPC Client Error %v", err.Error())
			close(p.Done)
			return
		}
		p.gw_enable = true
		p.LogInfo("Register gw server ok ")
	}
	/// start worker
	for i := 0; i < runtime.NumCPU()*2; i++ {
		go p.worker(i)
	}
	go p.wait()
}
func (p *Service) wait() {
loop:
	for {
		select {
		case <-p.Done:
			break loop
		case <-p.interrupt:
			p.LogInfo("shutting down gracefully")
			close(p.Done)
		}
	}
	p.LogInfo("End Service")

	// STOP SERVER LISTEN

	if p.grpc_server != nil {
		p.grpc_server.Close()
	}
	close(p.receiveRequest)
}
func (p *Service) Stop() {
	close(p.Done)
}
func (p *Service) worker(id int) {
loop:
	for {
		select {
		case <-p.Done:
			break loop
		case req := <-p.receiveRequest:
			// call processRequest
			p.processRequest(req)

		}
	}
}
func (p *Service) processRequest(payload *gBase.Payload) {

	reply := &api.Reply{
		Status: 0,
		Msg:    "OK",
	}
	if payload.Request.PayloadType == uint32(gBase.PayloadType_BIN) {
		/// convert bin_request to proto request
		var _rq api.Request
		if err := proto.Unmarshal(payload.Request.BinRequest, &_rq); err != nil {
			p.LogError("proto.Unmarshal %v", err.Error())
			reply.Status = uint32(api.ResultType_INTERNAL_SERVER_ERROR)
			goto on_reply
		}
		payload.Request.Request = _rq.Request
		payload.Request.BinRequest = nil
	} else if payload.Request.PayloadType == uint32(gBase.PayloadType_JSON) {
		/// convert bin_json to proto request
		var _rq api.Request
		if err := jsonpb.UnmarshalString(string(payload.Request.BinRequest), &_rq); err != nil {
			p.LogError("jsonpb.UnmarshalString %v", err.Error())
			reply.Status = uint32(api.ResultType_INTERNAL_SERVER_ERROR)
			goto on_reply
		}
		payload.Request.Request = _rq.Request
		payload.Request.BinRequest = nil
	}
	if p.cb == nil {
		p.LogError("Callback handler not setup")
		reply.Status = uint32(api.ResultType_INTERNAL_SERVER_ERROR)
		goto on_reply
	}
	p.cb(payload.Request, reply)

on_reply:
	{
		var dataByte []byte
		if reply.Status >= 1000 {
			reply.Msg = api.ResultType(reply.Status).String()
		}
		reply.Group = payload.Request.Group
		reply.Type = payload.Request.Type
		if payload.Request.PayloadType == uint32(gBase.PayloadType_JSON) {
			dataByte, _ = protojson.Marshal(reply)
		} else if payload.Request.PayloadType == uint32(gBase.PayloadType_BIN) {
			dataByte, _ = proto.Marshal(reply)
		}

		reply.BinReply = dataByte

		payload.ChReply <- reply
		return
	}

}
