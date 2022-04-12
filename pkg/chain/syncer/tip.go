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
	DefaultTipUpdaterConfig = &TipUpdaterConfiguration{
		Interval: 1 * time.Minute,
	}
)

// TipUpdaterConfiguration configures the behaviour of the TipUpdater.
type TipUpdaterConfiguration struct {
	// Interval specifies the interval at which the tip of the network will
	// be fetched.
	Interval time.Duration
}

// TipUpdater fetches the tip of the Cardano network regularly. The interval
// for the TipUpdater can be configured over the TipUpdaterConfiguration.
type TipUpdater struct {
	tip      tip
	observer observer
	config   *TipUpdaterConfiguration
}

type tip struct {
	lock  sync.RWMutex
	value *chain.Tip
}

type observer struct {
	lock      sync.RWMutex
	idCounter int
	listeners []*tipListener
}

type CancelFunc = func()

// tipListener offers a channel that signals when the tip has been updated.
type tipListener struct {
	id int
	c  chan<- struct{}
}

// getObserverID is getting the ID for an observer lister. This method is not
// safe to be called from multiple go routines. The caller of this method must
// handle synchronization.
func (o *observer) getObserverID() int {
	if o.idCounter == math.MaxInt {
		o.idCounter = 0
	}
	o.idCounter = o.idCounter + 1
	return o.idCounter
}

// NewTipUpdater creates a new TipUpdater with the given
// TipUpdaterConfiguration.
func NewTipUpdater(config *TipUpdaterConfiguration) *TipUpdater {
	if config != nil {
		config = DefaultTipUpdaterConfig
	}
	return &TipUpdater{
		config: config,
	}
}

// Subscribe subscribes to notifications about newly fetched tip updates. It
// returns the channel with the notifications, and a functions to cancel the
// subscription.
func (tu *TipUpdater) Subscribe() (<-chan struct{}, CancelFunc) {
	done := make(chan struct{})
	subChan := make(chan struct{})
	switchChan := make(chan struct{}, 1)
	go func() {
		for {
			select {
			case <-switchChan:
				subChan <- struct{}{}
				break
			case <-done:
				close(switchChan)
				close(subChan)
				return
			}
		}
	}()
	id := tu.observer.newListener(switchChan)
	return subChan, func() {
		tu.observer.removeListener(id)
		done <- struct{}{}
		close(done)
	}
}

func (o *observer) newListener(c chan<- struct{}) int {
	o.lock.Lock()
	defer o.lock.Unlock()
	t := &tipListener{
		id: o.getObserverID(),
		c:  c,
	}
	o.listeners = append(o.listeners, t)
	return t.id
}

func (o *observer) removeListener(id int) {
	o.lock.Lock()
	defer o.lock.Unlock()
	listeners := make([]*tipListener, 0)
	for _, l := range o.listeners {
		if l.id != id {
			listeners = append(listeners, l)
		}
	}
	o.listeners = listeners
}

// notify sends a signal to all the subscribed observers.
func (o *observer) notify() {
	o.lock.RLock()
	defer o.lock.RUnlock()
	for _, l := range o.listeners {
		l.c <- struct{}{}
	}
}

// GetTip fetches the last gathered tip from this TipUpdater.
func (tu *TipUpdater) GetTip() *chain.Tip {
	tu.tip.lock.RLock()
	defer tu.tip.lock.RUnlock()
	return tu.tip.value
}

// Run runs the TipUpdater using the given chain.Backend. The run of this method
// can be canceled over the given context. Otherwise, this method is running
// infinitely.
func (tu *TipUpdater) Run(ctx context.Context, backend chain.Backend) {
	gather := func(ctx context.Context, backend chain.Backend) {
		tip, err := backend.GetLatestBlock(ctx)
		if err != nil {
			log.Errorf("tip cpuldn't be gathered: %s", err.Error())
		} else {
			log.Infof("fetched the tip (%d,%d) with hash=%s (minted at %s)",
				tip.Epoch, tip.SlotInEpoch, tip.Hash,
				time.Unix(int64(tip.Timestamp), 0))
			tu.tip.lock.Lock()
			defer tu.tip.lock.Unlock()
			tu.tip.value = tip
			go tu.observer.notify()
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
