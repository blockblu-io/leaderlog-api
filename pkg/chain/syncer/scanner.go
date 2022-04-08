package syncer

import (
	"context"
	"time"

	"github.com/blockblu-io/leaderlog-api/pkg/db"
	log "github.com/sirupsen/logrus"
)

const (
	bufferSize uint = 100
)

// Scanner offers the ability to scan for blocks that have to be synced.
type Scanner struct {
	syncer   *Syncer
	listener chan db.ObserverMessage
}

// NewScanner creates a new scanner for this Syncer.
func (s *Syncer) NewScanner() *Scanner {
	obv := s.db.Observer()
	listener := make(chan db.ObserverMessage)
	obv.Sub(listener)
	return &Scanner{
		syncer:   s,
		listener: listener,
	}
}

func (sc *Scanner) Run(ctx context.Context) {
	sc.scanPastBlocks(ctx)
	keepOn := true
	nextBlockChan := make(chan struct{})
	lookCtx, cancel := context.WithCancel(ctx)
	go sc.lookForNextBlock(lookCtx, nextBlockChan)
	for keepOn {
		select {
		case msg := <-sc.listener:
			if msg.Code == db.ObserveNewLeaderLog {
				go sc.scanPastBlocks(ctx)
				cancel()
				lookCtx, cancel = context.WithCancel(ctx)
				go sc.lookForNextBlock(lookCtx, nextBlockChan)
			}
			break
		case <-nextBlockChan:
			go sc.scanPastBlocks(ctx)
			break
		case <-ctx.Done():
			keepOn = false
		}
	}
	cancel()
}

func (sc *Scanner) lookForNextBlock(ctx context.Context, nChan chan struct{}) {
	blocks, err := sc.syncer.db.GetAssignedBlocksAfterNow(ctx)
	if err != nil {
		log.Errorf("couldn't scan the unsynced blcoks: %s",
			err.Error())
		return
	}
	if len(blocks) == 0 {
		log.Warnf("couldn't find any next block in the database")
		return
	}
	wait := blocks[0].Timestamp.Sub(time.Now())
	if wait < 0 {
		wait = 10 * time.Second
	}
	log.Infof("waiting %s for the next block", wait)
	timer := time.NewTimer(wait)
	select {
	case <-timer.C:
		nChan <- struct{}{}
	case <-ctx.Done():
		close(nChan)
		break
	}
}

func (sc *Scanner) scanPastBlocks(ctx context.Context) {
	log.Infof("scanning past blocks that haven't been updated")
	loaded := false
	for n := uint(0); !loaded; n += bufferSize {
		blocks, err := sc.syncer.db.GetAssignedBlocksWithStatusBeforeNow(ctx,
			db.NotMinted, n, bufferSize)
		if err != nil {
			log.Errorf("couldn't scan the unsynced blcoks: %s",
				err.Error())
			return
		}
		n += bufferSize
		for _, block := range blocks {
			sc.syncer.pastBlockChan <- block
		}
		if uint(len(blocks)) != bufferSize {
			loaded = true
		}
	}
}
