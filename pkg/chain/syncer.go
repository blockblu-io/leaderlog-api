package chain

import (
	"context"
	"github.com/blockblu-io/leaderlog-api/pkg/db"
	log "github.com/sirupsen/logrus"
)

// Syncer is a service, which scans for blocks that
// are planned for the past and their status is still
// db.NotMinted. Moreover, it queries to chain to check
// for the correct status and store it in the db.DB.
type Syncer struct {
	poolID    string
	backend   Backend
	db        db.DB
	blockChan syncChannel
}

// syncChannel is a buffered channel that contains all
// assigned blocks for which the status has to be updated.
type syncChannel struct {
	blocks     chan db.AssignedBlock
	bufferSize int
}

// NewSyncer is creating a new Syncer for the pool with the given ID
// in hex format. The given backend is used for querying the chain, and
// the given db.DB is used to query as well as update the blocks and their
// status
func NewSyncer(poolID string, backend Backend, idb db.DB) *Syncer {
	blockChannel := make(chan db.AssignedBlock, 5)
	return &Syncer{
		poolID:  poolID,
		backend: backend,
		db:      idb,
		blockChan: syncChannel{
			blocks:     blockChannel,
			bufferSize: 5,
		},
	}
}

// Run starts this sync service. This method is blocking.
func (s *Syncer) Run(ctx context.Context) {
	log.Infof("started to sync blocks using '%s'", s.backend.Name())
	go s.scanPastBlocks(ctx)
	for {
		select {
		case b := <-s.blockChan.blocks:
			s.processBlock(ctx, b)
		case <-ctx.Done():
			return
		}
	}
}

func (s *Syncer) scanPastBlocks(ctx context.Context) {
	log.Infof("scanning past blocks that haven't been updated ...")
	n := 0
	for {
		blocks, err := s.db.GetAssignedBlocksWithStatusBeforeNow(ctx, db.NotMinted, uint(n),
			uint(s.blockChan.bufferSize))
		if err != nil {
			log.Errorf("couldn't scan the unsynced blcoks: %s", err.Error())
			return
		}
		n += len(blocks)
		log.Infof("scanned [%d] non-updated blocks", n)
		for _, block := range blocks {
			s.blockChan.blocks <- block
		}
		if len(blocks) != s.blockChan.bufferSize {
			break
		}
	}
}

func (s *Syncer) processBlock(ctx context.Context, block db.AssignedBlock) {
	log.Infof("processing block at (%d,%d) with no=%d for pool-id=%s", block.Epoch, block.EpochSlot,
		block.No, s.poolID)
	status, _, err := s.getStatusOfBlock(ctx, block.Slot)
	if err != nil {
		return
	}
	err = s.db.UpdateStatusForAssignment(ctx, block.Epoch, block.No, status)
	if err != nil {
		log.Errorf("couldn't update the status for block (%d,%d): %s", block.Epoch, block.No, err.Error())
	}
}

func (s *Syncer) getStatusOfBlock(ctx context.Context, slot uint) (db.BlockStatus, *MintedBlock, error) {
	mintedBlock, err := s.backend.GetMintedBlock(ctx, slot)
	if err != nil {
		return 0, nil, err
	}
	if mintedBlock != nil {
		if mintedBlock.Pool.HexID == s.poolID {
			return db.Minted, mintedBlock, nil
		} else {
			return db.DoubleAssignment, mintedBlock, nil
		}
	}
	mintedBlock, err = s.backend.TraverseAround(ctx, slot, 5)
	if err != nil {
		return 0, nil, err
	}
	if mintedBlock != nil {
		return db.HeightBattle, mintedBlock, nil
	}
	return db.GHOSTED, mintedBlock, nil
}
