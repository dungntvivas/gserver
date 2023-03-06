package gBase

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/internal/encryption/aes"
	"gitlab.vivas.vn/go/internal/encryption/rsa"
	"gitlab.vivas.vn/go/internal/encryption/xor"
	"gitlab.vivas.vn/go/internal/logger"
	"google.golang.org/protobuf/proto"
)

//   16 Bytes Header
// * 0       3        6       7       8       10             15                                   size
// * +-------+--------+-------+-------+-------+--------------+------------------------------------+
// * | magic |  size  | Group |  Type |  Flag |   MessageID  |   (rsa Lable nếu có ) Body bytes   |
// * +-------+--------+-------+-------+-------+--------------+------------------------------------+
// * Magic 4 Byte 0x82 0x68 0x80 0x65
// * Size 3 byte size của toàn bộ package
// * Group 1 Byte group của package
// * Type  1 Byte Type của package
// * Flag 2 Byte cờ package
// * MsgID 5 byte id của package
// * Body Data
//  FLAG format 2 byte 16 bit
// * 0        1        3                14            15(bit)
// * +--------+--------+----------------+-------------+
// * | encode |  Type  |  size RSA Data |  has Msg ID |
// * +--------+--------+----------------+-------------+
// * 1  bit nếu có encode Payload set = 1 ngươc lại set = 0
// * 2  bit type cua encode
// * 11 bit size cua rsa data ( nếu type = rsa encode )
// * 1  bit nếu có id 8 byte phía sau thì bit 15 được set = 1

// Nếu được encode Payload bằng RSA
// tiếu chuẩn sử dụng RSA_PKCS1_2048
// b1. (1 Key  20 byte được random ngẫu nhiên) xor với Payload => tạo ra 1 Payload mới
// b2. Key ngẫu nhiên ở bước 1 được RSA với public key ( public key này của client gửi lên ở bước setup connection ) ==> tạo ra 1 KEY RSA
// b3. xor Payload với key được tạo ra ở bước 1
// b3. đóng gói KEY RSA ở bước 2 + Payload được xor ở bước 3 , gửi về client

// Nếu được encode Payload bằng XOR
// b1. Payload được xor với key được client gửi lên ở bước setup connection => tạo ra 1 (msg Payload xor )


// Nếu được encode Payload bằng AES
// tiêu chuẩn được sử dụng AES-CBC-256-PKCS7_PADING
// b1 . random 1 iv key
// b2 . encode Payload sử dụng key của client setup lúc tạo kết nối + iv mới dc ramdom => tạo ra 1 msg Payload aes
// b3 . đóng gói iv key này + Payload được aes ở bước 2 , gửi về client



type SocketMessage struct {
	Payload                []byte
	MsgType                uint32
	MsgGroup               uint32
	MSG_ID                 []byte
	Fd                     int
	MSG_encode_decode_type Encryption_Type
	Lable                  []byte // dùng để giải mã,xor key với XOR , là  iv_key với AES , lable_rsa với RSA
	Conn *Connection
}
func (m *SocketMessage)ToProtoModel(src proto.Message) error{
	m.DecodePayloadIfNeed()
	return proto.Unmarshal(m.Payload, src)
}
func (m *SocketMessage)ToRequestProtoModel(src proto.Message) error{
	request := api.Request{}
	if err := m.ToProtoModel(&request);err != nil {
		return err
	}
	if err := request.Request.UnmarshalTo(src);err != nil{
		return err
	}
	return nil
}
func NewMessage(_payload []byte, _group uint32,_type uint32, _id []byte) *SocketMessage {
	p := &SocketMessage{
		Payload:  _payload,
		MsgGroup: _group,
		MsgType:  _type,
		MSG_ID:   _id,

	}
	return p
}
// 0,1,2,3 magic 4
// 4,5,6 size 3
// 7 group 1
// 8 type 1
// 9,10 flag 2
// 11,12,13,14,15 msg id 5
// 16 -> size Payload

