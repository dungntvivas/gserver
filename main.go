package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/dungntvivas/grpc_api/api"
	"github.com/dungntvivas/gserver/gBase"
	"github.com/dungntvivas/gserver/gTCP"
	"github.com/dungntvivas/gserver/gTLS"
	"github.com/pion/dtls/v2"

	"net"
	"time"

	"github.com/dungntvivas/libinternal/encryption/aes"
	"github.com/dungntvivas/libinternal/encryption/rsa"
	"github.com/dungntvivas/libinternal/encryption/xor"
	"github.com/dungntvivas/libinternal/logger"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var interrupt chan os.Signal
var tls_server *gTLS.TLSServer
var done chan struct{}

func main() {

	done = make(chan struct{})
	chReceiveRequest := make(chan *gBase.Payload)
	_logger, err := logger.New(logger.Info, []logger.Destination{logger.DestinationFile, logger.DestinationStdout}, "./server.log")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(99)
	}
	_logger.Log(logger.Info, "START GS SERVER")
	//
	//cf := gBase.DefaultHttpsConfigOption
	//cf.Logger = _logger
	////cf.Tls.Cert = "/Users/dungnt/Desktop/vivas/certificate.pem"
	////cf.Tls.Key = "/Users/dungnt/Desktop/vivas/private.key"
	////cf.Done = &done
	////hsv := gHTTP.New(cf,chReceiveRequest)
	////hsv.Serve()
	//cf_dtls := gBase.DefaultDTLSSocketConfigOption
	//cf_dtls.EncodeType = gBase.Encryption_AES
	//cf_dtls.Logger = _logger
	//cf_dtls.Done = &done
	//
	//cf_tls := gBase.DefaultTlsSocketConfigOption
	//cf_tls.Tls.Cert = "/Users/dungnt/Desktop/vivas/certificate.pem"
	//cf_tls.Tls.Key = "/Users/dungnt/Desktop/vivas/private.key"
	//cf_tls.Done = &done
	//cf_tls.EncodeType = gBase.Encryption_RSA
	//cf_tls.Logger = _logger
	//tls_server = gTLS.New(cf_tls, chReceiveRequest)
	//tls_server.Serve()

	interrupt = make(chan os.Signal, 1)

	signal.Notify(interrupt, os.Interrupt)
	//
	//
	tcp_option := gBase.DefaultTcpSocketConfigOption
	tcp_option.EncodeType = gBase.Encryption_XOR
	tcp_option.Logger = _logger
	tcp_option.Done = &done
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
	go func() {
		tcp := gTCP.New(tcp_option, chReceiveRequest)
		tcp.Serve()
	}()
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

	//conn, err := net.Dial("tcp", "10.3.3.119:8083")
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//buf := make([]byte, 4096)
	//
	//hlReceive := api.HelloReceive{}
	//
	//encode := api.EncodeType_AES
	//_rsa, _ := rsa.VRSA_NEW()
	//_ = _rsa.GetPublicKey()
	//xor_lable, _ := xor.NEW_XOR_KEY()
	//hlReply := api.Hello_Reply{}
	//aes_key, _ := aes.NEW_AES_KEY()
	//
	//for {
	//	n, err := conn.Read(buf)
	//	if err != nil {
	//		if err != io.EOF {
	//			fmt.Println("read error:", err)
	//		}
	//		break
	//	}
	//	if n != 0 {
	//		fmt.Printf("Receive %d byte\n", n)
	//		msg := gBase.DecodeData(buf[0:n], n)
	//		if msg.MSG_encode_decode_type == gBase.Encryption_RSA {
	//			msg.Conn = &gBase.Connection{
	//				Server: &gBase.ServerConnection{
	//					Rsa: _rsa,
	//				},
	//			}
	//		} else if msg.MSG_encode_decode_type == gBase.Encryption_AES {
	//			msg.Conn = &gBase.Connection{
	//				Server: &gBase.ServerConnection{
	//					PKey: aes_key,
	//				},
	//			}
	//		} else if msg.MSG_encode_decode_type == gBase.Encryption_XOR {
	//			msg.Lable = xor_lable
	//		}
	//
	//		fmt.Printf("Msg Type %v group %v\n", msg.MsgType, msg.MsgGroup)
	//		if msg.MsgType == uint32(api.TYPE_ID_RECEIVE_HELLO) && msg.MsgGroup == uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID) {
	//			/// Receive Msg
	//			receive := api.Receive{}
	//			msg.ToProtoModel(&receive)
	//			fmt.Printf("Server time => %v\n", receive.ServerTime)
	//
	//			receive.Receive.UnmarshalTo(&hlReceive)
	//			fmt.Printf("Server Encrypt Type => %v\n", hlReceive.ServerEncodeType.String())
	//
	//			// setup client receive encode
	//			request := api.Request{
	//				Type:  uint32(api.TYPE_ID_REQUEST_HELLO),
	//				Group: uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID),
	//			}
	//			request_hello := api.Hello_Request{
	//				EncodeType: encode,
	//				PKey:       aes_key,
	//				Platform:   api.Platform_OTHER,
	//			}
	//			_request_hello, _ := anypb.New(&request_hello)
	//			request.Request = _request_hello
	//			// api.Request --> binary
	//			_request, _ := proto.Marshal(&request)
	//
	//			// encode send to server
	//			newMsg := gBase.NewMessage(_request, uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID), request.Type, []byte{1, 2, 3, 4, 5})
	//			_p, _ := newMsg.Encode(gBase.Encryption_Type(hlReceive.ServerEncodeType), hlReceive.PKey,false)
	//
	//			fmt.Printf("%v", _p)
	//			conn.Write(_p)
	//		} else if msg.MsgType == uint32(api.TYPE_ID_REQUEST_HELLO) && msg.MsgGroup == uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID) {
	//			reply := api.Reply{}
	//			//msg.Lable = lable
	//			msg.ToProtoModel(&reply)
	//
	//			reply.Reply.UnmarshalTo(&hlReply)
	//
	//			if reply.Status == 0 {
	//				/// send login
	//				rq := gBase.NewRequest(uint32(api.AUTHEN_TYPE_ID_REQUEST_LOGIN), uint32(api.AUTHEN_GROUP_AUTHEN_GROUP_ID))
	//				login := api.Login_Request{
	//					UserName: "dungnt",
	//					Password: "admin",
	//				}
	//
	//				// send ping request
	//				rq := gBase.NewRequest(uint32(api.TYPE_ID_REQUEST_KEEPALIVE), api.Group_CONNECTION)
	//				pingRequest := api.KeepAlive_Request{
	//					ConnectionId: hlReply.ConnectionId,
	//				}
	//				if err := gBase.PackRequest(rq, &login); err != nil {
	//					fmt.Printf("%v", err.Error())
	//					break
	//				}
	//				_rq, err := gBase.MsgToByte(rq)
	//				if err != nil {
	//					fmt.Printf("%v", err.Error())
	//					break
	//				}
	//				newMsg := gBase.NewMessage(_rq, uint32(api.AUTHEN_GROUP_AUTHEN_GROUP_ID), rq.Type, []byte{2, 3, 4, 5, 6})
	//				_p, _ := newMsg.Encode(gBase.Encryption_Type(hlReceive.ServerEncodeType), hlReceive.PKey,false)
	//				fmt.Printf("Send Login Request ")
	//				conn.Write(_p)

	//			}
	//		} else if msg.MsgType == uint32(api.AUTHEN_TYPE_ID_REQUEST_LOGIN) && msg.MsgGroup == uint32(api.AUTHEN_GROUP_AUTHEN_GROUP_ID) {
	//			reply := api.Reply{}
	//			//msg.Lable = lable
	//			msg.ToProtoModel(&reply)
	//
	//			reply.Reply.UnmarshalTo(&hlReply)
	//			if reply.Status == 0 {
	//				reply_login := api.Login_Reply{}
	//
	//				if err := reply.Reply.UnmarshalTo(&reply_login); err != nil {
	//					fmt.Printf("Login Unmarshalto %v\n", err.Error())
	//
	//				} else {
	//					fmt.Printf("Session %v", reply_login.Session.SessionId)
	//
	//					rq := gBase.NewRequest(uint32(api.TYPE_ID_REQUEST_KEEPALIVE), uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID))
	//					pingRequest := api.KeepAlive_Request{
	//						ConnectionId: hlReply.ConnectionId,
	//					}
	//					if err := gBase.PackRequest(rq, &pingRequest); err != nil {
	//						fmt.Printf("%v", err.Error())
	//						break
	//					}
	//					_rq, err := gBase.MsgToByte(rq)
	//					if err != nil {
	//						fmt.Printf("%v", err.Error())
	//						break
	//					}
	//					newMsg := gBase.NewMessage(_rq, uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID), rq.Type, []byte{2, 3, 4, 5, 6})
	//					_p, _ := newMsg.Encode(gBase.Encryption_Type(hlReceive.ServerEncodeType), hlReceive.PKey,false)
	//					conn.Write(_p)
	//				}
	//
	//			}
	//
	//		} else if msg.MsgType == uint32(api.TYPE_ID_REQUEST_KEEPALIVE) && msg.MsgGroup == uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID) {
	//			time.Sleep(time.Second * 5)
	//			rq := gBase.NewRequest(uint32(api.TYPE_ID_REQUEST_KEEPALIVE), uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID))
	//			pingRequest := api.KeepAlive_Request{
	//				ConnectionId: hlReply.ConnectionId,
	//			}
	//			if err := gBase.PackRequest(rq, &pingRequest); err != nil {
	//				fmt.Printf("%v", err.Error())
	//				break
	//			}
	//			_rq, err := gBase.MsgToByte(rq)
	//			if err != nil {
	//				fmt.Printf("%v", err.Error())
	//				break
	//			}
	//			newMsg := gBase.NewMessage(_rq, uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID), rq.Type, []byte{2, 3, 4, 5, 6})
	//			_p, _ := newMsg.Encode(gBase.Encryption_Type(hlReceive.ServerEncodeType), hlReceive.PKey,false)
	//			conn.Write(_p)
	//
	//		}
	//
	//	}
	//}

	//go dtls_client(&done)
	//go tls_client(&done)
	go onClose()
	<-done

}

