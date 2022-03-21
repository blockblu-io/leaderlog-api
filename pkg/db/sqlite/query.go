package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/blockblu-io/leaderlog-api/pkg/db"
	log "github.com/sirupsen/logrus"
)

// queryAndScanLeaderLogIDs queries for assigned blocks with the specified query
// and scans the result set. If the scanning has been successful, then an array of
// leader log IDs is returned. Otherwise, an error will be returned, if the querying
// or the scanning fails.
func (l *SQLiteDB) queryAndScanLeaderLogIDs(ctx context.Context, query string, args ...interface{}) ([]uint, error) {
	rows, err := l.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	epochs := make([]uint, 0)
	for rows.Next() {
		var epoch uint
		err = rows.Scan(&epoch)
		if err != nil {
			return nil, err
		}
		epochs = append(epochs, epoch)
	}
	return epochs, err
}

func (l *SQLiteDB) GetRegisteredEpochs(ctx context.Context, ordering db.Ordering, limit uint) ([]uint, error) {
	orderingString := "ASC"
	if ordering == db.OrderingDesc {
		orderingString = "DESC"
	}
	query := fmt.Sprintf(`SELECT epoch FROM LeaderLog ORDER BY epoch %s LIMIT ?`, orderingString)
	ids, err := l.queryAndScanLeaderLogIDs(ctx, query, limit)
	if err != nil {
		log.Errorf("querying for the registered epochs failed: %s", err.Error())
		return nil, db.ReadError
	}
	return ids, nil
}

