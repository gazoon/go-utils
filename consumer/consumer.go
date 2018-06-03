package consumer

import (
	"context"
	log "github.com/Sirupsen/logrus"
	"github.com/gazoon/go-utils"
	"sync"
	"sync/atomic"
	"time"
)

type Consumer struct {
	fetch      func(context.Context) interface{}
	process    func(context.Context, interface{})
	fetchDelay time.Duration
	wg         sync.WaitGroup
	stopFlag   int32
}

func New(fetch func(context.Context) interface{},
	process func(context.Context, interface{}), fetchDelay int) *Consumer {

	return &Consumer{
		fetch: fetch, process: process,
		fetchDelay: time.Duration(fetchDelay) * time.Millisecond,
	}
}

func (self *Consumer) Run() {
	go self.runLoop()
}

func (self *Consumer) runLoop() {
	for {
		if atomic.LoadInt32(&self.stopFlag) == 1 {
			return
		}
		ctx := context.Background()
		data := self.fetch(ctx)
		if data == nil {
			time.Sleep(self.fetchDelay)
			continue
		}
		self.wg.Add(1)
		go func() {
			defer self.wg.Done()
			self.process(ctx, data)
		}()
	}
}

func (self *Consumer) Stop() {
	log.Info("Stop consuming")
	atomic.StoreInt32(&self.stopFlag, 1)
	isTimeout := utils.WaitTimeout(&self.wg, time.Second*5)
	if isTimeout {
		log.Warning("Stop processing took to long")
	}
}
