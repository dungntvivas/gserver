package main

import (
	"fmt"
	"io"
	"net"
	"time"

	"gitlab.vivas.vn/go/grpc_api/api"
	"gitlab.vivas.vn/go/gserver/gBase"
	"gitlab.vivas.vn/go/internal/encryption/aes"
	"gitlab.vivas.vn/go/internal/encryption/rsa"
	"gitlab.vivas.vn/go/internal/encryption/xor"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func main() {

	done := make(chan struct{})
	//chReceiveRequest := make(chan *gBase.Payload)
	//_logger, _ := logger.New(logger.Info, logger.LogDestinations{logger.DestinationFile: {}, logger.DestinationStdout: {}}, "/tmp/server.log")
	//
	//tcp_option := gBase.DefaultTcpSocketConfigOption
	//tcp_option.EncodeType = gBase.Encryption_XOR
	//tcp_option.Logger = _logger
	//udp_option := gBase.DefaultUdpSocketConfigOption
	//udp_option.EncodeType = gBase.Encryption_AES
	//udp_option.Logger = _logger
	//
	//uds_option := gBase.DefaultUdsSocketConfigOption
	//uds_option.EncodeType = gBase.Encryption_RSA
	//uds_option.Logger = _logger
	//
	//ws_option := gBase.DefaultWebSocketConfigOption
	//ws_option.EncodeType = gBase.Encryption_NONE
	//ws_option.Logger = _logger
	//
	//go func() {
	//	tcp := gTCP.New(tcp_option, chReceiveRequest)
	//	tcp.Serve()
	//}()
	//go func() {
	//	udp := gUDP.New(udp_option, chReceiveRequest)
	//	udp.Serve()
	//}()
	//go func() {
	//	uds := gUDS.New(uds_option, chReceiveRequest)
	//	uds.Serve()
	//}()
	//go func() {
	//	ws := gWebsocket.New(ws_option, chReceiveRequest)
	//	ws.Serve()
	//}()
	//
	//for n := 0; n < 1; n++ {
	//	go func() {
	//	loop:
	//		for {
	//			select {
	//			case p := <-chReceiveRequest:
	//				_logger.Log(logger.Info, "Receive Request")
	//				if p.Request.PayloadType == uint32(gBase.PayloadType_BIN) {
	//					fmt.Printf("%v\n", p)
	//					var rq api.Request
	//
	//					proto.Unmarshal(p.Request.BinRequest, &rq)
	//
	//					fmt.Printf("%v\n", rq)
	//
	//					var rqHl api.Hello_Request
	//
	//					rq.Request.UnmarshalTo(&rqHl)
	//
	//					fmt.Printf("%v\n", rqHl.Platform)
	//
	//				}
	//			case <-done:
	//				break loop
	//			}
	//		}
	//	}()
	//}
	//
	//time.Sleep(time.Second * 2)
	conn, err := net.Dial("tcp", "10.3.3.119:8083")
	if err != nil {
		fmt.Println(err)
		return
	}
	buf := make([]byte, 4096)

	hlReceive := api.HelloReceive{}

	encode := api.EncodeType_AES
	_rsa, _ := rsa.VRSA_NEW()
	_ = _rsa.GetPublicKey()
	xor_lable, _ := xor.NEW_XOR_KEY()
	hlReply := api.Hello_Reply{}
	aes_key, _ := aes.NEW_AES_KEY()

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
			msg := gBase.DecodeData(buf[0:n], n)
			if msg.MSG_encode_decode_type == gBase.Encryption_RSA {
				msg.Conn = &gBase.Connection{
					Server: &gBase.ServerConnection{
						Rsa: _rsa,
					},
				}
			} else if msg.MSG_encode_decode_type == gBase.Encryption_AES {
				msg.Conn = &gBase.Connection{
					Server: &gBase.ServerConnection{
						PKey: aes_key,
					},
				}
			} else if msg.MSG_encode_decode_type == gBase.Encryption_XOR {
				msg.Lable = xor_lable
			}

			fmt.Printf("Msg Type %v group %v\n", msg.MsgType, msg.MsgGroup)
			if msg.MsgType == uint32(api.TYPE_ID_RECEIVE_HELLO) && msg.MsgGroup == 0 {
				/// Receive Msg
				receive := api.Receive{}
				msg.ToProtoModel(&receive)
				fmt.Printf("Server time => %v\n", receive.ServerTime)

				receive.Receive.UnmarshalTo(&hlReceive)
				fmt.Printf("Server Encrypt Type => %v\n", hlReceive.ServerEncodeType.String())

				// setup client receive encode
				request := api.Request{
					Type:  uint32(api.TYPE_ID_REQUEST_HELLO),
					Group: api.Group_CONNECTION,
				}
				request_hello := api.Hello_Request{
					EncodeType: encode,
					PKey:       aes_key,
					Platform:   api.Platform_OTHER,
				}
				_request_hello, _ := anypb.New(&request_hello)
				request.Request = _request_hello
				// api.Request --> binary
				_request, _ := proto.Marshal(&request)

				// encode send to server
				newMsg := gBase.NewMessage(_request, uint32(api.Group_CONNECTION), request.Type, []byte{1, 2, 3, 4, 5})
				_p, _ := newMsg.Encode(gBase.Encryption_Type(hlReceive.ServerEncodeType), hlReceive.PKey)

				fmt.Printf("%v", _p)
				conn.Write(_p)
			} else if msg.MsgType == uint32(api.TYPE_ID_REQUEST_HELLO) && msg.MsgGroup == uint32(api.Group_CONNECTION) {
				reply := api.Reply{}
				//msg.Lable = lable
				msg.ToProtoModel(&reply)

				reply.Reply.UnmarshalTo(&hlReply)

				if reply.Status == 0 {
					/// send login
					rq := gBase.NewRequest(uint32(api.AUTHEN_TYPE_ID_REQUEST_LOGIN), api.Group_AUTHEN)
					login := api.Login_Request{
						UserName: "dungnt",
						Password: "admin",
					}

					// send ping request
					//rq := gBase.NewRequest(uint32(api.TYPE_ID_REQUEST_KEEPALIVE), api.Group_CONNECTION)
					//pingRequest := api.KeepAlive_Request{
					//	ConnectionId: hlReply.ConnectionId,
					//}
					if err := gBase.PackRequest(rq, &login); err != nil {
						fmt.Printf("%v", err.Error())
						break
					}
					_rq, err := gBase.MsgToByte(rq)
					if err != nil {
						fmt.Printf("%v", err.Error())
						break
					}
					newMsg := gBase.NewMessage(_rq, uint32(api.Group_AUTHEN), rq.Type, []byte{2, 3, 4, 5, 6})
					_p, _ := newMsg.Encode(gBase.Encryption_Type(hlReceive.ServerEncodeType), hlReceive.PKey)
					fmt.Printf("Send Login Request ")
					conn.Write(_p)

				}
			} else if msg.MsgType == uint32(api.AUTHEN_TYPE_ID_REQUEST_LOGIN) && msg.MsgGroup == uint32(api.Group_AUTHEN) {
				reply := api.Reply{}
				//msg.Lable = lable
				msg.ToProtoModel(&reply)

				reply.Reply.UnmarshalTo(&hlReply)
				if reply.Status == 0 {
					reply_login := api.Login_Reply{}

					if err := reply.Reply.UnmarshalTo(&reply_login); err != nil {
						fmt.Printf("Login Unmarshalto %v\n", err.Error())

					} else {
						fmt.Printf("Session %v", reply_login.Session.SessionId)
					}

				}

			} else if msg.MsgType == uint32(api.TYPE_ID_REQUEST_KEEPALIVE) && msg.MsgGroup == uint32(api.Group_CONNECTION) {
				time.Sleep(time.Second * 5)
				rq := gBase.NewRequest(uint32(api.TYPE_ID_REQUEST_KEEPALIVE), api.Group_CONNECTION)
				pingRequest := api.KeepAlive_Request{
					ConnectionId: hlReply.ConnectionId,
				}
				if err := gBase.PackRequest(rq, &pingRequest); err != nil {
					fmt.Printf("%v", err.Error())
					break
				}
				_rq, err := gBase.MsgToByte(rq)
				if err != nil {
					fmt.Printf("%v", err.Error())
					break
				}
				newMsg := gBase.NewMessage(_rq, uint32(api.Group_CONNECTION), rq.Type, []byte{2, 3, 4, 5, 6})
				_p, _ := newMsg.Encode(gBase.Encryption_Type(hlReceive.ServerEncodeType), hlReceive.PKey)
				conn.Write(_p)

			}

		}
	}
	<-done

}
