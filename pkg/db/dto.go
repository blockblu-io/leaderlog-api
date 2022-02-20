package db

import "time"

// LeaderLog is a list of blocks assigned to a certain
// pool and for a certain epoch.
type LeaderLog struct {
	// PoolID is the id in hex format of the pool for
	// which this leader log was created.
	PoolID string
	// Epoch is the epoch for which the leader log is
	// created.
	Epoch uint
	// Blocks is a list of blocks that have been assigned
	// to the pool in this epoch.
	Blocks []AssignedBlock
	// ExpectedBlockNumber
	ExpectedBlockNumber float32
	// MaxPerformance is the maximal possible performance
	// in the epoch given this leader log.
	MaxPerformance float32
}

// BlockStatus refers to the status of an assigned block.
type BlockStatus uint

const (
	NotMinted        BlockStatus = 0
	Minted                       = 1
	DoubleAssignment             = 2
	HeightBattle                 = 3
	GHOSTED                      = 4
)

// AssignedBlock is a block assigned to a pool in an
// epoch, which is part of an overall leader log.
type AssignedBlock struct {
	// Epoch is the epoch for which the block has been
	// scheduled.
	Epoch uint
	// No is the unique number of the block in an overall
	// leader log.
	No uint
	// EpochSlot is the slot number for which the block has
	// been scheduled. The number is counted from the start
	// of the epoch.
	EpochSlot uint
	// Slot is the slot number fow which the block has been
	// scheduled. The number is counted from the chain`s
	// inception.
	Slot uint
	// Timestamp is the exact time for which the block has
	// been scheduled.
	Timestamp time.Time
	// Status is the current status of this scheduled block.
	Status BlockStatus
}
