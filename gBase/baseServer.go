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

type PayloadType uint8

const (
	PayloadType_BIN   PayloadType = 0x2
	PayloadType_JSON  PayloadType = 0x4
	PayloadType_PROTO PayloadType = 0x8
	PayloadType_NONE  PayloadType = 0x0
)

type TLS struct {
	IsTLS     bool
	Cert      string
	Key       string
	H2_Enable bool // sử dụng với server http // grpc default h2
}
type Payload struct {
	ChReply chan *api.Reply
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
var DefaultGrpcConfigOption = ConfigOption{
	Addr:       ":44223",
	Tls:        TLS{IsTLS: false},
	Protocol:   RequestProtocol_GRPC,
	ServerName: "HTTP",
}
var DefaultTcpSocketConfigOption = ConfigOption{
	Addr:       ":44224",
	Tls:        TLS{IsTLS: false},
	Protocol:   RequestProtocol_TCP,
	ServerName: "TCP",
}
var DefaultWebSocketConfigOption = ConfigOption{
	Addr:       ":44225",
	Tls:        TLS{IsTLS: false},
	Protocol:   RequestProtocol_WS,
	ServerName: "Websocket",
}
var DefaultUdpSocketConfigOption = ConfigOption{
	Addr:       ":44225",
	Tls:        TLS{IsTLS: false},
	Protocol:   RequestProtocol_UDP,
	ServerName: "UDP",
}


type GServer struct {
	Config *ConfigOption
	//out
	ChReceiveRequest chan *Payload
}

func (p *GServer) Close(){

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