func onClose() {
loop:
	for {
		select {
		case <-interrupt:
			break loop
		}
	}
	if tls_server != nil {
		tls_server.Close()
	}
	close(done)

}

func tls_client(done *chan struct{}) {
	time.Sleep(time.Second * 2)
	conf := &tls.Config{
		InsecureSkipVerify: true,
	}

	conn, err := tls.Dial("tcp", "127.0.0.1:44224", conf)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()
	state := conn.ConnectionState()
	for _, v := range state.PeerCertificates {
		fmt.Println(x509.MarshalPKIXPublicKey(v.PublicKey))
		fmt.Println(v.Subject)
	}
	go func(_conn *tls.Conn) {
		buf := make([]byte, 8192)
		_rsa, _ := rsa.VRSA_NEW()
		aes_key, _ := aes.NEW_AES_KEY()
		xor_lable, _ := xor.NEW_XOR_KEY()
		encode := api.EncodeType_AES
		hlReceive := api.HelloReceive{}
		hlReply := api.Hello_Reply{}

		for {
			n, err := _conn.Read(buf)
			if err != nil {
				break
			}
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

			if msg.MsgType == uint32(api.TYPE_ID_RECEIVE_HELLO) && msg.MsgGroup == uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID) {
				receive := api.Receive{}
				msg.ToProtoModel(&receive)
				//fmt.Printf("Server time => %v\n", receive.ServerTime)

				receive.Receive.UnmarshalTo(&hlReceive)
				//fmt.Printf("Server Encrypt Type => %v\n", hlReceive.ServerEncodeType.String())

				// setup client receive encode
				request := api.Request{
					Type:  uint32(api.TYPE_ID_REQUEST_HELLO),
					Group: uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID),
				}
				request_hello := api.Hello_Request{
					EncodeType: encode,
					PKey:       aes_key,
					Platform:   api.Platform_WEB,
				}
				_request_hello, _ := anypb.New(&request_hello)
				request.Request = _request_hello
				// api.Request --> binary
				_request, _ := proto.Marshal(&request)

				// encode send to server
				newMsg := gBase.NewMessage(_request, uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID), request.Type, []byte{1, 2, 3, 4, 5})
				_p, _ := newMsg.Encode(gBase.Encryption_Type(hlReceive.ServerEncodeType), hlReceive.PKey, false)

				_conn.Write(_p)
			} else if msg.MsgType == uint32(api.TYPE_ID_REQUEST_HELLO) && msg.MsgGroup == uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID) {
				reply := api.Reply{}
				//msg.Lable = lable
				msg.ToProtoModel(&reply)

				reply.Data.UnmarshalTo(&hlReply)

				if reply.Status == 0 {
					fmt.Printf("Setup Connection OK => Send PING REQUEST\n")
					// send ping request
					rq := gBase.NewRequest(uint32(api.TYPE_ID_REQUEST_KEEPALIVE), uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID))
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
					newMsg := gBase.NewMessage(_rq, uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID), rq.Type, []byte{2, 3, 4, 5, 6})
					_p, _ := newMsg.Encode(gBase.Encryption_Type(hlReceive.ServerEncodeType), hlReceive.PKey, false)
					_conn.Write(_p)
				}
			} else if msg.MsgType == uint32(api.TYPE_ID_REQUEST_KEEPALIVE) && msg.MsgGroup == uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID) {
				fmt.Printf("PING REQUEST\n")
				time.Sleep(time.Second * 5)
				rq := gBase.NewRequest(uint32(api.TYPE_ID_REQUEST_KEEPALIVE), uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID))
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
				newMsg := gBase.NewMessage(_rq, uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID), rq.Type, []byte{2, 3, 4, 5, 6})
				_p, _ := newMsg.Encode(gBase.Encryption_Type(hlReceive.ServerEncodeType), hlReceive.PKey, false)
				_conn.Write(_p)
			}
		}
	}(conn)
	<-*done
}

