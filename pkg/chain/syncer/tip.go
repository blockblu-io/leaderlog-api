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
	listeners []*TipListener
}

// TipListener offers a channel that signals when the tip has been updated. If
// the listener will not be used anymore, call the unsubscribe method to free
// resources.
type TipListener struct {
	observer *observer
	id       int
	C        chan struct{}
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

// Subscribe subscribes to notifications about newly fetched tip updates.
func (tu *TipUpdater) Subscribe() *TipListener {
	tu.observer.lock.Lock()
	defer tu.observer.lock.Unlock()
	c := make(chan struct{})
	t := &TipListener{
		observer: &tu.observer,
		id:       tu.observer.getObserverID(),
		C:        c,
	}
	tu.observer.listeners = append(tu.observer.listeners, t)
	return t
}

// notify sends a signal to all the subscribed observers.
func (tu *TipUpdater) notify() {
	tu.observer.lock.RLock()
	defer tu.observer.lock.RUnlock()
	for _, l := range tu.observer.listeners {
		go func(l *TipListener) {
			l.C <- struct{}{}
		}(l)
	}
}

// Unsubscribe unsubscribes this tip listeners from getting notifications. This
// method frees resources and shall be called, if this TipListener isn't going
// to be used anymore.
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
			go tu.notify()
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
