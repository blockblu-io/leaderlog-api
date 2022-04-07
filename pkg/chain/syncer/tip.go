package syncer

import (
	"context"
	"github.com/blockblu-io/leaderlog-api/pkg/chain"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type TipUpdater struct {
	lock    sync.RWMutex
	tip     *chain.Tip
	UpdateC chan struct{}
}

func NewTipUpdater() *TipUpdater {
	tipUpdateChan := make(chan struct{})
	return &TipUpdater{
		lock:    sync.RWMutex{},
		tip:     nil,
		UpdateC: tipUpdateChan,
	}
}

func (tu *TipUpdater) GetTip() *chain.Tip {
	tu.lock.RLock()
	defer tu.lock.RUnlock()
	return tu.tip
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
			tu.UpdateC <- struct{}{}
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
		close(tu.UpdateC)
	}()
}
