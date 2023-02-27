package gBase

import (
	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/internal/logger"
)

type RequestProtocol uint8

const (
	RequestProtocol_HTTP   RequestProtocol = 0x2
	RequestProtocol_SOCKET RequestProtocol = 0x4
	RequestProtocol_GRPC   RequestProtocol = 0x8
	RequestProtocol_TCP    RequestProtocol = 0x10
	RequestProtocol_WS     RequestProtocol = 0x20
	RequestProtocol_UDP    RequestProtocol = 0x40
	RequestProtocol_NONE   RequestProtocol = 0x0
)

type TLS struct {
	IsTLS     bool
	Cert      string
	Key       string
	H2_Enable bool // sử dụng với server http // grpc default h2
}
type Payload struct {
	ChResult chan *Result
	Request  *api.Request
}

type ConfigOption struct {
	Done       *chan struct{}
	Logger     *logger.Logger
	Addr       string
	Tls        TLS
	ServerName string
	Protocol   RequestProtocol
}

var DefaultHttpConfigOption = ConfigOption{
	Addr:       ":44222",
	Tls:        TLS{IsTLS: false},
	Protocol:   RequestProtocol_HTTP,
	ServerName: "HTTP",
}

type GServer struct {
	Config *ConfigOption
	//out
	ChReceiveRequest chan *Payload
}

func NewGServer(config *ConfigOption, chReceiveRequest chan *Payload) *GServer {
	p := &GServer{
		Config: config,
	}
	return p
}

func (p *GServer) LogInfo(format string, args ...interface{}) {
	p.Config.Logger.Log(logger.Info, "["+p.Config.ServerName+"] "+format, args...)
}
func (p *GServer) LogDebug(format string, args ...interface{}) {
	p.Config.Logger.Log(logger.Debug, "["+p.Config.ServerName+"] "+format, args...)
}
func (p *GServer) LogError(format string, args ...interface{}) {
	p.Config.Logger.Log(logger.Error, "["+p.Config.ServerName+"] "+format, args...)
}

func (p *GServer) HandlerRequest(payload *Payload) {
	/// PUSH TO Handler
	p.ChReceiveRequest <- payload
}
