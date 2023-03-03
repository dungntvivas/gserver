package gBase

import (
	"gitlab.vivas.vn/go/internal/encryption/aes"
	"gitlab.vivas.vn/go/internal/encryption/rsa"
	"gitlab.vivas.vn/go/internal/encryption/xor"
)

type ClientConnection struct {
	EncType      Encryption_Type
	PKey       []byte
	Platfrom   int32
	IsSetupConnection bool
	Fd int // id của kết nối đối với os ( udp không xác định được fd)
	IsAuthen bool // kết nối này đã được xác thực hay chưa
	IsWebSocketConnection bool // kết nối này là websocket hay không
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
	UpdateAt   uint64 // thời gian cuối cùng có action ở kết nối này /// send , receive
	Session_id string  // session id của kết nối
	User_id    string  // user id của kết nối
	Connection_id []byte // id của kết nối xác định xem đang ở máy chủ nào
}
func NewConnection(sv *ServerConnection,c *ClientConnection) *Connection{
	p := Connection{
		Server: sv,
		Client: c,
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
