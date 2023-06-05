# GServer 

là project được xây dựng trên ngôn ngữ go ,
là Template khung để dựng dự án microservice cung cấp những công cụ cần thiết để tạo một máy chủ đa mục đích
và một template thống nhất để handler request , push trên một code duy nhất mà hỗ trợ tất cả loại kết nối phổ biến


## Overview

- [x] Hỗ trợ tạo dựng máy chủ GRPC 
- [x] Hỗ trợ GRPC Client 
- [x] Hỗ trợ tạo dựng máy chủ HTTP/HTTPS ( payload json (application/json) hoặc bin protobuf (application/octet-stream) )
- [x] Hỗ trợ tạo dựng máy chủ HTTP/2 ( payload json (application/json) hoặc bin protobuf (application/octet-stream) )
- [ ] Hỗ trợ tạo dựng máy chủ QUIC/3 => inprogress
- [x] Hỗ trợ tạo dựng máy chủ TCP
- [ ] Hỗ trợ tạo dựng máy chủ TLS (payload ở loại kết nối này đã được mã hóa nên không cần thiết lập mã hóa kết nối) => inprogress
- [x] Hỗ trợ tạo dựng máy chủ WS (payload json (opCode=text) hoặc bin protobuf (opCode = binary) )
- [x] Hỗ trợ tạo dựng máy chủ WSS 
- [x] Hỗ trợ tạo dựng máy chủ UDS (Unix domain socket)
- [x] Hỗ trợ tạo dựng máy chủ UDP 
- [x] Hỗ trợ tạo dựng máy chủ DTLS (pre-shared key = 0x86, 0x73, 0x86, 0x65, 0x83 , CipherSuites = TLS_PSK_WITH_AES_256_CCM_8)

## Ví dụ để khởi tạo một máy chủ microservice grpc 

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
	go p.run()
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

## Ví dụ để khởi tạo một máy chủ http/2
```go
    done := make(chan struct{})
	chReceiveRequest := make(chan *gBase.Payload)
	_logger, _ := logger.New(logger.Info, logger.LogDestinations{logger.DestinationFile: {}, logger.DestinationStdout: {}}, "/tmp/server.log")
	cf := gBase.DefaultHttp2ConfigOption
	cf.Logger = _logger
	cf.Tls.Cert = "./certificate.pem"
	cf.Tls.Key = "./private.key"
	cf.Done = &done
	hsv2 := gHTTP.New(cf,chReceiveRequest)
	hsv2.Serve()
```
## Ví dụ để khởi tạo một máy chủ Socket (tcp,udp,ws)
```go
    done := make(chan struct{})
	chReceiveRequest := make(chan *gBase.Payload)
	_logger, _ := logger.New(logger.Info, logger.LogDestinations{logger.DestinationFile: {}, logger.DestinationStdout: {}}, "/tmp/server.log")
	cf := gBase.DefaultDTLSSocketConfigOption
	cf.Logger = _logger
	cf.Tls.Cert = "./certificate.pem"
	cf.Tls.Key = "./private.key"
	cf.Done = &done
	sv := gDTLS.New(cf,chReceiveRequest)
    sv.Serve()
```
