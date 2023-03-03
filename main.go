package main

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/gserver/gBase"
	"gitlab.vivas.vn/go/gserver/gTCP"
	"gitlab.vivas.vn/go/internal/logger"
	"google.golang.org/protobuf/types/known/anypb"
	"io"
	"net"
	"time"
)

func main()  {

	done := make(chan struct{})
	chReceiveRequest := make(chan *gBase.Payload)
	tcp_option := gBase.DefaultTcpSocketConfigOption
	tcp_option.EncodeType = gBase.Encryption_RSA
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

	time.Sleep(time.Second * 2)

	conn, err := net.Dial("tcp", "localhost:44224")
	if err != nil {
		fmt.Println(err)
		return
	}
	buf := make([]byte,4096)

	hlReceive := api.HelloReceive{}

	//encode := api.EncodeType_NONE
	//lable,_ := xor.NEW_XOR_KEY()

	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				fmt.Println("read error:", err)
			}
			break
		}
		if n != 0 {
			msg, psize, _ := gBase.DecodeHeader(buf[0:n])
			if psize > n {
				break
			}
			msg.Payload = buf[16:n]
			fmt.Printf("Msg Type %v\n", msg.MsgType)
			if msg.MsgType == 100 {
				/// Receive Msg
				receive := api.Receive{}
				msg.ToProtoModel(&receive)
				fmt.Printf("Server time => %v\n", receive.ServerTime)

				receive.Receive.UnmarshalTo(&hlReceive)
				fmt.Printf("Server Encrypt Type => %v\n", hlReceive.ServerEncodeType.String())
				if hlReceive.ServerEncodeType != api.EncodeType_NONE {
					fmt.Printf("Pkey => %v\n", hlReceive.PKey)
				}
				// setup client receive encode
				request := api.Request{
					Type: 101,
					Group: api.Group_CONNECTION,
				}
				request_hello := api.Hello_Request{
					EncodeType: api.EncodeType_XOR,
					Platform: api.Platform_ANDROID,
				}
				_request_hello,_ := anypb.New(&request_hello)
				request.Request = _request_hello
				// api.Request --> binary
				_request , _ := proto.Marshal(&request)
				fmt.Printf("Payload Send %v \n", _request)



				// encode send to server
				newMsg := gBase.NewMessage(_request, uint32(api.Group_CONNECTION),101,[]byte{1,2,3,4,5})
				_p , _ := newMsg.Encode(gBase.Encryption_Type(hlReceive.ServerEncodeType),hlReceive.PKey)
				fmt.Printf("%v\n",_p)
				conn.Write(_p)

			}

		}
	}



	conn.Close()

	/// create tcp client test



	<-done



}
