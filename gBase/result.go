package gBase

import "gitlab.vivas.vn/go/grpc_api/api"

type ContextType uint8

const (
	ContextType_BIN   ContextType = 0x2
	ContextType_JSON  ContextType = 0x4
	ContextType_PROTO ContextType = 0x8
	ContextType_NONE  ContextType = 0x0
)

type Result struct {
	ReplyData   []byte     // với http chính là api.reply dc convert sẵn từ handler
	Reply       *api.Reply // với protocol còn lại
	Status      int
	ContextType ContextType
}
