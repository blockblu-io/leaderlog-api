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

// Run runs this scanner. It checks for past blocks in the DB that have not
// yet been updated and waits for the next assigned block being in the past.
// This method is running infinitely unless the given context has been
// cancelled.
func (sc *Scanner) Run(ctx context.Context) {
	sc.scanPastBlocks(ctx)
	keepOn := true
	nextBlockChan := make(chan db.AssignedBlock)
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
		case block := <-nextBlockChan:
			go func() {
				block.Timestamp = time.Now()
				sc.syncer.pastBlockChan <- block
			}()
			cancel()
			lookCtx, cancel = context.WithCancel(ctx)
			go sc.lookForNextBlock(lookCtx, nextBlockChan)
			break
		case <-ctx.Done():
			keepOn = false
		}
	}
	cancel()
}

// lookForNextBlock looks for the next assigned block in the DB and then blocks
// until the next assigned block is in the past or the given context has been
// canceled. When the next assigned block is in the past, then this block will
// be pushed to the specified channel.
func (sc *Scanner) lookForNextBlock(ctx context.Context,
	nChan chan db.AssignedBlock) {

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
		wait = 0
	}
	wait = wait + 10*time.Second
	log.Infof("waiting %s for the next block", wait)
	timer := time.NewTimer(wait)
	select {
	case <-timer.C:
		nChan <- blocks[0]
	case <-ctx.Done():
		close(nChan)
		break
	}
}

// scanPastBlocks scans for the for past blocks in the DB that have not yet been
// updated.
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
