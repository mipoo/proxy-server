package server

import (
	"github.com/fatih/pool"
	"github.com/mipoo/proxy-server/api"
	"github.com/mipoo/proxy-server/errors"
	"net"
	"sync"
	"time"
)

type NodeConnStore struct {
	mu  sync.Mutex
	ncs map[string]*nodeConns
}

func (ncs *NodeConnStore) AddRs(rs ...*api.RsNode) error {
	for _, node := range rs {
		ncs.mu.Lock()
		if _, ok := ncs.ncs[node.Addr]; !ok {
			conns, err := newNodeConns(node)
			if err != nil {
				return err
			}
			ncs.ncs[node.Addr] = conns
		}
		ncs.mu.Unlock()
	}
	return nil
}

func (ncs *NodeConnStore) DelRs(rs ...*api.RsNode) error {
	for _, rs := range rs {
		ncs.mu.Lock()
		if nc, ok := ncs.ncs[rs.Addr]; ok {
			delete(ncs.ncs, rs.Addr)
			nc.Close()
		}
		ncs.mu.Unlock()
	}
	return nil
}

func (ncs *NodeConnStore) GetConn(rs *api.RsNode) (net.Conn, error) {
	if nc, ok := ncs.ncs[rs.Addr]; ok {
		return nc.Get()
	}
	return nil, errors.ErrNotFound
}

func (ncs *NodeConnStore) Close() {
	ncs.mu.Lock()
	ns := ncs.ncs
	ncs.ncs = nil
	ncs.mu.Unlock()
	for _, nodeConn := range ns {
		nodeConn.Close()
	}
}

type nodeConns struct {
	addr     string
	metadata map[string]string
	connPool pool.Pool
}

func newNodeConns(node *api.RsNode) (*nodeConns, error) {
	cp, err := pool.NewChannelPool(0, 50, func() (net.Conn, error) {
		return net.Dial("tcp", node.Addr)
	})
	if err != nil {
		return nil, err
	}
	return &nodeConns{
		addr:     node.Addr,
		metadata: node.Metadata,
		connPool: cp,
	}, nil
}

func (nc *nodeConns) Get() (net.Conn, error) {
	conn, err := nc.connPool.Get()
	if err == nil {
		nc.reset(conn)
	}
	return conn, err
}
func (nc *nodeConns) reset(conn net.Conn) {
	_ = conn.SetReadDeadline(time.Time{})
}

func (nc *nodeConns) Close() {
	nc.connPool.Close()
}
