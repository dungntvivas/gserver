package gService

import (
	"gitlab.vivas.vn/go/grpc_api/api"
	"os"
	"os/signal"
	"runtime"

	"gitlab.vivas.vn/go/gserver/gBase"
	"gitlab.vivas.vn/go/gserver/gHTTP"
	"gitlab.vivas.vn/go/gserver/gRPC"
	"gitlab.vivas.vn/go/internal/logger"
)

type CallbackRequest func(request *api.Request,v_auth string,reply chan *api.Reply)
type gService struct {
	done           chan struct{}
	interrupt      chan os.Signal
	receiveRequest chan *gBase.Payload
	Logger         *logger.Logger
	cb             CallbackRequest
	SvName         string `default:"Service"`

	http_server *gHTTP.HTTPServer
	grpc_server *gRPC.GRPCServer
}

func New(_log *logger.Logger, configs ...gBase.ConfigOption) (*gService,int) {
	if _log == nil {
		return nil,-1
	}
	p := &gService{Logger: _log}
	p.done = make(chan struct{})
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
func (p *gService) Wait() {
	<-p.done
}

func (p *gService) miniWorker() {
	for j := range p.receiveRequest {
		if p.cb != nil {
             chrep := make(chan *api.Reply)
             p.cb(j.Request,j.V_Authorization,chrep)
             rep := <- chrep
             j.ChResult <- &gBase.Result{Status: int(rep.Status)}
		} else {
			j.ChResult <- &gBase.Result{Status: 1010} // request lạ
		}
	}
}

func (p *gService) StartListenAndReceiveRequest() chan struct{} {

	if p.http_server != nil {
		p.http_server.Serve()
	}

	if p.grpc_server != nil {
		p.grpc_server.Serve()
	}
	for num := 0; num < runtime.NumCPU()*2; num++ {
		go p.miniWorker()
	}
	go func() {
	loop:
		for {
			select {
			case <-p.done:
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
		p.done <- struct{}{}
	}()

	return p.done
}

func (p *gService) RegisterHandler(request CallbackRequest) {
	p.cb = request
}

func (p *gService) LogInfo(format string, args ...interface{}) {
	p.Logger.Log(logger.Info, "["+p.SvName+"] "+format, args...)
}
func (p *gService) LogDebug(format string, args ...interface{}) {
	p.Logger.Log(logger.Debug, "["+p.SvName+"] "+format, args...)
}
func (p *gService) LogError(format string, args ...interface{}) {
	p.Logger.Log(logger.Error, "["+p.SvName+"] "+format, args...)
}
