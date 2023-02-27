package gHTTP

import (
	"context"
	"io"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/gserver/gBase"
)

type RequestID struct {
	TYPE  int `uri:"type" binding:"required"`
	GROUP int `uri:"group" binding:"required"`
}

type HTTPServer struct {
	gBase.GServer
	listen net.Listener
	http_sv *http.Server
}

func New(config gBase.ConfigOption, chReceiveRequest chan *gBase.Payload) *HTTPServer {

	b := gBase.GServer{
		Config:           &config,
		ChReceiveRequest: chReceiveRequest,
	}
	p := &HTTPServer{
		GServer: b,
	}
	if p.Config.Tls.IsTLS {
		p.Config.ServerName = "HTTPS"
		if p.Config.Tls.H2_Enable {
			p.Config.ServerName = "H2"
		}
	} else {
		p.Config.ServerName = "HTTP"

	}
	gin.SetMode(gin.ReleaseMode)
	http_sv := gin.New()
	gV1 := http_sv.Group("/v1")
	{
		gV1.POST("/:group/:type", p.onReceiveRequest)
	}
	p.http_sv = &http.Server{
		Handler: http_sv,
	}

	return p
}

func (p *HTTPServer) Serve() error {
	p.LogInfo("Start %v server ", p.Config.ServerName)
    var err error
	p.listen, err = net.Listen("tcp", p.Config.Addr)
	if err != nil {
		return err
	}
	if p.Config.Tls.IsTLS {
		if p.Config.Tls.H2_Enable {
			go p.http_sv.ServeTLS(p.listen, p.Config.Tls.Cert, p.Config.Tls.Key)
		} else {
			go p.http_sv.ServeTLS(p.listen, p.Config.Tls.Cert, p.Config.Tls.Key)
		}
		p.LogInfo("Listener opened on %s", p.Config.Addr)
	} else {
		go p.http_sv.Serve(p.listen)
		p.LogInfo("Listener opened on %s", p.Config.Addr)
	}
	return nil
}

func (p *HTTPServer) Close(){
	p.LogInfo("Close")
	p.http_sv.Shutdown(context.Background())
	p.listen.Close()
}

func (p *HTTPServer) onReceiveRequest(ctx *gin.Context) {
	var urlParams RequestID
	status := http.StatusOK
	var res gBase.Result
	result := make(chan *gBase.Result)
	defer close(result)
	contenxtType := ctx.Request.Header.Get("Content-Type")
	ctType := gBase.ContextType_JSON
	var bindata []byte
	var err error
	if contenxtType == "application/octet-stream" {
		ctType = gBase.ContextType_BIN
	}
	request := &api.Request{
		Protocol:    uint32(gBase.RequestProtocol_HTTP),
		PayloadType: uint32(ctType),
	}
	if err := ctx.ShouldBindUri(&urlParams); err != nil {
		p.LogError("err read param = [%v]", err.Error())
		status = http.StatusBadRequest
		goto on_return
	}

	bindata, err = io.ReadAll(ctx.Request.Body)
	if err != nil {
		p.LogError("Error read request body %v", err.Error())
		goto on_return
	}
	request.BinRequest = bindata

	// send data to handler

	p.HandlerRequest(&gBase.Payload{Request: request, ChResult: result})
	// wait for return data
	res = *<-result
on_return:

	if res.Status != 0 {
		status = http.StatusBadRequest
	}
	ctx.Data(status, contenxtType, res.ReplyData)
}
