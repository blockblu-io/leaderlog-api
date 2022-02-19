package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/blockblu-io/leaderlog-api/pkg/db"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"time"
)

type SQLiteDB struct {
	db *sql.DB
}

// NewSQLiteDB opens a new SQLite database. This method should only
// be called once. It returns an DB instance with which the database
// can be queried, or an error, if opening the database has failed.
func NewSQLiteDB(path string) (db.DB, error) {
	dbFilePath := filepath.Join(path, "sql.db")
	needsToBeInitialized, err := prepareFile(dbFilePath)
	if err != nil {
		return nil, err
	}
	sqlDB, err := sql.Open("sqlite3", dbFilePath)
	if err != nil {
		return nil, err
	}
	if needsToBeInitialized {
		err = createTables(sqlDB)
		if err != nil {
			return nil, err
		}
	}
	return &SQLiteDB{
		db: sqlDB,
	}, nil
}

// prepareFile prepares the database file for sqlite such
// that it can be properly used and accessed. This method
// returns a boolean indicating whether the database file
// was newly created and hence, needs to be initialized or
// not. "True" will be returned, if it needs to be
// initialized. An error will be instead returned, if the
// path cannot be prepared for SQLite for some reason.
func prepareFile(dbFilePath string) (bool, error) {
	err := os.MkdirAll(filepath.Dir(dbFilePath), 0755)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(dbFilePath)
	needsToBeInitialized := errors.Is(err, os.ErrNotExist)
	if needsToBeInitialized {
		_, err := os.Create(dbFilePath)
		return true, err
	} else {
		return false, nil
	}
}

func (l *SQLiteDB) GetRegisteredEpochs(ctx context.Context, ordering db.Ordering, limit uint) ([]uint, error) {
	orderingString := "ASC"
	if ordering == db.OrderingDesc {
		orderingString = "DESC"
	}
	rows, err := l.db.QueryContext(ctx, fmt.Sprintf(`SELECT epoch FROM LeaderLog ORDER BY epoch %s LIMIT ?`,
		orderingString), limit)
	if err != nil {
		log.Errorf("the list of registered epochs couldn't be queried (%v,%d): %s", ordering, limit, err.Error())
		return nil, db.ReadError
	}
	epochs := make([]uint, 0)
	for rows.Next() {
		var epoch uint
		err = rows.Scan(&epoch)
		if err != nil {
			log.Errorf("the list of registered epochs couldn't be traversed (%v,%d): %s", ordering, limit,
				err.Error())
			return nil, db.ReadError
		}
		epochs = append(epochs, epoch)
	}
	return epochs, nil
}

func (l *SQLiteDB) GetLeaderLog(ctx context.Context, epoch uint) (*db.LeaderLog, error) {
	logRows, err := l.db.QueryContext(ctx, `SELECT epoch, poolID, expectedBlockNr, maxPerformance FROM LeaderLog WHERE epoch = ?;`, epoch)
	if err != nil {
		log.Errorf("querying the leaderlog of epoch '%d' failed: %s", epoch, err.Error())
		return nil, db.ReadError
	}
	defer logRows.Close()
	var leaderLog *db.LeaderLog
	if logRows.Next() {
		leaderLog = &db.LeaderLog{}
		err = logRows.Scan(&leaderLog.Epoch, &leaderLog.PoolID, &leaderLog.ExpectedBlockNumber,
			&leaderLog.MaxPerformance)
		if err != nil {
			log.Errorf("parsing leaderlog query result of epoch '%d' failed: %s", epoch, err.Error())
			return nil, db.ReadError
		}
	}
	if leaderLog != nil {
		blockRows, err := l.db.QueryContext(ctx, `
SELECT epoch, no, slotNr, slotInEpochNr, timestamp, status FROM AssignedBlock
WHERE epoch = ?
ORDER BY no ASC;
`, epoch)
		if err != nil {
			log.Errorf("querying assigned blocks to leaderlog of epoch '%d' failed: %s", epoch, err.Error())
			return nil, db.ReadError
		}
		for blockRows.Next() {
			blockAssignment := db.AssignedBlock{}
			err := blockRows.Scan(&blockAssignment.Epoch, &blockAssignment.No, &blockAssignment.Slot,
				&blockAssignment.EpochSlot, &blockAssignment.Timestamp, &blockAssignment.Status)
			if err != nil {
				log.Errorf("parsing the query result of assigned blocks to leaderlog of epoch '%d' failed: %s",
					epoch, err.Error())
				return nil, db.ReadError
			}
			leaderLog.Blocks = append(leaderLog.Blocks, &blockAssignment)
		}
	}
	return leaderLog, err
}

func (l *SQLiteDB) GetAssignedBlocksBeforeNow(ctx context.Context, epoch uint) ([]db.AssignedBlock, error) {
	now := time.Now()
	rows, err := l.db.QueryContext(ctx, `
SELECT epoch, no, slotNr, slotInEpochNr, timestamp, status FROM AssignedBlock
WHERE epoch = ? and timestamp <= ?
ORDER BY no ASC;
`, epoch, now)
	if err != nil {
		log.Errorf("querying the assigned blocks before %v of epoch '%d' failed: %s", now, epoch, err.Error())
		return nil, db.ReadError
	}
	var blocks []db.AssignedBlock = nil
	for rows.Next() {
		block := db.AssignedBlock{}
		err := rows.Scan(&block.Epoch, &block.No, &block.Slot, &block.EpochSlot, &block.Timestamp, &block.Status)
		if err != nil {
			log.Errorf("traversing the assigned blocks before %v of epoch '%d' failed: %s", now, epoch,
				err.Error())
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return blocks, err
}

func (l *SQLiteDB) GetAssignedBlocksWithStatusBeforeNow(ctx context.Context,
	status db.BlockStatus, offset, limit uint) ([]db.AssignedBlock, error) {
	now := time.Now()
	rows, err := l.db.QueryContext(ctx, `
SELECT epoch, no, slotNr, slotInEpochNr, timestamp, status FROM AssignedBlock
WHERE timestamp <= ? and status = ?
ORDER BY no ASC
LIMIT ?
OFFSET ?;
`, now, status, limit, offset)
	if err != nil {
		log.Errorf("querying all assigned blocks before now (%s) with status=%d failed: %s", now, status,
			err.Error())
		return nil, err
	}
	blocks := make([]db.AssignedBlock, 0)
	for rows.Next() {
		block := db.AssignedBlock{}
		err := rows.Scan(&block.Epoch, &block.No, &block.Slot, &block.EpochSlot, &block.Timestamp, &block.Status)
		if err != nil {
			log.Errorf("traversing all assigned blocks before now (%s) with status=%d failed: %s", now, status,
				err.Error())
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return blocks, err
}

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
	return nil
}

func (l *SQLiteDB) Close() error {
	return l.db.Close()
}