// DecodeHeader chuyển data byte về dữ liệu SocketMessage ,MsgSize , Lable size
func DecodeHeader(header []byte,n int) (*SocketMessage,int){
	header_size := 0x10
	if(len(header) < header_size){
		fmt.Printf("%v", "Header size Invalid \n")
		return nil,0
	}
	if header[0] != 0x82 || header[1] != 0x68 || header[2] != 0x80 || header[3] != 0x65 { // RDPA
		fmt.Printf("%v", "Error Header \n")
		return nil,0
	}
	msgSize := (int(header[4]) << 0x10) + (int(header[5]) << 0x8) + int(header[6])
	lable_size := 0
	_socketMSG := SocketMessage{}
	/// GROUP
	_socketMSG.MsgGroup = uint32(header[7])
	///
	_socketMSG.MsgType = uint32(header[8])
	/// MSG ID
	_socketMSG.MSG_ID = header[0xB:header_size]
	_hasEnc := (header[9] & 0x80) == 0x80
	if _hasEnc {
		_enc_type := uint8(header[9] & 0x60)
		_socketMSG.MSG_encode_decode_type = Encryption_Type(_enc_type)
		if (_socketMSG.MSG_encode_decode_type == Encryption_AES || _socketMSG.MSG_encode_decode_type == Encryption_RSA){
			// get iv_size from header
			lable_size = int(header[9]&0x1F) << 6
			lable_size = lable_size | (int(header[10]&0xFC) >> 2)
			_socketMSG.Payload = header[header_size+lable_size:]
			_socketMSG.Lable = header[header_size:header_size+lable_size]
		}else{
			_socketMSG.Payload = header[header_size:n]
		}
	}else{
		_socketMSG.Payload = header[header_size:n]
	}
	return &_socketMSG,msgSize
}

func DecodePacket(log *logger.Logger,c gnet.Conn) []*SocketMessage {
	header_size := 0x10
	models := []*SocketMessage{}

	/// Check client
	conn := c.Context().(*Connection)
	if (conn == nil){
		log.Log(logger.Info,"Connection Null")
		return nil
	}
	loop:
		for{
			if c.InboundBuffered() < header_size { // tối thiểu 16 byte header
				break loop
			}
			header, err := c.Peek(header_size) // lấy ra 16 byte header
			if err != nil {
				c.Peek(-1) // xóa buffer
				break loop
			}
			if header[0] != 0x82 || header[1] != 0x68 || header[2] != 0x80 || header[3] != 0x65 { // RDPA
				c.Discard(-1) // clear buffer
				log.Log(logger.Info,"Magic byte not math")
				break loop
			}
			msgSize := (int(header[4]) << 0x10) + (int(header[5]) << 0x8) + int(header[6])
			if msgSize > c.InboundBuffered() {
				log.Log(logger.Info,"Buffer size truncate")
				break loop
			}
			c.Discard(header_size) // xóa header ra khỏi buffer
			payload_size := msgSize - header_size
			raw_payload,err := c.Peek(payload_size) /// read Payload content
			if err != nil {
				c.Peek(-1) // xóa buffer
				break loop
			}
			c.Discard(payload_size)

			_socketMSG := SocketMessage{
				Fd: c.Fd(),
			}
			if ct := c.Context().(*Connection);ct != nil{
				_socketMSG.Conn = ct
			}
			/// GROUP
			_socketMSG.MsgGroup = uint32(header[7])
			///
			_socketMSG.MsgType = uint32(header[8])
			/// MSG ID
			_socketMSG.MSG_ID = header[11:16]
			_hasEnc := (header[9] & 0x80) == 0x80
			if _hasEnc {
				if conn.Server.IsSetupConnection == false {
					/// client chưa setup encode
					log.Log(logger.Info,"server Decode chưa được cài đặt")
					c.Peek(-1) // xóa buffer
					break loop
				}
				_enc_type := uint8(header[9] & 0x60)
				if(conn.Server.DecType != Encryption_Type(_enc_type)){
					log.Log(logger.Info,"Decode không khớp với setup decode của server")
					c.Peek(-1) // xóa buffer
					break loop
				}
				_socketMSG.MSG_encode_decode_type = Encryption_Type(_enc_type)
				if (conn.Server.DecType == Encryption_AES){
					// get iv_size from header
					_iv_size := int(header[9]&0x1F) << 6
					_iv_size = _iv_size | (int(header[10]&0xFC) >> 2)
					_iv_key := raw_payload[0:_iv_size]
					_socketMSG.Lable = _iv_key
					_socketMSG.Payload = raw_payload[_iv_size:]

				}else if(conn.Server.DecType == Encryption_RSA){ /// decode với private key của server
					_raa_key_size := int(header[9]&0x1F) << 6
					_raa_key_size = _raa_key_size | (int(header[10]&0xFC) >> 2)
					_rsa_key := raw_payload[0:_raa_key_size]
					_socketMSG.Lable = _rsa_key
					_socketMSG.Payload = raw_payload[_raa_key_size:]

				}else if conn.Server.DecType == Encryption_XOR{ /// với xor ko có Lable đi kèm
					_socketMSG.Lable = conn.Server.PKey
					_socketMSG.Payload = raw_payload
				}else{
					_socketMSG.Payload = raw_payload
				}

			}else{
				if _socketMSG.MsgGroup != uint32(api.Group_CONNECTION) && conn.Server.IsSetupConnection == false{
					log.Log(logger.Info,"Kết nối chưa xác lập encode-decode")
					c.Peek(-1) // xóa buffer
					break loop
				}
				_socketMSG.Payload = raw_payload
			}
			models = append(models, &_socketMSG)
		}

	return models
}

