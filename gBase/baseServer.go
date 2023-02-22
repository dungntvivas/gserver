package gBase

import "gitlab.vivas.vn/go/internal/logger"
type TLS struct {
	IsTLS bool
	Cert  string
	Key   string
	h2_Enable bool
}

type GServer struct{
	Done   *chan struct{}
	Logger *logger.Logger
	Addr   string
	Tls    TLS
}


