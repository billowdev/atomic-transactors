package transactors

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type ITransactors interface {
	GetDatabaseConnection() *gorm.DB
	WithAtomicCommitCondition(ctx context.Context, commitCondition func() bool, timeout time.Duration, tFunc func(txCtx context.Context) error) error
	WithAtomicBasicCondition(ctx context.Context, isCommit bool, timeout time.Duration, tFunc func(txCtx context.Context) error) error
	WithAtomic(ctx context.Context, timeout time.Duration, tFunc func(txCtx context.Context) error) error
	BeginTransaction() (*gorm.DB, error)
	BeginTransactionWithContext(ctx context.Context) (*gorm.DB, error)
	RollbackTransaction(tx *gorm.DB) error
	CommitTransaction(tx *gorm.DB) error
}

func NewTransactors(db *gorm.DB) ITransactors {
	return &TransactorImpl{db: db}
}