func (p *SocketMessage) DecodePayloadIfNeed() {
	if(p.MSG_encode_decode_type != Encryption_NONE){
		if(p.MSG_encode_decode_type == Encryption_XOR){ /// XOR -> RAW
			p.Payload = xor.EncryptDecrypt(p.Payload,p.Lable)
		}else if(p.MSG_encode_decode_type == Encryption_RSA){
			xor_key ,_ := p.Conn.Server.Rsa.RSA_PKCS1_Decrypt(p.Lable)
			p.Payload = xor.EncryptDecrypt(p.Payload,xor_key)
		}else if(p.MSG_encode_decode_type == Encryption_AES){
			p.Payload, _ = aes.CBCDecrypter(p.Payload, p.Conn.Server.PKey, p.Lable)
		}
		p.MSG_encode_decode_type = Encryption_NONE
	}

}



func (p *SocketMessage) Encode(encodeType Encryption_Type, pKey []byte) ([]byte, error) { // sử dụng public key của client để encode Payload

	out_buf := []byte{0x82 ,0x68 ,0x80 ,0x65 ,0x0, 0x0, 0x0, 0x0, 0x0, 0x0,0x0}
	var msg_id_size = 5

	size := len(out_buf)
	/// GROUP
	out_buf[7] = byte(p.MsgGroup & 0xFF)
	/// TYPE
	out_buf[8] = byte(p.MsgType & 0xFF)

	/// FLAG MSG ID
	if len(p.MSG_ID) != 0 { // no id
		size += msg_id_size
		out_buf[10] = out_buf[10] | 0x2
		out_buf = append(out_buf, p.MSG_ID...)
	}

	/// FLAG - ENCODE , ENCODE TYPE
	if encodeType != Encryption_NONE {
		out_buf[9] = out_buf[9] | 0x80 // 1 bit
		/// FLAG ENCODE TYPE
		out_buf[9] = out_buf[9] | byte(encodeType) // 2 bit tiếp theo
	}
	if encodeType == Encryption_NONE { // Payload raw
		size += len(p.Payload)
		out_buf = append(out_buf, p.Payload...)
	}else if encodeType == Encryption_XOR { // xor Payload
		size += len(p.Payload)
		out_buf = append(out_buf, xor.EncryptDecrypt(p.Payload, pKey)...)
	}else if encodeType == Encryption_AES {
		iv, err := aes.IV_RANDOM()
		if err != nil {
			return nil, err
		}
		payload_aes, err := aes.CBCEncrypterWithClientKey(pKey, iv, p.Payload)
		if err != nil {
			return nil ,err
		}
		size += len(iv)
		size += len(payload_aes)
		iv_size := len(iv)
		out_buf[10] = out_buf[10] | byte((iv_size&0x3F)<<2) // 6 bit đầu của byte 7
		out_buf[9] = out_buf[9] | byte((iv_size>>6)&0x1F) // 5 bit cuối của byte 6
		out_buf = append(out_buf, iv...)
		out_buf = append(out_buf, payload_aes...)
	}else if encodeType == Encryption_RSA { // encode với public key của client
		new_uid := uuid.New()
		lable := []byte{82, 68, 80, 65}
		lable = append(lable, new_uid[:]...)
		// 20 byte
		xor_payload := xor.EncryptDecrypt(p.Payload, lable)
		_public_key, err_pubkey := rsa.BytesToPublicKey(pKey)
		if err_pubkey != nil {
			return nil, err_pubkey
		}
		lable_rsa, err := rsa.RSA_PKCS1_Encrypt(lable, _public_key)
		if err != nil {
			return nil, err
		}
		size += len(lable_rsa)
		size += len(xor_payload)
		// size Lable
		lable_size := len(lable_rsa)
		out_buf[10] = out_buf[10] | byte((lable_size&0x3F)<<2) // 6 bit đầu của byte 7
		out_buf[9] = out_buf[9] | byte((lable_size>>6)&0x1F) // 5 bit cuối của byte 6

		out_buf = append(out_buf, lable_rsa...)
		out_buf = append(out_buf, xor_payload...)
	}



	// SET SIZE 3 byte
	out_buf[4] = byte((size >> 0x10) & 0xFF)
	out_buf[5] = byte((size >> 0x8) & 0xFF)
	out_buf[6] = byte(size & 0xFF)
	return out_buf, nil

}
