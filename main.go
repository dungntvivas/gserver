package main

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/internal/encryption/aes"
	"gitlab.vivas.vn/go/internal/encryption/rsa"
	"gitlab.vivas.vn/go/internal/encryption/xor"
	"google.golang.org/protobuf/types/known/anypb"
	"io"
	"net"
	"time"
	"gitlab.vivas.vn/go/gserver/gBase"
	"gitlab.vivas.vn/go/gserver/gTCP"
	"gitlab.vivas.vn/go/internal/logger"

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

	encode := api.EncodeType_AES
	_rsa, _ := rsa.VRSA_NEW()
	_ = _rsa.GetPublicKey()
	xor_lable,_ := xor.NEW_XOR_KEY()

	aes_key,_ := aes.NEW_AES_KEY()

	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				fmt.Println("read error:", err)
			}
			break
		}
		if n != 0 {
			fmt.Printf("Receive %d byte\n", n)
			msg, psize := gBase.DecodeHeader(buf[0:n],n)
			if(msg.MSG_encode_decode_type == gBase.Encryption_RSA){
				msg.Conn = &gBase.Connection{
					Server: &gBase.ServerConnection{
						Rsa: _rsa,
					},
				}
			}else if(msg.MSG_encode_decode_type == gBase.Encryption_AES){
				msg.Conn = &gBase.Connection{
					Server: &gBase.ServerConnection{
						PKey: aes_key,
					},
				}
			}else if(msg.MSG_encode_decode_type == gBase.Encryption_XOR){
				msg.Lable = xor_lable
			}
			fmt.Printf("Message encode Type %s\n", msg.MSG_encode_decode_type.String())
			if psize > n {
				break
			}

			fmt.Printf("Msg Type %v\n", msg.MsgType)
			if msg.MsgType == uint32(api.TYPE_ID_RECEIVE_HELLO) {
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
					Type: uint32(api.TYPE_ID_REQUEST_HELLO),
					Group: api.Group_CONNECTION,
				}
				request_hello := api.Hello_Request{
					EncodeType: encode,
					PKey: aes_key,
					Platform: api.Platform_OTHER,
				}
				_request_hello,_ := anypb.New(&request_hello)
				request.Request = _request_hello
				// api.Request --> binary
				_request , _ := proto.Marshal(&request)
				fmt.Printf("Payload Send %v \n", _request)

				// encode send to server
				newMsg := gBase.NewMessage(_request, uint32(api.Group_CONNECTION),request.Type,[]byte{1,2,3,4,5})
				_p , _ := newMsg.Encode(gBase.Encryption_Type(hlReceive.ServerEncodeType),hlReceive.PKey)
				fmt.Printf("%v\n",_p)
				conn.Write(_p)
			}else if msg.MsgType == uint32(api.TYPE_ID_REQUEST_HELLO) {
				reply := api.Reply{}
				//msg.Lable = lable
				msg.ToProtoModel(&reply)
				fmt.Printf("%s\n", "Client Decode Msg ")
				hlReply := api.Hello_Reply{}
				reply.Reply.UnmarshalTo(&hlReply)
				fmt.Printf("Connection ID %v\n", hlReply.ConnectionId)

			}

		}
	}



	conn.Close()




	<-done



}
