# GServer 

là project được xây dựng trên ngôn ngữ go , Template khung để dựng dự án microservice cung cấp những công cụ cần thiết để tạo một máy chủ phục vụ 

## Overview

* Hỗ trợ tạo dựng máy chủ GRPC
* Hỗ trợ tạo dựng máy chủ HTTP
* Hỗ trợ tạo dựng máy chủ TCP
* Hỗ trợ tạo dựng máy chủ WS
* Hỗ trợ tạo dựng máy chủ UDS

## Ví dụ để khởi tạo một máy chủ microservice

```go
package main

import (
	"fmt"
	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/gserver/gBase"
	"gitlab.vivas.vn/go/gserver/gService"
	"gitlab.vivas.vn/go/internal/logger"
	"os"
)

type EchoServer struct {
	logger  *logger.Logger
	service *gService.Service

}
func NewEchoServer() (*EchoServer, bool) {
	_logger, err := logger.New(logger.Level(logger.Info), logger.LogDestinations{logger.DestinationFile: {}, logger.DestinationStdout: {}},"log.log")
	if err != nil {
		fmt.Printf("Error => %v", err.Error())
		return nil, false
	}

	grpc_config := gBase.DefaultGrpcConfigOption
	grpc_config.Addr = ":1234"

	p := EchoServer{
		service: gService.NewService("Echo Server", _logger, grpc_config),
		logger:  _logger,
	}
	p.service.SvName = "Echo Server"
	p.service.SetCallBackRequest(p.HanderRequest)

	return &p,true
}
func (p *EchoServer) run() {
	p.service.Start()
}
func (p *EchoServer) Wait() {
	<-p.service.Done
}
func (p *EchoServer) HanderRequest(request *api.Request, reply *api.Reply) uint32 {
	reply.Status = uint32(api.ResultType_OK)
	reply.Msg = "OK"
	return uint32(api.ResultType_OK)

}

func main()  {
	s,isOK := NewEchoServer()
	if !isOK {
		os.Exit(99)
	}
	s.Wait()

}

```