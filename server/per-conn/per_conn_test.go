package per_conn

import (
	"github.com/mipoo/proxy-server/api"
	"github.com/mipoo/proxy-server/errors"
	"github.com/mipoo/proxy-server/server"
	"testing"
)

func TestPerConn(t *testing.T) {
	proxyServer, err := server.NormalProxyServer(PerConnRunnableFactory())
	if err != nil {
		t.Fatal(err)
	}
	proxyServer.Init(func(opts *api.Options) {
		opts.Proxy.DsAddr = ":8999"
	})
	proxyServer.AddRs(&api.RsNode{
		Addr: "127.0.0.1:51754",
		//Addr: "183.36.123.53:11111",
	})
	if err = proxyServer.Start(); err != nil && err != errors.ErrServerClosed {
		t.Fatal(err)
	}
}
