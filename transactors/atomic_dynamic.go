package transactors

import (
	"context"
	"fmt"
	"time"

	"github.com/billowdev/clog"
)

// WithAtomicCommitCondition implements IDatabaseTransactor.
func (d *TransactorImpl) WithAtomicCommitCondition(
	ctx context.Context,
	commitCondition func() bool, // Dynamic condition for committing
	timeout time.Duration,
	tFunc func(txCtx context.Context) error,
) error {
	// Create a new context with timeout
	transactionCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	tx := ExtractTx(ctx)
	var isNewTransaction bool
	if tx == nil {
		newTx, err := d.BeginTransaction()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		tx = newTx
		isNewTransaction = true
	}

	var err error

	// Finalize the transaction
	finalizeTransaction := func() {
		if tx == nil {
			return
		}

		if err != nil || tx.Error != nil {
			// Rollback on errors or explicit failure
			if rollbackErr := d.RollbackTransaction(tx); rollbackErr != nil {
				clog.Trace("failed to rollback transaction: %v", rollbackErr)
			}
			return
		}

		select {
		case <-transactionCtx.Done():
			// Rollback if the context is done (timeout or cancel)
			if rollbackErr := d.RollbackTransaction(tx); rollbackErr != nil {
				clog.Trace("failed to rollback transaction: %v", rollbackErr)
			}
		default:
			if isNewTransaction && commitCondition() {
				// Commit if no errors, context is valid, and commit condition is met
				if commitErr := tx.Commit().Error; commitErr != nil {
					traceID, ok := ctx.Value("trace_id").(string)
					if ok && traceID != "" {
						clog.Trace("failed to commit transaction: %v TraceID: %s", commitErr, traceID)
					} else {
						clog.Trace("failed to commit transaction: %v", commitErr)
					}

					// Rollback if commit fails
					if rollbackErr := d.RollbackTransaction(tx); rollbackErr != nil {
						clog.Trace("failed to rollback transaction after commit failure: %v", rollbackErr)
					}
					err = commitErr
				}
			} else if isNewTransaction {
				// Rollback if commit condition is not met
				if rollbackErr := d.RollbackTransaction(tx); rollbackErr != nil {
					clog.Trace("failed to rollback transaction (commit condition false): %v", rollbackErr)
				}
			}
		}
	}
	defer finalizeTransaction()

	// Run the callback function within the transaction context
	err = tFunc(InjectTx(transactionCtx, tx))
	if err != nil {
		tx.Error = err // Mark the transaction as needing a rollback
		return err
	}

	return nil
}

// SAMPLE USAGE
// err := transactor.WithAtomicCommitCondition(ctx, func() bool {
//     // Dynamic commit condition based on some logic
//     return shouldCommit()
// }, 5*time.Second, func(txCtx context.Context) error {
//     // Your transactional logic here
//     return doDatabaseWork(txCtx)
// })
// if err != nil {
//     log.Printf("Transaction failed: %v", err)
// }
