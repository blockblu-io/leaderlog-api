package syncer

import (
	"context"
	"github.com/blockblu-io/leaderlog-api/pkg/chain"
	"github.com/blockblu-io/leaderlog-api/pkg/db"
	log "github.com/sirupsen/logrus"
)

// Syncer is a service, which scans for blocks that
// are planned for the past and their status is still
// db.NotMinted. Moreover, it queries to chain to check
// for the correct status and store it in the db.DB.
type Syncer struct {
	poolID        string
	backend       chain.Backend
	db            db.DB
	pastBlockChan syncChannel
}

// syncChannel is a buffered channel that contains all
// assigned blocks for which the status has to be updated.
type syncChannel struct {
	blocks     chan db.AssignedBlock
	lastSlot   uint
	bufferSize int
}

// NewSyncer is creating a new Syncer for the pool with the given ID
// in hex format. The given backend is used for querying the chain, and
// the given db.DB is used to query as well as update the blocks and their
// status
func NewSyncer(poolID string, backend chain.Backend, idb db.DB) *Syncer {
	blockChannel := make(chan db.AssignedBlock, 5)
	return &Syncer{
		poolID:  poolID,
		backend: backend,
		db:      idb,
		pastBlockChan: syncChannel{
			blocks:     blockChannel,
			bufferSize: 5,
		},
	}
}

// Run starts this sync service. This method is blocking.
func (s *Syncer) Run(ctx context.Context) {
	log.Infof("started to sync blocks using '%s'", s.backend.Name())
	scanner := s.NewScanner()
	go scanner.Run(ctx)
	for {
		select {
		case b := <-s.pastBlockChan.blocks:
			if b.Slot >= s.pastBlockChan.lastSlot {
				s.processBlock(ctx, b)
				s.pastBlockChan.lastSlot = b.Slot
			}
			break
		case <-ctx.Done():
			return
		}
	}
}

// processBlock gathers the status of the assigned block and updates the
// status in the database.
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

// getStatusOfBlock gathers the status of an assigned block with the given slot
// number. The slot and the neighbourhood are respectively scanned for a minted
// block. A minted block or its absence leads to a conclusion about the status.
// This method returns the gathered status and found minted block. The returned
// minted block is nil, if the db.BlockStatus is db.GHOSTED.
//
// An error will be returned, if the slot and neighbourhood couldn't be scanned
// correctly.
func (s *Syncer) getStatusOfBlock(ctx context.Context, slot uint) (db.BlockStatus, *chain.MintedBlock, error) {
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

// Close is closing this syncer.
func (s *Syncer) Close() {
	close(s.pastBlockChan.blocks)
}
