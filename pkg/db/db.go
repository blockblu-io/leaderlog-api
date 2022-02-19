package db

import (
	"context"
	"errors"
	"time"
)

// DB is an interface to store and query leader logs as well
// as to manage related data.
type DB interface {

	// GetRegisteredEpochs gets a list of all registered epochs.
	GetRegisteredEpochs(ctx context.Context, ordering Ordering, limit uint) ([]uint, error)

	// GetLeaderLog gets the leader log for the given epoch.
	GetLeaderLog(ctx context.Context, epoch uint) (*LeaderLog, error)

	// GetAssignedBlocksBeforeNow gets the assigned blocks that
	// have been planned before now for the given epoch.
	GetAssignedBlocksBeforeNow(ctx context.Context, epoch uint) ([]AssignedBlock, error)

	// GetAssignedBlocksWithStatusBeforeNow gets the assigned blocks that
	// have been planned before current time and have the given BlockStatus.
	GetAssignedBlocksWithStatusBeforeNow(ctx context.Context, status BlockStatus, offset, limit uint) ([]AssignedBlock, error)

	// UpdateStatusForAssignment updates the status for the block assignment of
	// the given epoch with the specified unique id called "no".
	UpdateStatusForAssignment(ctx context.Context, epoch, no uint, status BlockStatus) error

	// WriteLeaderLog writes the given list of assigned blocks
	// for the given epoch to the DB. If a leader log has already
	// written for this epoch, then the old leader log will be
	// overwritten.
	WriteLeaderLog(ctx context.Context, log *LeaderLog) error

	// Close closes this database and all connections.
	Close() error
}

type Ordering uint

const (
	OrderingAsc  Ordering = 0
	OrderingDesc          = 1
)

type LeaderLog struct {
	PoolID              string
	Epoch               uint
	Blocks              []*AssignedBlock
	ExpectedBlockNumber float32
	MaxPerformance      float32
}

type BlockStatus uint

const (
	NotMinted        BlockStatus = 0
	Minted                       = 1
	DoubleAssignment             = 2
	HeightBattle                 = 3
	GHOSTED                      = 4
)

type AssignedBlock struct {
	Epoch     uint
	No        uint
	Slot      uint
	EpochSlot uint
	Timestamp time.Time
	Status    BlockStatus
}

var (
	// ReadError is returned, when querying the database failed for some reason.
	ReadError = errors.New("read from the database failed")
	// WriteError is returned, when querying the database failed for some reason.
	WriteError = errors.New("write to the database failed")
)
