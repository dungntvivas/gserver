package gBase

import (
	"gitlab.vivas.vn/go/internal/encryption/aes"
	"gitlab.vivas.vn/go/internal/encryption/rsa"
	"gitlab.vivas.vn/go/internal/encryption/xor"
)

type ClientConnection struct {
	encType      Encryption_Type
	pKey       []byte
	platfrom   int32
	IsSetupConnection bool
}
type ServerConnection struct {
	DecType      Encryption_Type
	PKey       []byte
	Rsa    *rsa.Vras // dc set khi Type = rsa
	IsSetupConnection bool
}

type Connection struct {

	Client *ClientConnection
	Server *ServerConnection
	IsAuthen bool // kết nối này đã được xác thực hay chưa
	Fd int // id của kết nối đối với os ( udp không xác định được fd)
	IsWebSocketConnection bool // kết nối này là websocket hay không
	UpdateAt   uint64 // thời gian cuối cùng có action ở kết nối này /// send , receive
	Session_id string  // session id của kết nối
	User_id    string  // user id của kết nối
	Connection_id string // id của kết nối xác định xem đang ở máy chủ nào
}
func NewConnection(sv *ServerConnection) *Connection{
	p := Connection{
		Server: sv,
	}
	p.Server.IsSetupConnection = true
	if(p.Server.DecType == Encryption_RSA){
		p.Server.Rsa,_ = rsa.VRSA_NEW()
		p.Server.PKey = p.Server.Rsa.GetPublicKey()

	}else if(p.Server.DecType == Encryption_AES){
		p.Server.PKey,_ = aes.NEW_AES_KEY()
	}else if(p.Server.DecType == Encryption_XOR){
		// pkey = 20 byte
		p.Server.PKey,_ =  xor.NEW_XOR_KEY()
	}
	return &p
}
