package gService

import (
	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/gserver/gBase"
	"gitlab.vivas.vn/go/gserver/gHTTP"
	"gitlab.vivas.vn/go/gserver/gRPC"
	"gitlab.vivas.vn/go/internal/logger"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"os"
	"os/signal"
	"runtime"
)

type CallbackRequest func(*gBase.Payload)
type CallbackApiRequest func(request *api.Request,reply chan *api.Reply,v_auth string)
type GService struct {
	done           *chan struct{}
	interrupt      chan os.Signal
	receiveRequest chan *gBase.Payload
	Logger         *logger.Logger
	rawcb             CallbackRequest
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
					request := api.Request{}
					chreply := make(chan *api.Reply)
					if err := proto.Unmarshal(j.Request.BinRequest, &request); err != nil {
						j.ChResult <- &gBase.Result{Status: int(api.ResultType_REQUEST_INVALID), Reply: &api.Reply{Status: uint32(api.ResultType_REQUEST_INVALID)}}
						goto loop
					}
					p.cb(&request,chreply,j.V_Authorization)
					reply := <- chreply

					if(reply.Status >= 1000){
						reply.Msg = api.ResultType(reply.Status).String()
					}
					//convert reply to json or binary
					if j.Request.PayloadType == uint32(gBase.ContextType_BIN){
						b, err := proto.Marshal(reply)
						if err != nil {
							j.ChResult <- &gBase.Result{Status: int(api.ResultType_INTERNAL_SERVER_ERROR), Reply: &api.Reply{Status: uint32(api.ResultType_INTERNAL_SERVER_ERROR),Msg: "INTERNAL_SERVER_ERROR"}}
							goto loop
						}
						j.ChResult <- &gBase.Result{Status: int(reply.Status), Reply: reply,ReplyData: b,ContextType: gBase.ContextType(j.Request.PayloadType)}
					}else if j.Request.PayloadType == uint32(gBase.ContextType_JSON){
						jsonBytes, err := protojson.Marshal(reply)
						if err != nil {
							j.ChResult <- &gBase.Result{Status: int(api.ResultType_INTERNAL_SERVER_ERROR), Reply: &api.Reply{Status: uint32(api.ResultType_INTERNAL_SERVER_ERROR)}}
							goto loop
						}
						j.ChResult <- &gBase.Result{Status: int(reply.Status), Reply: reply,ReplyData: jsonBytes,ContextType: gBase.ContextType(j.Request.PayloadType)}
					}else if j.Request.PayloadType == uint32(gBase.ContextType_PROTO){ // return reply only
						j.ChResult <- &gBase.Result{Status: int(reply.Status), Reply: reply,ContextType: gBase.ContextType(j.Request.PayloadType)}
					}else{
						j.ChResult <- &gBase.Result{Status: int(api.ResultType_STRANGE_REQUEST), Reply: &api.Reply{Status: uint32(api.ResultType_STRANGE_REQUEST),Msg: "STRANGE_REQUEST"}}
					}
				}else if p.rawcb != nil {
					p.rawcb(j) // thường sử dụng cho gateway vì ko đụng chạm gì đến request mà chuyển tiếp trực tiếp
				} else {
					j.ChResult <- &gBase.Result{Status: int(api.ResultType_STRANGE_REQUEST), Reply: &api.Reply{Status: uint32(api.ResultType_STRANGE_REQUEST),Msg: "STRANGE_REQUEST"}}
				}
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

// RegisterHandlerRawRequest sử dụng khi muốn service call raw request từ server
func (p *GService) RegisterHandlerRawRequest(request CallbackRequest) {
	p.rawcb = request
}

// RegisterHandlerRequest sử dụng khi muốn service chuyển đổi payload từ raw sang api.request
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
