package dto

import (
	"encoding/json"
	"errors"
	"github.com/blockblu-io/leaderlog-api/pkg/db"
	"io"
	"io/ioutil"
	"time"
)

// LeaderLog is a list of assigned blocks for a certain pool
// and a certain epoch.
type LeaderLog struct {
	// PoolID is the unique id of the pool for which this log
	// was created.
	PoolID string `json:"poolId"`
	//Epoch is the epoch for which the leader log was created.
	Epoch uint `json:"epoch"`
	// Blocks is the list of assigned blocks for this leader log.
	Blocks []*AssignedBlock `json:"assignedSlots"`
	// ExpectedBlockNumber states the expected number of blocks based
	// on the active pool size of the pool for this epoch.
	ExpectedBlockNumber float32 `json:"epochSlotsIdeal"`
	// MaxPerformance is the maximally possible performance given this
	// assignment of blocks. It relates assigned number of blocks to the
	// expected one.
	MaxPerformance float32 `json:"maxPerformance"`
}

// ToPlain transforms this leader log object from the api package into
// a leader log object from the db package.
func (l *LeaderLog) ToPlain() *db.LeaderLog {
	blocks := make([]db.AssignedBlock, len(l.Blocks))
	for i, b := range l.Blocks {
		blocks[i] = b.ToPlain()
	}
	return &db.LeaderLog{
		PoolID:              l.PoolID,
		Epoch:               l.Epoch,
		Blocks:              blocks,
		ExpectedBlockNumber: l.ExpectedBlockNumber,
		MaxPerformance:      l.MaxPerformance,
	}
}

// AssignedBlock is a block that has been assigned to
// a certain pool for a certain epoch.
type AssignedBlock struct {
	// ID of the assigned block, which is unique for this epoch.
	No uint `json:"no"`
	// Slot is the slot number for which this block is assigned.
	// This number is counted from the beginning of the chain.
	Slot uint `json:"slot"`
	// EpochSlot is the slot number for which this block is assigned.
	// This number is counted from the beginning of the epoch.
	EpochSlot uint `json:"slotInEpoch"`
	// Timestamp is the timestamp of the starting time of the slot to
	// which the block is assigned.
	Timestamp time.Time `json:"at"`
}

// ToPlain transforms this assigned block object from the api package
// into the assigned block object from the db package.
func (a *AssignedBlock) ToPlain() db.AssignedBlock {
	return db.AssignedBlock{
		No:        a.No,
		Slot:      a.Slot,
		EpochSlot: a.EpochSlot,
		Timestamp: a.Timestamp,
	}
}

var (
	// ReadError is returned, when the leader log couldn't be read while parsing.
	ReadError = errors.New("couldn't read the leader log")
	// WriteError is returned, when the leader log couldn't be serialized and written.
	WriteError = errors.New("couldn't write the leader log")
	// ParsingError is returned, when an error occurred during the unmarshalling of the leader log.
	ParsingError = errors.New("couldn't parse the leader log properly")
)

// ParseLeaderLog parses the content of the give reader into
// a leader log object. If the passing fails, then a corresponding
// error will be returned. Otherwise, the parsed leader log is returned.
func ParseLeaderLog(reader io.Reader) (*LeaderLog, error) {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, ReadError
	}
	var leaderLog LeaderLog
	err = json.Unmarshal(data, &leaderLog)
	if err != nil {
		return nil, ParsingError
	}
	return &leaderLog, nil
}

// WriteLeaderLog serializes the given leader log into a JSON object
// and then writes the JSON object to the given writer. An error will
// be returned, if the serialization or the writing failed.
func WriteLeaderLog(log LeaderLog, writer io.Writer) error {
	data, err := json.Marshal(log)
	if err != nil {
		return WriteError
	}
	_, err = writer.Write(data)
	if err != nil {
		return WriteError
	}
	return nil
}
