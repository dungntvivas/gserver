package gHTTP

import (
	"github.com/gin-gonic/gin"
	"gitlab.vivas.vn/go/gserver/gBase"
	"gitlab.vivas.vn/go/internal/logger"
	"net"
	"net/http"
	"time"
)


type RequestID struct {
	TYPE  int `uri:"type" binding:"required"`
	GROUP int `uri:"group" binding:"required"`
}

type HTTPServer struct {
    gBase.GServer
	http_sv *http.Server
}

func (p *HTTPServer) LogInfo(format string, args ...interface{}) {
	p.Logger.Log(logger.Info, "[HTTP] "+format, args...)
}
func (p *HTTPServer) LogDebug(format string, args ...interface{}) {
	p.Logger.Log(logger.Debug, "[HTTP] "+format, args...)
}
func (p *HTTPServer) LogError(format string, args ...interface{}) {
	p.Logger.Log(logger.Error, "[HTTP] "+format, args...)
}

func NewHttpServer(_addr string, _logger *logger.Logger, _done *chan struct{}, _tls gBase.TLS) (*HTTPServer, bool) {
	p := &HTTPServer{
		GServer: gBase.GServer{
			Addr: _addr,
			Logger: _logger,
			Done: _done,
			Tls: _tls,
		},
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
func (p *HTTPServer) Start() error {
	listen, err := net.Listen("tcp", p.Addr)
	if err != nil {
		return err
	}
    if(p.Tls.IsTLS){
		go p.http_sv.ServeTLS(listen, p.Tls.Cert, p.Tls.Key)
		p.LogInfo("Listener opened on %s", p.Addr)
	}else{
		go p.http_sv.Serve(listen)
		p.LogInfo("Listener opened on %s", p.Addr)
	}
	return nil
}

func (p *HTTPServer) onReceiveRequest(ctx *gin.Context) {
	var urlParams RequestID
	status := http.StatusOK
	contenxtType := ctx.Request.Header.Get("Content-Type")
	if err := ctx.ShouldBindUri(&urlParams); err != nil {
		p.LogError("err read param = [%v]", err.Error())
		status = http.StatusBadRequest

	}
	result := make(chan gBase.Result)
	go func() {
		time.Sleep(5 * time.Second)
		result <- gBase.Result{Status: 1,}
	}()
	_v := <- result
    if(_v.Status != 0){
    	status = http.StatusBadRequest
	}
	ctx.Data(status, contenxtType,_v.Data)
}


