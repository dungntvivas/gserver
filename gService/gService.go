package gService

import (
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/gserver/gBase"
	"gitlab.vivas.vn/go/gserver/gHTTP"
	"gitlab.vivas.vn/go/gserver/gRPC"
	"gitlab.vivas.vn/go/internal/logger"
	"os"
	"os/signal"
	"runtime"
)

type CallbackApiRequest func(request *api.Request,reply chan *api.Reply)
type GService struct {
	done           *chan struct{}
	interrupt      chan os.Signal
	receiveRequest chan *gBase.Payload
	Logger         *logger.Logger
	cb CallbackApiRequest
	SvName         string `default:"Service"`



	http_server *gHTTP.HTTPServer
	grpc_server *gRPC.GRPCServer
}

func New(_log *logger.Logger,_done *chan struct{}, configs ...gBase.ConfigOption) (*GService,int) {
	if _log == nil {
		return nil,-1
	}
	p := &GService{Logger: _log}
	p.done = _done
	p.receiveRequest = make(chan *gBase.Payload, 100)
	p.interrupt = make(chan os.Signal, 1)
	signal.Notify(p.interrupt, os.Interrupt)

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

	return p,0
}
func (p *GService) Wait() {
	<-*p.done
}

func (p *GService)runner(){
    loop:
    	for{
			select {
			case <-*p.done:
				break loop
			case j := <- p.receiveRequest:
				if p.cb != nil {
					chreply := make(chan *api.Reply)
					p.LogDebug("Service Send Request")
					// convert request if need
					if j.Request.PayloadType == uint32(gBase.PayloadType_JSON) {
						var _rq api.Request
						if err := jsonpb.UnmarshalString(string(j.Request.BinRequest), &_rq);err != nil{
							p.LogError("Convert Json to Proto Error")
							p.LogError("%v",err.Error())
							continue
						}
						j.Request.Request = _rq.Request
						j.Request.BinRequest = nil
					}else if j.Request.PayloadType == uint32(gBase.PayloadType_BIN){
						var _rq api.Request
						if err := proto.Unmarshal(j.Request.BinRequest, &_rq); err != nil {
							p.LogError("Convert bin to Proto Error")
							p.LogError("%v",err.Error())
							continue
						}
						j.Request.Request = _rq.Request
						j.Request.BinRequest = nil
					}

					p.LogDebug("%v",j.Request)


					p.cb(j.Request,chreply)
					reply := <- chreply
					p.LogDebug("Service Reply")
					if(reply.Status >= 1000){
						reply.Msg = api.ResultType(reply.Status).String()
					}
					if j.Request.PayloadType == uint32(gBase.PayloadType_JSON) {
						// convert reply to json data
					}
					j.ChResult <- &gBase.Result{
						Status: int(reply.Status),
						ContextType: gBase.PayloadType(j.Request.PayloadType),
						Reply: reply,
					}

				}else{
					j.ChResult <- &gBase.Result{
						Status: int(api.ResultType_STRANGE_REQUEST),
						ContextType: gBase.PayloadType(j.Request.PayloadType),
						Reply: &api.Reply{
							Status: uint32(api.ResultType_STRANGE_REQUEST),
							Msg: "STRANGE_REQUEST",
						},
					}
				}
				//if p.cb != nil {
				//	request := j.Request
				//	chreply := make(chan *api.Reply)
				//	if request.PayloadType == uint32(gBase.PayloadType_BIN){ /// bin to ptoto
				//		if err := proto.Unmarshal(j.Request.BinRequest, request); err != nil {
				//			p.LogError("Lỗi đọc data  body = [%v]", err.Error())
				//			j.ChResult <- &gBase.Result{Status: int(api.ResultType_REQUEST_INVALID), Reply: &api.Reply{Status: uint32(api.ResultType_REQUEST_INVALID)}}
				//			goto loop
				//		}
				//	}else if request.PayloadType == uint32(gBase.PayloadType_JSON){ /// json to proto
				//		if err := jsonpb.UnmarshalString(string(j.Request.BinRequest), request); err != nil {
				//			p.LogError("Lỗi đọc data json body = [%v]", err.Error())
				//			goto loop
				//		}
				//	}else{ // request copy
				//		request = j.Request
				//	}
				//
				//	p.cb(request,chreply,j.V_Authorization)
				//	reply := <- chreply
				//
				//	if(reply.Status >= 1000){
				//		reply.Msg = api.ResultType(reply.Status).String()
				//	}
				//	//convert reply to json or binary
				//	if j.Request.PayloadType == uint32(gBase.PayloadType_BIN){
				//		b, err := proto.Marshal(reply)
				//		if err != nil {
				//			j.ChResult <- &gBase.Result{Status: int(api.ResultType_INTERNAL_SERVER_ERROR), Reply: &api.Reply{Status: uint32(api.ResultType_INTERNAL_SERVER_ERROR),Msg: "INTERNAL_SERVER_ERROR"}}
				//			goto loop
				//		}
				//		j.ChResult <- &gBase.Result{Status: int(reply.Status), Reply: reply,ReplyData: b,PayloadType: gBase.PayloadType(j.Request.PayloadType)}
				//	}else if j.Request.PayloadType == uint32(gBase.PayloadType_JSON){
				//		jsonBytes, err := protojson.Marshal(reply)
				//		if err != nil {
				//			j.ChResult <- &gBase.Result{Status: int(api.ResultType_INTERNAL_SERVER_ERROR), Reply: &api.Reply{Status: uint32(api.ResultType_INTERNAL_SERVER_ERROR)}}
				//			goto loop
				//		}
				//		j.ChResult <- &gBase.Result{Status: int(reply.Status), Reply: reply,ReplyData: jsonBytes,PayloadType: gBase.PayloadType(j.Request.PayloadType)}
				//	}else if j.Request.PayloadType == uint32(gBase.PayloadType_PROTO){ // return reply only
				//		j.ChResult <- &gBase.Result{Status: int(reply.Status), Reply: reply,PayloadType: gBase.PayloadType(j.Request.PayloadType)}
				//	}else{
				//		j.ChResult <- &gBase.Result{Status: int(api.ResultType_STRANGE_REQUEST), Reply: &api.Reply{Status: uint32(api.ResultType_STRANGE_REQUEST),Msg: "STRANGE_REQUEST"},PayloadType: gBase.PayloadType(j.Request.PayloadType)}
				//	}
				//} else {
				//	j.ChResult <- &gBase.Result{Status: int(api.ResultType_STRANGE_REQUEST), Reply: &api.Reply{Status: uint32(api.ResultType_STRANGE_REQUEST),Msg: "STRANGE_REQUEST"},PayloadType: gBase.PayloadType(j.Request.PayloadType)}
				//}
			}
		}
}

