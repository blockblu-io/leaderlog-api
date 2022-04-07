package chain

import (
	"context"
	"github.com/blockblu-io/leaderlog-api/pkg/db"
)

// StakePool is an object containing metadata information about a certain pool.
type StakePool struct {
	// Ticker is the short pool name.
	Ticker string
	// Name is the full pool name.
	Name string
	// HexID is the unique pool hash in hex format.
	HexID string
}

type Tip struct {
	// Height is the height number of this block.
	Height uint
	// Hash is the unique hash of this block.
	Hash string
	// Epoch is the epoch in which this block has been minted.
	Epoch uint
	// SlotInEpoch is the slot in the epoch in which the
	// block has been minted.
	SlotInEpoch uint
	// Slot is the slot in which the block has been minted, but the slot number
	// is counted from the inception of the chain.
	Slot uint
	// Timestamp is the Unix timestamp in seconds of the time at which this
	// block has been minted.
	Timestamp uint
}

// MintedBlock is an object containing information about a minted block.
type MintedBlock struct {
	Tip
	// Pool is the stake pool, which minted this block.
	Pool StakePool
}

func (b *MintedBlock) ToDTO() *db.MintedBlock {
	dto := db.MintedBlock{
		Epoch:     b.Epoch,
		EpochSlot: b.SlotInEpoch,
		Slot:      b.Slot,
		Hash:      b.Hash,
		Height:    b.Height,
		PoolID:    b.Pool.HexID,
	}
	return &dto
}

// Backend is an interface for querying the underlying blockchain.
type Backend interface {

	// Name returns the name of this backend.
	Name() string

	// GetLatestBlock queries for the latest minted block (i.e. the tip of the
	// chain).
	//
	// If querying the chain failed, an error will be returned instead.
	GetLatestBlock(ctx context.Context) (*Tip, error)

	// GetMintedBlock queries the chain looking at the given slot. If a block
	// has been minted for the given slot, then the minted block will be
	// returned. Otherwise, nil will be returned.
	//
	// If querying the chain failed, an error will be returned instead.
	GetMintedBlock(ctx context.Context, slot uint) (*MintedBlock, error)

	// TraverseAround queries the slots around the given slot searching for a
	// minted block. The first found minted block will be returned. Nil will be
	// returned, if no minted block could be found. The interval specifies the
	// size of search area around the slot. The specified (interval) number of
	// slots in both direction will be investigated.
	//
	// If querying the chain failed, an error will be returned instead.
	TraverseAround(ctx context.Context, slot uint,
		interval uint) (*MintedBlock, error)
}
