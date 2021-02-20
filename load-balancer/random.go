package load_balancer

import (
	"github.com/mipoo/proxy-server/api"
	"github.com/mipoo/proxy-server/errors"
	"math/rand"
)

type randomLB struct {
}

func Random() api.Option {
	return func(opt *api.Options) {
		opt.Lb = &randomLB{}
	}
}

func (lb *randomLB) Strategy(proxy *api.ProxyConfig) api.Next {
	return func() (*api.RsNode, error) {
		len := len(proxy.Rss)
		if len == 0 {
			return nil, errors.ErrNotFound
		}
		return proxy.Rss[rand.Intn(len)], nil
	}
}
