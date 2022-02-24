package syncer

import (
	"context"
	"github.com/blockblu-io/leaderlog-api/pkg/db"
	log "github.com/sirupsen/logrus"
	"math"
	"time"
)

type Scanner struct {
	syncer   *Syncer
	listener chan db.ObserverMessage
}

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
	go sc.scanPastBlocks(ctx)
	nextBlockChan := sc.newNextBlockChannel(ctx)
	for {
		select {
		case msg := <-sc.listener:
			if msg.Code == db.ObserveNewLeaderLog {
				go sc.scanPastBlocks(ctx)
			}
			break
		case <-nextBlockChan:
			go sc.scanPastBlocks(ctx)
			break
		case <-ctx.Done():
			return
		}
	}
}

func (sc *Scanner) scanPastBlocks(ctx context.Context) {
	log.Infof("scanning past blocks that haven't been updated")
	n := 0
	for {
		blocks, err := sc.syncer.db.GetAssignedBlocksWithStatusBeforeNow(ctx, db.NotMinted, uint(n),
			uint(sc.syncer.pastBlockChan.bufferSize))
		if err != nil {
			log.Errorf("couldn't scan the unsynced blcoks: %s", err.Error())
			return
		}
		n += len(blocks)
		log.Infof("found [%d] non-updated blocks", n)
		for _, block := range blocks {
			sc.syncer.pastBlockChan.blocks <- block
		}
		if len(blocks) != sc.syncer.pastBlockChan.bufferSize {
			break
		}
	}
}

func (sc *Scanner) newNextBlockChannel(ctx context.Context) <-chan struct{} {
	channel := make(chan struct{})
	go sc.next(ctx, channel, 0, nil)
	return channel
}

func (sc *Scanner) next(ctx context.Context, channel chan<- struct{}, depth int, block *db.AssignedBlock) {
	wait := time.Hour
	if block == nil {
		if depth <= 6 {
			wait = time.Duration(math.Pow(2.0, float64(depth))) * time.Minute
		}
		blocks, err := sc.syncer.db.GetAssignedBlocksAfterNow(ctx)
		if err == nil && len(blocks) != 0 {
			block = &blocks[0]
			wait = block.Timestamp.Add(time.Minute).Sub(time.Now())
			log.Infof("waiting %v for the next block", wait)
		} else {
			if err != nil {
				log.Errorf("couldn't query for next block: %s", err.Error())
			} else {
				log.Warnf("couldn't find any next block in the database")
			}
		}
	} else {
		wait = block.Timestamp.Add(time.Minute).Sub(time.Now())
	}
	select {
	case <-time.Tick(wait):
		if block != nil {
			pubNextBlock := func() {
				channel <- struct{}{}
			}
			go pubNextBlock()
			sc.next(ctx, channel, 0, nil)
		} else {
			sc.next(ctx, channel, depth+1, nil)
		}
		break
	case msg := <-sc.listener:
		if msg.Code == db.ObserveNewLeaderLog {
			sc.next(ctx, channel, depth+1, nil)
		} else {
			sc.next(ctx, channel, depth+1, block)
		}
		break
	case <-ctx.Done():
		break
	}
}
