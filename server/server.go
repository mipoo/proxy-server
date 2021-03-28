package server

import (
	"context"
	"github.com/mipoo/proxy-server/api"
	"github.com/mipoo/proxy-server/errors"
	load_balancer "github.com/mipoo/proxy-server/load-balancer"
	"net"
	"sync"
	"time"
)

type normalServer struct {
	opts       api.Options
	ncs        *NodeConnStore
	running    bool
	mu         sync.Mutex
	doneChan   chan struct{}
	activeRuna map[api.Runnable]struct{}
	next       api.Next
}

func (s *normalServer) Options() api.Options {
	return s.opts
}

func (s *normalServer) Init(option ...api.Option) error {
	for _, opt := range option {
		opt(&s.opts)
	}
	return nil
}

func (s *normalServer) AddRs(rs ...*api.RsNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, node := range rs {
		if !s.containsNode(node) {
			err := s.ncs.AddRs(node)
			if err != nil {
				return err
			}
			s.opts.Proxy.Rss = append(s.opts.Proxy.Rss, node)
		}
	}

	return nil

	//if s.running {
	// TODO: re balance
	//}
}

func (s *normalServer) containsNode(node *api.RsNode) bool {
	for _, rs := range s.opts.Proxy.Rss {
		if rs.Addr == node.Addr {
			return true
		}
	}
	return false
}

func (s *normalServer) DelRs(rs ...*api.RsNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, node := range rs {
		for i, rs := range s.opts.Proxy.Rss {
			if rs.Addr == node.Addr {
				// remove rs .    eg: [3,5,7,9]  remove index 1  =>  [1,9,7]
				s.opts.Proxy.Rss[i] = s.opts.Proxy.Rss[len(s.opts.Proxy.Rss)-1]
				s.opts.Proxy.Rss[len(s.opts.Proxy.Rss)-1] = nil
				s.opts.Proxy.Rss = s.opts.Proxy.Rss[:len(s.opts.Proxy.Rss)-1]
				break
			}
		}
	}
	_ = s.ncs.DelRs(rs...)
	// TODO: re balance
	return nil
}

func (s *normalServer) Start() error {
	listen, err := net.Listen("tcp", s.Options().Proxy.DsAddr)
	if err != nil {
		return err
	}

	defer listen.Close()

	var tempDelay time.Duration // how long to sleep on accept failure

	for {
		rw, err := listen.Accept()
		if err != nil {
			select {
			case <-s.doneChan:
				return errors.ErrServerClosed
			default:
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				//s.logf("gateway: Accept error: %v; retrying in %v", err, tempDelay)
				//TODO: log
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		cc := s.newClientConn(rw)
		s.handler(cc)
	}
}

func (s *normalServer) newClientConn(rw net.Conn) api.Runnable {
	return s.opts.RunnableFactory(rw, s.nextRconn)
}

func (s *normalServer) nextRconn() (rc net.Conn, err error) {
	err = errors.ErrNotFound
	var node *api.RsNode
	for maxRetry := len(s.opts.Proxy.Rss); maxRetry > 0; maxRetry-- {
		node, err = s.nextRc()()
		if err != nil {
			continue
		}
		rc, err = s.ncs.GetConn(node)
		if err != nil {
			continue
		}
	}
	return
}

func (s *normalServer) nextRc() api.Next {
	if s.next == nil {
		s.mu.Lock()
		if s.next == nil {
			s.next = s.opts.Lb.Strategy(&s.opts.Proxy)
		}
		s.mu.Unlock()
	}
	return s.next
}

func (s *normalServer) handler(runnable api.Runnable) {
	s.trackRunnable(runnable, true)

	if err := runnable.Run(func() {
		s.trackRunnable(runnable, false)
	}); err != nil {
		s.trackRunnable(runnable, false)
	}
}
func (s *normalServer) trackRunnable(runnable api.Runnable, add bool) {
	s.mu.Lock()
	if add {
		s.activeRuna[runnable] = struct{}{}
	} else {
		delete(s.activeRuna, runnable)
	}
	s.mu.Unlock()
}

func (s *normalServer) Stop() error {
	s.mu.Lock()
	runas := s.activeRuna
	s.activeRuna = map[api.Runnable]struct{}{}
	s.mu.Unlock()

	for runnable := range runas {
		runnable.Interrupt()
	}
	s.doneChan <- struct{}{}
	s.ncs.Close()
	return nil
}

func NormalProxyServer(opt ...api.Option) (api.Server, error) {
	opts := api.Options{
		Proxy: api.ProxyConfig{
			Rss: make([]*api.RsNode, 0),
			Ctx: context.Background(),
		},
		Lb:              nil,
		RunnableFactory: nil,
	}
	load_balancer.Random()(&opts)
	for _, o := range opt {
		o(&opts)
	}

	ncs := &NodeConnStore{
		ncs: map[string]*nodeConns{},
	}
	err := ncs.AddRs(opts.Proxy.Rss...)
	if err != nil {
		// TODO: add log
		return nil, err
	}
	return &normalServer{
		opts:       opts,
		ncs:        ncs,
		activeRuna: map[api.Runnable]struct{}{},
	}, nil
}