// queryAndScanAssignedBlocksWithMintedBlock queries for assigned blocks with the
// specified query and scans the result set. If the scanning has been successful,
// then an array of assigned blocks is returned. Otherwise, an error will be
// returned, if the querying or the scanning fails.
func (l *SQLiteDB) queryAndScanAssignedBlocksWithMintedBlock(ctx context.Context, query string,
	args ...interface{}) ([]db.AssignedBlock, error) {
	rows, err := l.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	blocks := make([]db.AssignedBlock, 0)
	for rows.Next() {
		block := db.AssignedBlock{}
		var id, epoch, epochSlot, slot, height sql.NullInt64
		var hash, poolID sql.NullString
		err = rows.Scan(&block.Epoch, &block.No, &block.Slot, &block.EpochSlot, &block.Timestamp, &block.Status,
			&id, &epoch, &epochSlot, &slot, &hash, &height, &poolID)
		if err != nil {
			return nil, err
		}
		if id.Valid {
			mID := uint(id.Int64)
			mEpoch := uint(epoch.Int64)
			mEpochSlot := uint(epochSlot.Int64)
			mSlot := uint(slot.Int64)
			mHeight := uint(height.Int64)
			block.RelevantBlock = &db.MintedBlock{
				ID:        &mID,
				Epoch:     mEpoch,
				EpochSlot: mEpochSlot,
				Slot:      mSlot,
				Hash:      hash.String,
				Height:    mHeight,
				PoolID:    poolID.String,
			}
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

// queryAndScanAssignedBlocks queries for assigned blocks with the specified query
// and scans the result set. If the scanning has been successful, then an array of
// assigned blocks is returned. Otherwise, an error will be returned, if the querying
// or the scanning fails.
func (l *SQLiteDB) queryAndScanAssignedBlocks(ctx context.Context, query string,
	args ...interface{}) ([]db.AssignedBlock, error) {
	rows, err := l.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	blocks := make([]db.AssignedBlock, 0)
	for rows.Next() {
		block := db.AssignedBlock{}
		err = rows.Scan(&block.Epoch, &block.No, &block.Slot, &block.EpochSlot, &block.Timestamp, &block.Status)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

// queryAndScanLeaderLogs queries for a specific leaderlog with the given query and scans the
// result set for one entry. If the scanning has been succefuly, this one leader log will be
// returned. Otherwise, an error will be returned, if the scanning or querying failed.
func (l *SQLiteDB) queryAndScanLeaderLogs(ctx context.Context, q string, args ...interface{}) (*db.LeaderLog, error) {
	logRows, err := l.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer logRows.Close()
	var leaderLog *db.LeaderLog
	if logRows.Next() {
		leaderLog = &db.LeaderLog{}
		err = logRows.Scan(&leaderLog.Epoch, &leaderLog.PoolID, &leaderLog.ExpectedBlockNumber,
			&leaderLog.MaxPerformance)
		if err != nil {
			return nil, err
		}
	}
	return leaderLog, nil
}

func (l *SQLiteDB) GetLeaderLog(ctx context.Context, epoch uint) (*db.LeaderLog, error) {
	leaderLog, err := l.queryAndScanLeaderLogs(ctx,
		`SELECT epoch, poolID, expectedBlockNr, maxPerformance FROM LeaderLog WHERE epoch = ?;`, epoch)
	if err != nil {
		log.Errorf("querying the leaderlog of epoch=%d failed: %s", epoch, err.Error())
		return nil, db.ReadError
	}
	if leaderLog != nil {
		blocks, err := l.queryAndScanAssignedBlocks(ctx, `
SELECT epoch, no, slotNr, slotInEpochNr, timestamp, status FROM AssignedBlock
WHERE epoch = ?
ORDER BY no ASC;
`, epoch)
		if err != nil {
			log.Errorf("querying the blocks for leaderlog of epoch=%d failed: %s", leaderLog.Epoch, err.Error())
			return nil, db.ReadError
		}
		leaderLog.Blocks = blocks
	}
	return leaderLog, nil
}

func (l *SQLiteDB) GetAssignedBlocksAfterNow(ctx context.Context) ([]db.AssignedBlock, error) {
	now := time.Now()
	blocks, err := l.queryAndScanAssignedBlocks(ctx, `
SELECT epoch, no, slotNr, slotInEpochNr, timestamp, status FROM AssignedBlock
WHERE timestamp > ?
ORDER BY no ASC;
`, now)
	if err != nil {
		log.Errorf("querying the blocks after now=%v failed: %s", now, err.Error())
		return nil, db.ReadError
	}
	return blocks, nil
}

func (l *SQLiteDB) GetAssignedBlocksBeforeNow(ctx context.Context, epoch uint) ([]db.AssignedBlock, error) {
	now := time.Now()
	blocks, err := l.queryAndScanAssignedBlocksWithMintedBlock(ctx, `
SELECT a.epoch, a.no, a.slotNr, a.slotInEpochNr, a.timestamp, a.status, m.id, m.epoch, m.slotNr, m.slotInEpochNr,
	m.hash, m.height, m.poolID
FROM AssignedBlock a LEFT JOIN MintedBlock m on a.relevant = m.id
WHERE a.epoch = ? and a.timestamp <= ?
ORDER BY no ASC;
`, epoch, now)
	if err != nil {
		log.Errorf("querying the blocks of epoch=%d before now=%v failed: %s", epoch, now, err.Error())
		return nil, db.ReadError
	}
	return blocks, nil
}

func (l *SQLiteDB) GetAssignedBlocksWithStatusBeforeNow(ctx context.Context, status db.BlockStatus,
	offset, limit uint) ([]db.AssignedBlock, error) {
	now := time.Now()
	blocks, err := l.queryAndScanAssignedBlocks(ctx, `
SELECT epoch, no, slotNr, slotInEpochNr, timestamp, status FROM AssignedBlock
WHERE timestamp <= ? and status = ?
ORDER BY timestamp ASC
LIMIT ?
OFFSET ?;
`, now, status, limit, offset)
	if err != nil {
		log.Errorf("querying the blocks (%d,%d) with status=%d before now=%v failed: %s", offset, limit, status,
			now, err.Error())
		return nil, db.ReadError
	}
	return blocks, nil
}
