package gDTLS

import (
	"context"
	"github.com/pion/dtls/v2"
	"gitlab.vivas.vn/go/gserver/gBase"
	"net"
	"sync"
	"time"
)

type DTLSServer struct {
	gBase.GServer
	Done chan struct{}
	ctx context.Context
	cancel context.CancelFunc
	listener net.Listener


	mu      sync.Mutex
	clients sync.Map // ADDRESS ==> Connection ( 1 connection chứa thông tin kết nối ) client_id has connection

	mu_token sync.Mutex
	sessions sync.Map // token ==> FD ( 1 token có nhiều connection ) session_id has client_id

	mu_user sync.Mutex
	users   sync.Map // user ==> TOKEN ( 1 user có nhiều token ) user_id has session_id

	// out
	chReceiveMsg chan *gBase.SocketMessage
	// in


}
func New(config gBase.ConfigOption, chReceiveRequest chan *gBase.Payload) *DTLSServer {

	b := gBase.GServer{
		Config:           &config,
		ChReceiveRequest: chReceiveRequest,
	}
	p := &DTLSServer{
		GServer: b,
		Done:         make(chan struct{}),
		chReceiveMsg: make(chan *gBase.SocketMessage, 100),
	}

	var err error
	p.ctx, p.cancel = context.WithCancel(context.Background())
	_config := dtls.Config{
		PSK: func(hint []byte) ([]byte, error) {
			return []byte{0x86, 0x73, 0x86, 0x65, 0x83}, nil
		},
		PSKIdentityHint:      []byte("VIVAS-RDPA DTLS Client"),
		CipherSuites:         []dtls.CipherSuiteID{dtls.TLS_PSK_WITH_AES_256_CCM_8},
		ExtendedMasterSecret: dtls.RequireExtendedMasterSecret,
		ConnectContextMaker: func() (context.Context, func()) {
			return context.WithTimeout(p.ctx, 30*time.Second)
		},
	}

	addr := &net.UDPAddr{IP: net.ParseIP(p.Config.Addr), Port: 44444}
	p.listener, err = dtls.Listen("udp", addr, &_config)
	if err != nil {
		p.LogError("dtls listen %v",err.Error())
		return nil
	}



	return p
}

func (p *DTLSServer) Serve() error {
	p.LogInfo("Listening")
	go p.wait_for_a_connection()
	return nil
}
func (p *DTLSServer)wait_for_a_connection(){
	// Wait for a connection.
	conn, err := p.listener.Accept()
	if err != nil { /// store connection
		p.LogInfo("New Connection Error %v",err.Error())
	}
	p.LogInfo("Client %v %v",conn.RemoteAddr().String(),conn.RemoteAddr().Network())

}
func (p *DTLSServer) Close() {
	p.LogInfo("Close")
	p.cancel()
	p.listener.Close()

}
