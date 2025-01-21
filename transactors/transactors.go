package transactors

import (
	"context"
	"fmt"

	"github.com/billowdev/clog"

	"gorm.io/gorm"
)

type TransactorImpl struct {
	db *gorm.DB
}

func (d *TransactorImpl) GetDatabaseConnection() *gorm.DB {
	return d.db
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
