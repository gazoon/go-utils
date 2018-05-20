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
	name       string
	fetchDelay time.Duration
	wg         sync.WaitGroup
	stopFlag   int32
}

func New(fetch func(context.Context) interface{},
	process func(context.Context, interface{}),
	consumerName string, fetchDelay int) *Consumer {

	return &Consumer{
		fetch: fetch, process: process, name: consumerName,
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
	logger := log.WithField("consumer", self.name)
	logger.Info("Stop processing")
	atomic.StoreInt32(&self.stopFlag, 1)
	isTimeout := utils.WaitTimeout(&self.wg, time.Second*5)
	if isTimeout {
		logger.Warning("Stop processing took to long")
	}
}
