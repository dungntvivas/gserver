package gHTTP

import (
	"io/ioutil"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/gserver/gBase"
	"gitlab.vivas.vn/go/internal/logger"
)

type RequestID struct {
	TYPE  int `uri:"type" binding:"required"`
	GROUP int `uri:"group" binding:"required"`
}

type HTTPServer struct {
	gBase.GServer
	http_sv *http.Server
}

func NewServer(_addr string, _logger *logger.Logger, _done *chan struct{}, _tls gBase.TLS) (*HTTPServer, bool) {
	p := &HTTPServer{
		GServer: gBase.GServer{
			Addr:   _addr,
			Logger: _logger,
			Done:   _done,
			Tls:    _tls,
		},
	}
	if _tls.IsTLS {
		p.ServerName = "HTTPS"
	} else {
		p.ServerName = "HTTP"
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
	return p, true
}

func (p *HTTPServer) Serve() error {
	listen, err := net.Listen("tcp", p.Addr)
	if err != nil {
		return err
	}
	if p.Tls.IsTLS {
		go p.http_sv.ServeTLS(listen, p.Tls.Cert, p.Tls.Key)
		p.LogInfo("Listener opened on %s", p.Addr)
	} else {
		go p.http_sv.Serve(listen)
		p.LogInfo("Listener opened on %s", p.Addr)
	}
	return nil
}

func (p *HTTPServer) onReceiveRequest(ctx *gin.Context) {
	var urlParams RequestID
	status := http.StatusOK
	var res gBase.Result
	result := make(chan gBase.Result)
	contenxtType := ctx.Request.Header.Get("Content-Type")
	ctType := gBase.ContextType_JSON
	var bindata []byte
	var err error
	if contenxtType == "application/octet-stream" {
		ctType = gBase.ContextType_BIN
	}
	request := &api.Request{
		Protocol:    uint32(gBase.RequestFrom_HTTP),
		PayloadType: uint32(ctType),
	}
	if err := ctx.ShouldBindUri(&urlParams); err != nil {
		p.LogError("err read param = [%v]", err.Error())
		status = http.StatusBadRequest
		goto on_return
	}
	bindata, err = ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		p.LogError("Error read request body %v", err.Error())
		goto on_return
	}
	request.BinRequest = bindata

	// send data to handler
	p.HandlerRequest(&result, request)
	// wait for return data
	res = <-result
on_return:

	if res.Status != 0 {
		status = http.StatusBadRequest
	}
	ctx.Data(status, contenxtType, res.ReplyData)
}
