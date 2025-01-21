package transactors

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/billowdev/clog"

	"gorm.io/gorm"
)

type TransactorImpl struct {
	db *gorm.DB
}

func (d *TransactorImpl) GetDatabaseConnection() *gorm.DB {
	return d.db
}

func (d *TransactorImpl) IsTransactionActive() bool {
	// Check if the connection pool is a transactional type
	_, ok := d.db.Statement.ConnPool.(*sql.Tx)
	return ok
}

// BeginTransaction implements IDatabasePorts.
func (d *TransactorImpl) BeginTransaction() (*gorm.DB, error) {
	tx := d.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	return tx, nil
}

func (d *TransactorImpl) BeginTransactionWithContext(ctx context.Context) (*gorm.DB, error) {
	tx := ExtractTx(ctx)
	if tx == nil {
		tx = d.db.Begin()
	}
	return tx, nil
}

// RollbackTransaction rolls back the transaction if it was started and returns any error encountered.
func (d *TransactorImpl) RollbackTransaction(tx *gorm.DB) error {
	if tx == nil {
		clog.Info(">>>>>>No transaction to rollback<<<<<<")
		return nil // No transaction to rollback
	}

	// Rollback the transaction
	if err := tx.Rollback().Error; err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	return nil
}

// CommitTransaction commits the transaction if it was started.
// If the commit fails, it attempts to rollback and returns any errors encountered.
func (d *TransactorImpl) CommitTransaction(tx *gorm.DB) error {
	if tx == nil {
		return nil // No transaction to commit
	}
	if tx.Error != nil {
		tx.Rollback()
		return tx.Error // If there was an error, return it
	}

	// Attempt to commit the transaction
	if err := tx.Commit().Error; err != nil {
		// If commit fails, attempt to rollback
		if rbErr := tx.Rollback().Error; rbErr != nil {
			return fmt.Errorf("failed to commit transaction: %w, and failed to rollback: %v", err, rbErr)
		}
		return fmt.Errorf("failed to commit transaction and rolled back: %w", err)
	}
	return nil
}

type ITransactors interface {
	IsTransactionActive() bool
	GetDatabaseConnection() *gorm.DB
	WithAtomicCommitCondition(ctx context.Context, commitCondition func() bool, timeout time.Duration, tFunc func(txCtx context.Context) error) error
	WithAtomicBasicCondition(ctx context.Context, isCommit bool, timeout time.Duration, tFunc func(txCtx context.Context) error) error
	WithAtomic(ctx context.Context, timeout time.Duration, tFunc func(txCtx context.Context) error) error

	BeginTransaction() (*gorm.DB, error)
	BeginTransactionWithContext(ctx context.Context) (*gorm.DB, error)
	RollbackTransaction(tx *gorm.DB) error
	CommitTransaction(tx *gorm.DB) error
}

func NewTransactorRepo(db *gorm.DB) ITransactors {
	return &TransactorImpl{db: db}
}
