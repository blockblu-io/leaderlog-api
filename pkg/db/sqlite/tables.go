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
	epoch INT NOT NULL PRIMARY KEY,
	poolID VARCHAR(128) NOT NULL,
	expectedBlockNr DECIMAL(8,2) NOT NULL,
    maxPerformance DECIMAL(8,2) NOT NULL
);
`
	_, err := tx.ExecContext(ctx, sqlStmt)
	return err
}

func createAssignedBlockTable(tx *sql.Tx, ctx context.Context) error {
	sqlStmt := `
CREATE TABLE AssignedBlock (
	epoch INT NOT NULL, 
    no INT NOT NULL,
	slotNr INT NOT NULL,
	slotInEpochNr INT NOT NULL,
	timestamp Date NOT NULL,
	status INT DEFAULT 0,
	FOREIGN KEY (epoch) REFERENCES LeaderLog(epoch),
	PRIMARY KEY(epoch, no)
);
`
	_, err := tx.ExecContext(ctx, sqlStmt)
	return err
}
