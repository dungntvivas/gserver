package gBase

import (
	"gitlab.vivas.vn/go/grpc_api/api"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"time"
)

func NewReceiveMsg(msgType int, payloadReceive *anypb.Any) *api.Receive {
	receive := &api.Receive{
		ServerTime: uint64(time.Now().Unix()),
		Type:       uint32(msgType),
		Receive:    payloadReceive,
	}
	return receive
}
func NewHelloReceive(pKey []byte, encode int) (*anypb.Any, error) {
	hb := api.HelloReceive{
		ServerTime:       uint64(time.Now().Unix()),
		PKey:             pKey,
		ServerEncodeType: api.EncodeType(encode),
	}
	return anypb.New(&hb)
}

func NewRequest(requestType uint32,group api.Group) *api.Request{
	request := api.Request{
		Type: requestType,
		Group: group,
	}
	return &request
}
func NewReply(status uint32) *api.Reply{
	return &api.Reply{Status: status}
}

func PackRequest(request *api.Request,src proto.Message) error{
	_rq , err := anypb.New(src)
	if err != nil {
		return err
	}
	request.Request = _rq
	return nil
}
func PackReply(reply *api.Reply,src proto.Message) error{
	_rq , err := anypb.New(src)
	if err != nil {
		return err
	}
	reply.Reply = _rq
	return nil
}
func MsgToByte(src proto.Message) ([]byte, error){
	return proto.Marshal(src)
}

func GetReceiveBuffer(msgType uint32,msgGroup uint32,encodeType Encryption_Type,pKey []byte,receive []byte) ([]byte,error){
	_msg := SocketMessage{
		Payload: receive,
		MsgType: msgType,
		MsgGroup: msgGroup,
		MSG_ID: []byte{0x86,0x73,0x86,0x65,0x83},
	}
	/// MSG SOCKET ENCODE
	_buf, err := _msg.Encode(encodeType, pKey)
	if err != nil {
		return  nil,err
	}
	// RETURN
	return _buf,nil
}

func GetReplyBuffer(msgType uint32,msgGroup uint32,msgID []byte,src *api.Reply,encodeType Encryption_Type,pKey []byte) ([]byte,error){
	reply := NewReply(src.Status)
	reply.Msg = src.Msg
	reply.Type = msgType
	reply.Group = api.Group(msgGroup)

	if (src.Reply != nil){
		reply.Reply = src.Reply
	}
	_rep_buf,err := MsgToByte(reply)
	if err != nil {
		return nil,err
	}
	//Socket MSG
	_msg := SocketMessage{
		Payload: _rep_buf,
		MsgType: msgType,
		MsgGroup: msgGroup,
		MSG_ID: msgID,
	}
	/// MSG SOCKET ENCODE
	_buf, err := _msg.Encode(encodeType, pKey)
	if err != nil {
		return  nil,err
	}
	// RETURN
	return _buf,nil
}