func (p *GService) StartListenAndReceiveRequest() chan struct{} {

	if p.http_server != nil {
		p.http_server.Serve()
	}

	if p.grpc_server != nil {
		p.grpc_server.Serve()
	}

	for i := 0;i<runtime.NumCPU()*2;i++ {
		go p.runner()
	}

	go func() {
	loop:
		for {
			select {
			case <-*p.done:
				break loop
			case <-p.interrupt:
				p.LogInfo("shutting down gracefully")
				break loop
			}
		}

		p.LogInfo("End Service")

		if p.http_server != nil {
			p.http_server.Close()
		}
		if p.grpc_server != nil {
			p.grpc_server.Close()
		}
		close(p.receiveRequest)
		close(*p.done)
	}()


	return *p.done
}

func (p *GService) RegisterHandlerRequest(request CallbackApiRequest) {
	p.cb = request
}

func (p *GService) LogInfo(format string, args ...interface{}) {
	p.Logger.Log(logger.Info, "["+p.SvName+"] "+format, args...)
}
func (p *GService) LogDebug(format string, args ...interface{}) {
	p.Logger.Log(logger.Debug, "["+p.SvName+"] "+format, args...)
}
func (p *GService) LogError(format string, args ...interface{}) {
	p.Logger.Log(logger.Error, "["+p.SvName+"] "+format, args...)
}
