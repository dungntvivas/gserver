package gBase

import (
	"gitlab.vivas.vn/go/grpc_api/api"
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
