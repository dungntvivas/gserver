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
	var res api.Reply
	result := make(chan *api.Reply)
	defer close(result)
	contenxtType := ctx.Request.Header.Get("Content-Type")
	vAuthorization := ctx.Request.Header.Get("V-Authorization")
	ctType := gBase.PayloadType_JSON
	var bindata []byte
	var err error
	if contenxtType == "application/octet-stream" {
		ctType = gBase.PayloadType_BIN
	}
	request := &api.Request{
		Protocol:    uint32(gBase.RequestProtocol_HTTP),
		PayloadType: uint32(ctType),
	}
	if err := ctx.ShouldBindUri(&urlParams); err != nil {
		p.LogError("err read param = [%v]", err.Error())
		status = http.StatusBadRequest
		goto on_return
	}else{

	}
	request.Type = uint32(urlParams.TYPE)
    request.Group = api.Group(urlParams.GROUP)
	bindata, err = io.ReadAll(ctx.Request.Body)
	if err != nil {
		p.LogError("Error read request body %v", err.Error())
		status = http.StatusBadRequest
		goto on_return
	}
	request.BinRequest = bindata
	request.Session = &api.Session{SessionId: vAuthorization}
	// send data to handler
	p.HandlerRequest(&gBase.Payload{Request: request, ChReply: result})
	// wait for return data
	res = *<-result
on_return:

	if res.Status != 0 {
		if(res.Status == uint32(api.ResultType_INTERNAL_SERVER_ERROR)){
			status = http.StatusInternalServerError
		}else if(res.Status == uint32(api.ResultType_SESSION_EXPIRE)){
			status = http.StatusForbidden
		}else if(res.Status == uint32(api.ResultType_SESSION_INVALID)){
			status = http.StatusForbidden
		}
	}
	ctx.Data(status, contenxtType, res.BinReply)
}
