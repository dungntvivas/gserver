package gBase

import (
	"bytes"
	"fmt"
	"io"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/panjf2000/gnet/v2"
	"gitlab.vivas.vn/go/internal/encryption/aes"
	"gitlab.vivas.vn/go/internal/encryption/rsa"
	"gitlab.vivas.vn/go/internal/encryption/xor"
)

type wsMessageBuf struct {
	firstHeader *ws.Header
	curHeader   *ws.Header
	cachedBuf   bytes.Buffer
}
type readWrite struct {
	io.Reader
	io.Writer
}

type ClientConnection struct {
	EncType           Encryption_Type
	PKey              []byte
	Platfrom          int32
	IsSetupConnection bool

	Fd       int  // id của kết nối đối với os ( udp không xác định được fd)
	IsAuthen bool // kết nối này đã được xác thực hay chưa
}
func (p *ClientConnection)isOK() bool{
	return p.IsAuthen && p.IsSetupConnection
}
type ServerConnection struct {
	DecType           Encryption_Type
	PKey              []byte
	Rsa               *rsa.Vras // dc set khi Type = rsa
	IsSetupConnection bool
}

type Connection struct {
	Client *ClientConnection
	Server *ServerConnection

	UpdateAt      uint64 // thời gian cuối cùng có action ở kết nối này /// send , receive
	Session_id    string // session id của kết nối
	User_id       string // user id của kết nối
	Connection_id []byte // id của kết nối xác định xem đang ở máy chủ nào

	// websocket
	upgraded bool         // 链接是否升级
	buf      bytes.Buffer // 从实际socket中读取到的数据缓存
	wsMsgBuf wsMessageBuf // ws 消息缓存
}
func (c *Connection)isOK() bool {
	return len(c.Session_id) != 0 && c.Client.isOK()
}

// websocket upgrade connection
func (w *Connection) upgrade(c gnet.Conn) (ok bool, action gnet.Action) {
	if w.upgraded {
		ok = true
		return
	}
	buf := &w.buf
	tmpReader := bytes.NewReader(buf.Bytes())
	oldLen := tmpReader.Len()

	_, err := ws.Upgrade(readWrite{tmpReader, c})
	skipN := oldLen - tmpReader.Len()
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF { //数据不完整
			return
		}
		buf.Next(skipN)
		//logging.Infof("conn[%v] [err=%v]", c.RemoteAddr().String(), err.Error())
		action = gnet.Close
		return
	}
	buf.Next(skipN)
	if err != nil {
		//logging.Infof("conn[%v] [err=%v]", c.RemoteAddr().String(), err.Error())
		action = gnet.Close
		return
	}
	ok = true
	w.upgraded = true
	return
}
func (w *Connection) readBufferBytes(c gnet.Conn) gnet.Action {
	size := c.InboundBuffered()
	buf := make([]byte, size, size)
	read, err := c.Read(buf)
	if err != nil {
		//logging.Infof("read err! %w", err)
		return gnet.Close
	}
	if read < size {
		//logging.Infof("read bytes len err! size: %d read: %d", size, read)
		return gnet.Close
	}
	w.buf.Write(buf)
	return gnet.None
}
func (w *Connection) wsDecode(c gnet.Conn) (outs []wsutil.Message, err error) {
	messages, err := w.readWsMessages()
	if err != nil {
		//logging.Infof("Error reading message! %v", err)
		return nil, err
	}
	if messages == nil || len(messages) <= 0 { //没有读到完整数据 不处理
		return
	}
	for _, message := range messages {
		if message.OpCode.IsControl() {
			err = wsutil.HandleClientControlMessage(c, message)
			if err != nil {
				return
			}
			continue
		}
		if message.OpCode == ws.OpText || message.OpCode == ws.OpBinary {
			outs = append(outs, message)
		}
	}
	return
}

func (w *Connection) readWsMessages() (messages []wsutil.Message, err error) {
	msgBuf := &w.wsMsgBuf
	in := &w.buf
	for {
		if msgBuf.curHeader == nil {
			if in.Len() < ws.MinHeaderSize { //头长度至少是2
				return
			}
			var head ws.Header
			if in.Len() >= ws.MaxHeaderSize {
				head, err = ws.ReadHeader(in)
				if err != nil {
					return messages, err
				}
			} else { //有可能不完整，构建新的 reader 读取 head 读取成功才实际对 in 进行读操作
				tmpReader := bytes.NewReader(in.Bytes())
				oldLen := tmpReader.Len()
				head, err = ws.ReadHeader(tmpReader)
				skipN := oldLen - tmpReader.Len()
				if err != nil {
					if err == io.EOF || err == io.ErrUnexpectedEOF { //数据不完整
						return messages, nil
					}
					in.Next(skipN)
					return nil, err
				}
				in.Next(skipN)
			}

			msgBuf.curHeader = &head
			err = ws.WriteHeader(&msgBuf.cachedBuf, head)
			if err != nil {
				return nil, err
			}
		}
		dataLen := (int)(msgBuf.curHeader.Length)
		if dataLen > 0 {
			if in.Len() >= dataLen {
				_, err = io.CopyN(&msgBuf.cachedBuf, in, int64(dataLen))
				if err != nil {
					return
				}
			} else { //数据不完整
				fmt.Println(in.Len(), dataLen)
				//logging.Infof("incomplete data")
				return
			}
		}
		if msgBuf.curHeader.Fin { //当前 header 已经是一个完整消息
			messages, err = wsutil.ReadClientMessage(&msgBuf.cachedBuf, messages)
			if err != nil {
				return nil, err
			}
			msgBuf.cachedBuf.Reset()
		} else {
			//logging.Infof("The data is split into multiple frames")
		}
		msgBuf.curHeader = nil
	}
}

func NewConnection(sv *ServerConnection, c *ClientConnection) *Connection {
	p := Connection{
		Server: sv,
		Client: c,
	}
	p.Server.IsSetupConnection = true
	if p.Server.DecType == Encryption_RSA {
		p.Server.Rsa, _ = rsa.VRSA_NEW()
		p.Server.PKey = p.Server.Rsa.GetPublicKey()

	} else if p.Server.DecType == Encryption_AES {
		p.Server.PKey, _ = aes.NEW_AES_KEY()
	} else if p.Server.DecType == Encryption_XOR {
		// pkey = 20 byte
		p.Server.PKey, _ = xor.NEW_XOR_KEY()
	}
	return &p
}
