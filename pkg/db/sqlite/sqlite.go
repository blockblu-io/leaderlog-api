package sqlite

import (
	"database/sql"
	"errors"
	"github.com/blockblu-io/leaderlog-api/pkg/db"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"path/filepath"
)

// SQLiteDB is a db.DB making use of sqlite3 database.
type SQLiteDB struct {
	db  *sql.DB
	obv *db.Observer
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
		db:  sqlDB,
		obv: &db.Observer{},
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

func (l *SQLiteDB) Observer() *db.Observer {
	return l.obv
}

func (l *SQLiteDB) Close() error {
	l.obv.Close()
	return l.db.Close()
}
