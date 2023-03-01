package gService
import (
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/gserver/gBase"
	"gitlab.vivas.vn/go/gserver/gHTTP"
	"gitlab.vivas.vn/go/gserver/gRPC"
	"gitlab.vivas.vn/go/internal/logger"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"os"
	"os/signal"
	"runtime"
	"sync"
)

type HandlerRequest func(request *api.Request,reply *api.Reply) uint32

type Service struct {
	Done chan struct{}
	interrupt      chan os.Signal
	Logger *logger.Logger
	receiveRequest chan *gBase.Payload
	SvName string
	cb HandlerRequest
	Operator sync.Map

	http_server *gHTTP.HTTPServer
	grpc_server *gRPC.GRPCServer
}
func NewService(SvName string,_log *logger.Logger, configs ...gBase.ConfigOption)*Service{
	p := &Service{
		Logger: _log,
		Done: make(chan struct{}),
		interrupt: make(chan os.Signal, 1),
		receiveRequest: make(chan *gBase.Payload,runtime.NumCPU()*2),
		SvName: SvName,
	}
	signal.Notify(p.interrupt, os.Interrupt)
	// config server http grpc ...
	for _, cf := range configs {
		if cf.Protocol == gBase.RequestProtocol_HTTP { // init http server listen
			cf.Logger = _log
			p.http_server = gHTTP.New(cf, p.receiveRequest)
		} else if cf.Protocol == gBase.RequestProtocol_GRPC {
			cf.Logger = _log
			p.grpc_server = gRPC.New(cf, p.receiveRequest)
		}
		// other service
	}

	return p
}
func (p *Service)SetCallBackRequest(cb HandlerRequest){
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

func (p *Service)Start(){
	/// start server
	if p.http_server != nil {
		p.http_server.Serve()
	}

	if p.grpc_server != nil {
		p.grpc_server.Serve()
	}
	/// start worker
	for i := 0;i<runtime.NumCPU()*2;i++ {
		go p.worker(i)
	}
	go p.wait()
}
func (p *Service)wait(){
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
	if p.http_server != nil {
		p.http_server.Close()
	}
	if p.grpc_server != nil {
		p.grpc_server.Close()
	}
	close(p.receiveRequest)
}
func (p *Service)Stop() {
	close(p.Done)
}

func (p *Service)worker(id int){
loop:
	for{
		select {
		case <-p.Done:
			break loop
		case req := <-p.receiveRequest:
			// call processRequest
			if(req.Request.Type == 1){
				// handler discovery service
				p.discoveryService(req)
			}else{
				p.processRequest(req)
			}

		}
	}
}
func (p *Service)discoveryService(payload *gBase.Payload){
	reply := &api.Reply{
		Status: 0,
		Msg: "OK",
	}
	discovery_reply := api.DiscoveryService_Reply{}
	p.operator.Range(func(key, value any) bool {
		discovery_reply.Types =  append(discovery_reply.Types, key.(uint32))
		return true
	})
	_discovery_reply, _ := anypb.New(&discovery_reply)
	reply.Reply = _discovery_reply
	payload.ChReply <- reply
}
func (p *Service)processRequest(payload *gBase.Payload){
	reply := &api.Reply{
		Status: 0,
		Msg: "OK",
	}
	if (payload.Request.PayloadType == uint32(gBase.PayloadType_BIN)){
		/// convert bin_request to proto request
		var _rq api.Request
		if err := jsonpb.UnmarshalString(string(payload.Request.BinRequest), &_rq);err != nil{
			p.LogError("%v",err.Error())
			reply.Status = uint32(api.ResultType_INTERNAL_SERVER_ERROR)
			goto on_reply
		}
		payload.Request.Request = _rq.Request
		payload.Request.BinRequest = nil
	}else if (payload.Request.PayloadType == uint32(gBase.PayloadType_JSON)){
		/// convert bin_json to proto request
		var _rq api.Request
		if err := jsonpb.UnmarshalString(string(payload.Request.BinRequest), &_rq); err != nil {
			p.LogError("%v",err.Error())
			reply.Status = uint32(api.ResultType_INTERNAL_SERVER_ERROR)
			goto on_reply
		}
		p.LogDebug("%v",_rq)
		payload.Request.Request = _rq.Request
		payload.Request.BinRequest = nil
	}
	if(p.cb == nil){
		p.LogError("Callback handler not setup")
		reply.Status = uint32(api.ResultType_INTERNAL_SERVER_ERROR)
		goto on_reply
	}
	p.cb(payload.Request,reply)


on_reply:{
	var dataByte []byte
	if(reply.Status >= 1000){
		reply.Msg = api.ResultType(reply.Status).String()
	}
	if (payload.Request.PayloadType == uint32(gBase.PayloadType_JSON)){
		dataByte, _ = protojson.Marshal(reply)
	}else if (payload.Request.PayloadType == uint32(gBase.PayloadType_BIN)){
		dataByte, _ = proto.Marshal(reply)
	}

	reply.BinReply = dataByte
	payload.ChReply <- reply
	return
}

}