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
	RequestProtocol_UDS    RequestProtocol = 0x80 // Unix Domain Socket ( linux , macos ) only
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
	//
	connection_id string
}

type Encryption_Type uint8

const (
	Encryption_XOR  Encryption_Type = 0x20
	Encryption_RSA  Encryption_Type = 0x40
	Encryption_AES  Encryption_Type = 0x60
	Encryption_NONE Encryption_Type = 0x0
)

func (s Encryption_Type) String() string {
	if s == Encryption_NONE {
		return "Raw"
	} else if s == Encryption_XOR {
		return "XOR"
	} else if s == Encryption_AES {
		return "AES-CBC-256-PKCS7_PADING"
	} else if s == Encryption_RSA {
		return "RSA-1024-PKCS1_PADING"
	} else {
		return "Unknown"
	}
}


type ConfigOption struct {
	Done       *chan struct{}
	Logger     *logger.Logger
	Addr       string
	Tls        TLS
	ServerName string
	Protocol   RequestProtocol
	EncodeType Encryption_Type
}

var DefaultHttpConfigOption = ConfigOption{
	Addr:       ":44222",
	Tls:        TLS{IsTLS: false},
	Protocol:   RequestProtocol_HTTP,
	ServerName: "HTTP",
	EncodeType:Encryption_NONE,
}
var DefaultGrpcConfigOption = ConfigOption{
	Addr:       ":44223",
	Tls:        TLS{IsTLS: false},
	Protocol:   RequestProtocol_GRPC,
	ServerName: "HTTP",
	EncodeType:Encryption_NONE,
}
var DefaultTcpSocketConfigOption = ConfigOption{
	Addr:       ":44224",
	Tls:        TLS{IsTLS: false},
	Protocol:   RequestProtocol_TCP,
	ServerName: "TCP",
	EncodeType:Encryption_NONE,
}
var DefaultWebSocketConfigOption = ConfigOption{
	Addr:       ":44225",
	Tls:        TLS{IsTLS: false},
	Protocol:   RequestProtocol_WS,
	ServerName: "Websocket",
	EncodeType:Encryption_NONE,
}
var DefaultUdpSocketConfigOption = ConfigOption{
	Addr:       ":44226",
	Tls:        TLS{IsTLS: false},
	Protocol:   RequestProtocol_UDP,
	ServerName: "UDP",
	EncodeType:Encryption_NONE,
}
var DefaultUdsSocketConfigOption = ConfigOption{
	Addr:       "/tmp/uds",
	Tls:        TLS{IsTLS: false},
	Protocol:   RequestProtocol_UDS,
	ServerName: "Unix networks Socket",
	EncodeType:Encryption_NONE,
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
