package api

import (
	"context"
	"net"
)

type RsNode struct {
	Addr     string
	Weight   int
	Metadata map[string]string
}
type ProxyConfig struct {
	// proxy机器启动的本地监听地址，如":8080"，如果未指定，则随机一个可用端口
	DsAddr string
	// 后端服务列表（real server）
	Rss []*RsNode
	Ctx context.Context
}

type Options struct {
	Proxy           ProxyConfig
	Lb              LoadBalancer
	RunnableFactory RunnableFactory
}

type Option func(*Options)

type Server interface {
	Options() Options
	Init(...Option) error
	// add real server
	AddRs(rs ...*RsNode) error
	// del real server
	DelRs(addr ...*RsNode) error
	// 启动
	Start() error
	Stop() error
}

type NextConn func() (net.Conn, error)
type RunnableFactory func(rw net.Conn, next NextConn) Runnable
type CompleteCallback func()
type Next func() (*RsNode, error)

type Runnable interface {
	Run(CompleteCallback) error
	Interrupt()
}

type LoadBalancer interface {
	Strategy(config *ProxyConfig) Next
}
