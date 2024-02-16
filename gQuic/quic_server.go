package gquic

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"

	"github.com/dungntvivas/gserver/gBase"
	"github.com/quic-go/quic-go"
)

type QuicServer struct {
	gBase.GServer
	listener quic.Listener
}

func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-rdpa-vivas"},
	}
}
func New(config gBase.ConfigOption, chReceiveRequest chan *gBase.Payload) *QuicServer {
	b := gBase.GServer{
		Config:           &config,
		ChReceiveRequest: chReceiveRequest,
	}

	p := &QuicServer{
		GServer: b,
	}
	var err error
	p.listener, err = quic.ListenAddr(b.Config.Addr, generateTLSConfig(), nil)
	if err != nil {
		return nil
	}

	return p

}
func (p *QuicServer) Serve() error {
	go p.AcceptConnection()

	return nil
}

func (p *QuicServer) AcceptConnection() {
loop:
	for {
		conn, err := p.listener.Accept(context.Background())
		if err != nil {
			break loop
		}
		go p.AcceptStreamFromConnection(&conn)

	}
}
func (p *QuicServer) AcceptStreamFromConnection(conn *quic.Connection) {
loop:
	for {
		_, err := (*conn).OpenStreamSync(context.Background())
		if err != nil {
			break loop
		}

	}
}
