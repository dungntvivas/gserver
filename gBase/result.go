package gBase

import "gitlab.vivas.vn/go/grpc_api/api"

type PayloadType uint8

const (
	PayloadType_BIN   PayloadType = 0x2
	PayloadType_JSON  PayloadType = 0x4
	PayloadType_PROTO PayloadType = 0x8
	PayloadType_NONE  PayloadType = 0x0
)

type Result struct {
	ReplyData   []byte     // với http chính là api.reply dc convert sẵn từ handler
	Reply       *api.Reply // với protocol còn lại
	Status      int
	ContextType PayloadType
}
