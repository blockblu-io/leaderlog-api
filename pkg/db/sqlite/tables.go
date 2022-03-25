package sqlite

import (
	"context"
	"database/sql"
)

func createTables(sqlDB *sql.DB) error {
	ctx := context.Background()
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	err = createLeaderLogTable(tx, ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	err = createMintedBlockTable(tx, ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	err = createAssignedBlockTable(tx, ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func createLeaderLogTable(tx *sql.Tx, ctx context.Context) error {
	sqlStmt := `
CREATE TABLE LeaderLog (
	epoch INTEGER NOT NULL PRIMARY KEY,
	poolID TEXT NOT NULL,
	expectedBlockNr REAL NOT NULL,
	maxPerformance REAL NOT NULL
);
`
	_, err := tx.ExecContext(ctx, sqlStmt)
	return err
}

func createMintedBlockTable(tx *sql.Tx, ctx context.Context) error {
	sqlStmt := `
CREATE TABLE MintedBlock (
	id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
	epoch INTEGER NOT NULL,
	slotNr INTEGER NOT NULL,
	slotInEpochNr INTEGER NOT NULL,
	hash TEXT NOT NULL,
	height INTEGER NOT NULL,
	poolID TEXT NOT NULL
);
`
	_, err := tx.ExecContext(ctx, sqlStmt)
	return err
}

func createAssignedBlockTable(tx *sql.Tx, ctx context.Context) error {
	sqlStmt := `
CREATE TABLE AssignedBlock (
	epoch INTEGER NOT NULL,
	no INTEGER NOT NULL,
	slotNr INTEGER NOT NULL,
	slotInEpochNr INTEGER NOT NULL,
	timestamp INTEGER NOT NULL,
	status INTEGER DEFAULT 0,
	relevant INTEGER,
	PRIMARY KEY(epoch, no),
	FOREIGN KEY (epoch) REFERENCES LeaderLog(epoch) ON DELETE CASCADE,
	FOREIGN KEY (relevant) REFERENCES MintedBlock(id) ON DELETE CASCADE
);
`
	_, err := tx.ExecContext(ctx, sqlStmt)
	return err
}
