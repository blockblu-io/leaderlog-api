package sqlite

import (
	"context"
	"github.com/blockblu-io/leaderlog-api/pkg/db"
	log "github.com/sirupsen/logrus"
)

func (l *SQLiteDB) WriteLeaderLog(ctx context.Context, leaderLog *db.LeaderLog) error {
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		log.Errorf("couldn't start a transaction to write the leaderlog of epoch '%d': %s", leaderLog.Epoch,
			err.Error())
		return db.WriteError
	}
	_, err = tx.Exec(`
INSERT INTO LeaderLog (epoch, poolID, expectedBlockNr, maxPerformance) VALUES (?, ?, ?, ?)
`, leaderLog.Epoch, leaderLog.PoolID, leaderLog.ExpectedBlockNumber, leaderLog.MaxPerformance)
	if err != nil {
		_ = tx.Rollback()
		log.Errorf("inserting the leaderlog of epoch '%d' failed: %s", leaderLog.Epoch, err.Error())
		return db.WriteError
	}
	insertAssignmentStmt, err := tx.Prepare(`
INSERT INTO AssignedBlock (epoch, no, slotNr, slotInEpochNr, timestamp) VALUES (?,?,?,?,?);
`)
	if err != nil {
		_ = tx.Rollback()
		log.Errorf("preparing the query for assigned block insertion for epoch '%d' failed: %s", leaderLog.Epoch,
			err.Error())
		return db.WriteError
	}
	for _, block := range leaderLog.Blocks {
		_, err = insertAssignmentStmt.ExecContext(ctx, leaderLog.Epoch, block.No, block.Slot, block.EpochSlot,
			block.Timestamp)
		if err != nil {
			_ = tx.Rollback()
			log.Errorf("assigned block insertion for epoch '%d' failed: %s", leaderLog.Epoch,
				err.Error())
			return db.WriteError
		}
	}
	err = tx.Commit()
	if err != nil {
		return db.WriteError
	}
	go l.obv.Pub(db.ObserverMessage{Code: db.ObserveNewLeaderLog, Response: leaderLog.Epoch})
	return nil
}

func (l *SQLiteDB) UpdateStatusForAssignment(ctx context.Context, epoch, no uint, status db.BlockStatus) error {
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		log.Errorf("couldn't start a transaction to update status of block (%d,%d): %s", epoch, no,
			err.Error())
		return db.WriteError
	}
	_, err = tx.ExecContext(ctx, `
UPDATE AssignedBlock SET status = ? WHERE epoch = ? and no = ?;
`, status, epoch, no)
	if err != nil {
		_ = tx.Rollback()
		log.Errorf("setting the status=%v of block (%d,%d) failed: %s", status, epoch, no, err.Error())
		return db.WriteError
	}
	err = tx.Commit()
	if err != nil {
		return db.WriteError
	}
	go l.obv.Pub(db.ObserverMessage{Code: db.ObserveUpdatedBlockStatus, Response: []uint{epoch, no}})
	return nil
}
