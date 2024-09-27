package lib

import (
	"errors"
	"os"
	"path/filepath"
)

// DatabaseManager will manage multiple databases
type DatabaseManager struct {
	currentDB  string
	databasePath string
}

// NewDatabaseManager initializes a DatabaseManager with a base path
func NewDatabaseManager(basePath string) *DatabaseManager {
	return &DatabaseManager{
		databasePath: ensureTrailingSlash(basePath),
	}
}

// CreateDatabase creates a new database directory and files
func (dm *DatabaseManager) CreateDatabase(dbName string) error {
	dbDir := filepath.Join(dm.databasePath, dbName)
	if _, err := os.Stat(dbDir); !os.IsNotExist(err) {
		return errors.New("database already exists")
	}
	return os.MkdirAll(dbDir, 0755)
}

// DropDatabase removes a database directory and files
func (dm *DatabaseManager) DropDatabase(dbName string) error {
	dbDir := filepath.Join(dm.databasePath, dbName)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		return errors.New("database does not exist")
	}
	return os.RemoveAll(dbDir)
}

// UseDatabase sets the current database to be used
func (dm *DatabaseManager) UseDatabase(dbName string) error {
	dbDir := filepath.Join(dm.databasePath, dbName)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		return errors.New("database does not exist")
	}
	dm.currentDB = dbName
	return nil
}

// ShowDatabases lists all databases in the base path
func (dm *DatabaseManager) ShowDatabases() ([]string, error) {
	var databases []string
	files, err := os.ReadDir(dm.databasePath)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() {
			databases = append(databases, file.Name())
		}
	}
	return databases, nil
}

// CurrentDatabase returns the name of the currently used database
func (dm *DatabaseManager) CurrentDatabase() string {
	return dm.currentDB
}

// GetDatabasePath returns the full path to the current database
func (dm *DatabaseManager) GetDatabasePath() string {
	if dm.currentDB == "" {
		return ""
	}
	return filepath.Join(dm.databasePath, dm.currentDB)
}
