package syncer

import (
	"context"
	"github.com/blockblu-io/leaderlog-api/pkg/chain"
	log "github.com/sirupsen/logrus"
	"math"
	"sync"
	"time"
)

var (
	observerIDLock = sync.Mutex{}
	observerID     = 0
)

func getObserverID() int {
	observerIDLock.Lock()
	defer observerIDLock.Unlock()
	if observerID == math.MaxInt {
		observerID = 0
	}
	observerID = observerID + 1
	return observerID
}

type TipUpdater struct {
	lock     sync.RWMutex
	tip      *chain.Tip
	observer observer
}

type observer struct {
	lock      sync.RWMutex
	listeners []*TipListener
}

type TipListener struct {
	observer *observer
	id       int
	C        chan struct{}
}

func NewTipUpdater() *TipUpdater {
	return &TipUpdater{
		lock: sync.RWMutex{},
		tip:  nil,
	}
}

func (tu *TipUpdater) Subscribe() *TipListener {
	c := make(chan struct{})
	t := &TipListener{
		observer: &tu.observer,
		id:       getObserverID(),
		C:        c,
	}
	tu.observer.lock.Lock()
	defer tu.observer.lock.Unlock()
	tu.observer.listeners = append(tu.observer.listeners, t)
	return t
}

func (tpl *TipListener) Unsubscribe() {
	tpl.observer.lock.Lock()
	defer tpl.observer.lock.Unlock()
	var listeners []*TipListener
	for _, otpl := range tpl.observer.listeners {
		if tpl.id != otpl.id {
			listeners = append(listeners, otpl)
		}
	}
	tpl.observer.listeners = listeners
	close(tpl.C)
}

func (tu *TipUpdater) GetTip() *chain.Tip {
	tu.lock.RLock()
	defer tu.lock.RUnlock()
	return tu.tip
}

func (tu *TipUpdater) publish() {
	tu.observer.lock.RLock()
	defer tu.observer.lock.RUnlock()
	for _, l := range tu.observer.listeners {
		go func(l *TipListener) {
			l.C <- struct{}{}
		}(l)
	}
}

func (tu *TipUpdater) Run(ctx context.Context, backend chain.Backend) {
	gather := func(ctx context.Context, backend chain.Backend) {
		tip, err := backend.GetLatestBlock(ctx)
		if err != nil {
			log.Errorf("tip cpuldn't be gathered: %s", err.Error())
		} else {
			log.Infof("fetched the tip (%d,%d) with hash=%s (minted at %s)",
				tip.Epoch, tip.SlotInEpoch, tip.Hash,
				time.Unix(int64(tip.Timestamp), 0))
			tu.lock.Lock()
			defer tu.lock.Unlock()
			tu.tip = tip
			go tu.publish()
		}
	}
	go func() {
		gather(ctx, backend)
		keepOn := true
		for keepOn {
			timer := time.NewTimer(1 * time.Minute)
			select {
			case <-timer.C:
				gather(ctx, backend)
			case <-ctx.Done():
				keepOn = false
			}
			timer.Stop()
		}

	}()
}