func dtls_client(done *chan struct{}) {
	time.Sleep(time.Second * 2)
	addr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: 44444}
	config := &dtls.Config{
		PSK: func(hint []byte) ([]byte, error) {
			fmt.Printf("Server's hint: %s \n", hint)
			return []byte{0x86, 0x73, 0x86, 0x65, 0x83}, nil
		},
		PSKIdentityHint:      []byte("Pion DTLS Server"),
		CipherSuites:         []dtls.CipherSuiteID{dtls.TLS_PSK_WITH_AES_256_CCM_8},
		ExtendedMasterSecret: dtls.RequireExtendedMasterSecret,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dtlsConn, err := dtls.DialWithContext(ctx, "udp", addr, config)
	if err != nil {
		fmt.Printf("err %v", err.Error())
		return
	}
	fmt.Printf("Connected %v\n", dtlsConn.RemoteAddr().String())

	go func(_conn *dtls.Conn) {
		buf := make([]byte, 8192)
		_rsa, _ := rsa.VRSA_NEW()
		aes_key, _ := aes.NEW_AES_KEY()
		xor_lable, _ := xor.NEW_XOR_KEY()
		encode := api.EncodeType_AES
		hlReceive := api.HelloReceive{}
		hlReply := api.Hello_Reply{}
		for {
			n, err := _conn.Read(buf)
			if err != nil {
				break
			}
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

			//fmt.Printf("Msg Type %v group %v\n", msg.MsgType, msg.MsgGroup)
			if msg.MsgType == uint32(api.TYPE_ID_RECEIVE_HELLO) && msg.MsgGroup == uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID) {
				/// Receive Msg
				receive := api.Receive{}
				msg.ToProtoModel(&receive)
				//fmt.Printf("Server time => %v\n", receive.ServerTime)

				receive.Receive.UnmarshalTo(&hlReceive)
				//fmt.Printf("Server Encrypt Type => %v\n", hlReceive.ServerEncodeType.String())

				// setup client receive encode
				request := api.Request{
					Type:  uint32(api.TYPE_ID_REQUEST_HELLO),
					Group: uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID),
				}
				request_hello := api.Hello_Request{
					EncodeType: encode,
					PKey:       aes_key,
					Platform:   api.Platform_WEB,
				}
				_request_hello, _ := anypb.New(&request_hello)
				request.Request = _request_hello
				// api.Request --> binary
				_request, _ := proto.Marshal(&request)

				// encode send to server
				newMsg := gBase.NewMessage(_request, uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID), request.Type, []byte{1, 2, 3, 4, 5})
				_p, _ := newMsg.Encode(gBase.Encryption_Type(hlReceive.ServerEncodeType), hlReceive.PKey, false)

				_conn.Write(_p)
			} else if msg.MsgType == uint32(api.TYPE_ID_REQUEST_HELLO) && msg.MsgGroup == uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID) {
				reply := api.Reply{}
				//msg.Lable = lable
				msg.ToProtoModel(&reply)

				reply.Data.UnmarshalTo(&hlReply)

				if reply.Status == 0 {
					fmt.Printf("Setup Connection OK => Send PING REQUEST\n")
					// send ping request
					rq := gBase.NewRequest(uint32(api.TYPE_ID_REQUEST_KEEPALIVE), uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID))
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
					newMsg := gBase.NewMessage(_rq, uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID), rq.Type, []byte{2, 3, 4, 5, 6})
					_p, _ := newMsg.Encode(gBase.Encryption_Type(hlReceive.ServerEncodeType), hlReceive.PKey, false)
					_conn.Write(_p)
				}
			} else if msg.MsgType == uint32(api.TYPE_ID_REQUEST_KEEPALIVE) && msg.MsgGroup == uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID) {
				fmt.Printf("PING REQUEST\n")
				time.Sleep(time.Second * 5)
				rq := gBase.NewRequest(uint32(api.TYPE_ID_REQUEST_KEEPALIVE), uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID))
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
				newMsg := gBase.NewMessage(_rq, uint32(api.CONNECTION_GROUP_CONNECTION_GROUP_ID), rq.Type, []byte{2, 3, 4, 5, 6})
				_p, _ := newMsg.Encode(gBase.Encryption_Type(hlReceive.ServerEncodeType), hlReceive.PKey, false)
				_conn.Write(_p)
			}
		}
	}(dtlsConn)
	<-*done
}
