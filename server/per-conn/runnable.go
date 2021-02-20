package per_conn

import (
	"context"
	"github.com/fatih/pool"
	"github.com/mipoo/proxy-server/api"
	"github.com/mipoo/proxy-server/logger"
	"go.uber.org/zap"
	"io"
	"net"
	"strings"
	"time"
)

type perConnRunnable struct {
	// client connection
	cc net.Conn
	// real server connection
	rc     net.Conn
	nextRc api.NextConn
	ctx    context.Context
	done   context.CancelFunc
}

func PerConnRunnableFactory() api.Option {
	return func(opt *api.Options) {
		opt.RunnableFactory = perConnRunnableFactory
	}
}

func perConnRunnableFactory(cc net.Conn, nextRc api.NextConn) api.Runnable {
	return &perConnRunnable{
		cc:     cc,
		nextRc: nextRc,
	}
}

func (p perConnRunnable) Run(callback api.CompleteCallback) error {
	go p.run(callback)
	return nil
}

func (p perConnRunnable) run(callback api.CompleteCallback) {
	defer callback()
	defer p.cc.Close()
	rc, err := p.nextRc()
	if err != nil {
		logger.Warn("found next rc error", zap.String("cc", p.cc.RemoteAddr().String()), zap.Error(err))
		return
	}
	defer rc.Close()

	ctx, done := context.WithCancel(context.Background())
	p.rc = rc
	p.ctx = ctx
	p.done = done

	go func() {
		// copy cc to rc
		_, err := io.Copy(p.rc, p.cc)
		if err == nil {
			// cc EOF  , notify rc no need to read
			_ = p.rc.SetReadDeadline(time.Now())
		} else if !isTimeout(err) {
			logger.Error("copy cc to rc error",
				zap.String("rc", p.rc.RemoteAddr().String()),
				zap.String("cc", p.cc.RemoteAddr().String()),
				zap.Error(err))
		}
		done()
	}()

	go func() {
		// copy rc to cc
		_, err := io.Copy(p.cc, p.rc)
		if err == nil {
			// rc EOF  , notify cc no need to read
			_ = p.cc.SetReadDeadline(time.Now())
			if poolConn, ok := p.rc.(*pool.PoolConn); ok {
				poolConn.MarkUnusable()
			}
		} else if !isTimeout(err) {
			if poolConn, ok := p.rc.(*pool.PoolConn); ok {
				poolConn.MarkUnusable()
			}
			logger.Error("copy rc to cc error",
				zap.String("rc", p.rc.RemoteAddr().String()),
				zap.String("cc", p.cc.RemoteAddr().String()),
				zap.Error(err))
		}
		done()
	}()
	<-ctx.Done()
}

func (p perConnRunnable) Interrupt() {
	p.done()
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	if opErr, ok := err.(*net.OpError); ok {
		// poll.ErrDeadlineExceeded
		if strings.Contains(opErr.Err.Error(), "i/o timeout") {
			return true
		}
	}
	return false
}
