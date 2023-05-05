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
	RequestProtocol_QUIC   RequestProtocol = 0x81
	RequestProtocol_NONE   RequestProtocol = 0x0
)

func (s RequestProtocol) String() string {
	if s == RequestProtocol_HTTP {
		return "HTTP"
	} else if s == RequestProtocol_GRPC {
		return "GRPC"
	} else if s == RequestProtocol_TCP {
		return "TCP"
	} else if s == RequestProtocol_WS {
		return "WS"
	} else if s == RequestProtocol_UDP {
		return "UDP"
	} else if s == RequestProtocol_UDS {
		return "UDS"
	} else if s == RequestProtocol_QUIC {
		return "QUIC"
	} else {
		return "Unknown"
	}
}

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
	Request *api.Request
	//
	Connection_id string
	IsAuth        bool   /// người dùng này đã authen mark connection với 1 tài khoản nào hay chưa
	Session_id    string // sesion_id mark với tài khoản
	User_id       string // id của người dùng khi đã authen
}
type PayloadPush struct {
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

type Push_Type uint8

const (
	Push_Type_ALL        Push_Type = 0
	Push_Type_USER       Push_Type = 1
	Push_Type_SESSION    Push_Type = 2
	Push_Type_CONNECTION Push_Type = 3
)

func (s Push_Type) Push_Type_to_proto_type() api.PushReceive_PUSH_TYPE {
	if s == Push_Type_ALL {
		return api.PushReceive_TO_ALL
	} else if s == Push_Type_USER {
		return api.PushReceive_TO_USER
	} else if s == Push_Type_SESSION {
		return api.PushReceive_TO_SESSION
	} else if s == Push_Type_CONNECTION {
		return api.PushReceive_TO_CONNECTION
	} else {
		return api.PushReceive_TO_ALL
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
	GW         []string
}

var DefaultHttpConfigOption = ConfigOption{
	Addr:       ":44222",
	Tls:        TLS{IsTLS: false},
	Protocol:   RequestProtocol_HTTP,
	ServerName: "HTTP",
	EncodeType: Encryption_NONE,
}
var DefaultHttpsConfigOption = ConfigOption{
	Addr: ":44422",
	Tls: TLS{
		IsTLS:     true,
		H2_Enable: false,
	},
	Protocol:   RequestProtocol_HTTP,
	ServerName: "HTTPS",
	EncodeType: Encryption_NONE,
}
var DefaultHttp2ConfigOption = ConfigOption{
	Addr: ":44322",
	Tls: TLS{
		IsTLS:     true,
		H2_Enable: true,
	},
	Protocol:   RequestProtocol_HTTP,
	ServerName: "H2",
	EncodeType: Encryption_NONE,
}
var DefaultHttp3QUICConfigOption = ConfigOption{
	Addr: ":44322",
	Tls: TLS{
		IsTLS: true,
	},
	Protocol:   RequestProtocol_QUIC,
	ServerName: "QUIC",
	EncodeType: Encryption_NONE,
}
var DefaultGrpcConfigOption = ConfigOption{
	Addr:       ":44226",
	Tls:        TLS{IsTLS: false},
	Protocol:   RequestProtocol_GRPC,
	ServerName: "HTTP",
	EncodeType: Encryption_NONE,
}
var DefaultTcpSocketConfigOption = ConfigOption{
	Addr:       ":44224",
	Tls:        TLS{IsTLS: false},
	Protocol:   RequestProtocol_TCP,
	ServerName: "TCP",
	EncodeType: Encryption_NONE,
}
var DefaultWebSocketConfigOption = ConfigOption{
	Addr:       ":44223",
	Tls:        TLS{IsTLS: false},
	Protocol:   RequestProtocol_WS,
	ServerName: "Websocket",
	EncodeType: Encryption_NONE,
}
var DefaultUdpSocketConfigOption = ConfigOption{
	Addr:       ":44225",
	Tls:        TLS{IsTLS: false},
	Protocol:   RequestProtocol_UDP,
	ServerName: "UDP",
	EncodeType: Encryption_NONE,
}
var DefaultUdsSocketConfigOption = ConfigOption{
	Addr:       "/tmp/uds",
	Tls:        TLS{IsTLS: false},
	Protocol:   RequestProtocol_UDS,
	ServerName: "Unix Domain Socket",
	EncodeType: Encryption_NONE,
}

type GServer struct {
	Config *ConfigOption
	//out
	ChReceiveRequest chan *Payload
}

func (p *GServer) Close() {

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
