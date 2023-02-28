package gService

import (
	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/gserver/gBase"
	"gitlab.vivas.vn/go/gserver/gHTTP"
	"gitlab.vivas.vn/go/gserver/gRPC"
	"gitlab.vivas.vn/go/internal/logger"
	"os"
	"os/signal"
)

type CallbackRequest func(*gBase.Payload)
type GService struct {
	done           *chan struct{}
	interrupt      chan os.Signal
	receiveRequest chan *gBase.Payload
	Logger         *logger.Logger
	cb             CallbackRequest
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

func (p *GService) StartListenAndReceiveRequest() chan struct{} {

	if p.http_server != nil {
		p.http_server.Serve()
	}

	if p.grpc_server != nil {
		p.grpc_server.Serve()
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
			case j := <-p.receiveRequest:
				if p.cb != nil {
					p.cb(j)
				} else {
					j.ChResult <- &gBase.Result{Status: int(api.ResultType_STRANGE_REQUEST), Reply: &api.Reply{Status: uint32(api.ResultType_STRANGE_REQUEST)}}
				}
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
		*p.done <- struct{}{}


	}()


	return *p.done
}

func (p *GService) RegisterHandler(request CallbackRequest) {
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
