package main

import (
	"gitlab.vivas.vn/go/gserver/gBase"
	"gitlab.vivas.vn/go/gserver/gTCP"
	"gitlab.vivas.vn/go/internal/logger"
)




func main()  {

	done := make(chan struct{})
	chReceiveRequest := make(chan *gBase.Payload)
	tcp_option := gBase.DefaultTcpSocketConfigOption
	_logger, _ := logger.New(logger.Info, logger.LogDestinations{logger.DestinationFile: {}, logger.DestinationStdout: {}}, "/tmp/server.log")
	tcp_option.Logger = _logger
	go func() {
		tcp := gTCP.New(tcp_option,chReceiveRequest)
		tcp.Serve()
	}()
	
	
	for n:=0;n<1;n++{
		go func() {
			loop:
				for{
					select {
					case <-chReceiveRequest:
						_logger.Log(logger.Info,"Receive Request")
					case <-done:
						break loop
					}
				}
		}()
	}
	<-done



}
