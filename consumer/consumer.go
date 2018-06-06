package consumer

import (
	"context"
	log "github.com/Sirupsen/logrus"
	"github.com/gazoon/go-utils"
	"sync"
	"sync/atomic"
	"time"
)

type Process func()
type Fetch func(context.Context) Process

type Consumer struct {
	fetch      Fetch
	fetchDelay time.Duration
	wg         sync.WaitGroup
	stopFlag   int32
}

func New(fetch Fetch, fetchDelay int) *Consumer {

	return &Consumer{
		fetch: fetch, fetchDelay: time.Duration(fetchDelay) * time.Millisecond,
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
		process := self.fetch(ctx)
		if process == nil {
			time.Sleep(self.fetchDelay)
			continue
		}
		self.wg.Add(1)
		go func() {
			defer self.wg.Done()
			process()
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
