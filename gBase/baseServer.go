package gBase

import (
	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/internal/logger"
)

type RequestFrom uint8

const (
	RequestFrom_HTTP   RequestFrom = 0x20
	RequestFrom_SOCKET RequestFrom = 0x40
	RequestFrom_GRPC   RequestFrom = 0x60
	RequestFrom_NONE   RequestFrom = 0x0
)

type TLS struct {
	IsTLS     bool
	Cert      string
	Key       string
	H2_Enable bool // sử dụng với server http // grpc default h2
}
type Payload struct {
	ChResult chan *Result
}

type GServer struct {
	Done        *chan struct{}
	Logger      *logger.Logger
	Addr        string
	Tls         TLS
	ServerName  string
	ChannelData chan *Payload
}

func (p *GServer) LogInfo(format string, args ...interface{}) {
	p.Logger.Log(logger.Info, "["+p.ServerName+"] "+format, args...)
}
func (p *GServer) LogDebug(format string, args ...interface{}) {
	p.Logger.Log(logger.Debug, "["+p.ServerName+"] "+format, args...)
}
func (p *GServer) LogError(format string, args ...interface{}) {
	p.Logger.Log(logger.Error, "["+p.ServerName+"] "+format, args...)
}

func (p *GServer) HandlerRequest(rs chan *Result, payload *api.Request) {
	p.ChannelData <- &Payload{ChResult: rs}
}
