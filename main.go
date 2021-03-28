package main

import (
	"fmt"
	"github.com/mipoo/proxy-server/api"
	"github.com/mipoo/proxy-server/errors"
	"github.com/mipoo/proxy-server/server"
	per_conn "github.com/mipoo/proxy-server/server/per-conn"
	"net/http"
	_ "net/http/pprof"
	"os"
)

func main() {

	go func() {
		if err := http.ListenAndServe(":8079", nil); err != nil {
		}
	}()

	proxyServer, err := server.NormalProxyServer(per_conn.PerConnRunnableFactory(true))
	if err != nil {
		return
	}
	proxyServer.Init(func(opts *api.Options) {
		opts.Proxy.DsAddr = ":8999"
	})
	proxyServer.AddRs(&api.RsNode{
		Addr: "127.0.0.1:6380",
		//Addr: "10.218.15.42:59359",
		//Addr: "183.36.123.53:11111",
	})
	fmt.Printf("pid:%v", os.Getpid())
	if err = proxyServer.Start(); err != nil && err != errors.ErrServerClosed {
		return
	}
}
