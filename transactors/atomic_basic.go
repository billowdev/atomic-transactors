package transactors

import (
	"context"
	"fmt"
	"time"

	"github.com/billowdev/clog"
)

// WithAtomicBasicCondition implements IDatabaseTransactor.
func (d *TransactorImpl) WithAtomicBasicCondition(ctx context.Context, isCommit bool, timeout time.Duration, tFunc func(txCtx context.Context) error) error {

	// Create a new context with timeout
	transactionCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	tx := ExtractTx(ctx)
	if tx == nil {
		newTx, err := d.BeginTransaction()
		if err != nil {
			return fmt.Errorf("begin transaction: %w", tx.Error)
		}
		tx = newTx
	}

	var err error
	// Ensure that the transaction is finalized properly
	defer func() {
		// First check if there were any errors during execution
		if err != nil || tx.Error != nil {
			if rollbackErr := d.RollbackTransaction(tx); rollbackErr != nil {
				clog.Trace("failed to rollback transaction: %v", rollbackErr)
			}
			return
		}

		select {
		case <-transactionCtx.Done():
			// Rollback if context is done (timeout or cancel)
			if rollbackErr := d.RollbackTransaction(tx); rollbackErr != nil {
				clog.Trace("failed to rollback transaction: %v", rollbackErr)
			}
		default:
			// Commit if no error and context is still valid
			if isCommit {
				// Attempt to commit only if no errors and context is valid
				if commitErr := tx.Commit().Error; commitErr != nil {
					traceID, ok := ctx.Value("trace_id").(string)
					if ok && traceID != "" {
						clog.Trace("failed to commit transaction: %v TraceID: %s", commitErr, traceID)
					} else {
						clog.Trace("failed to commit transaction: %v", commitErr)
					}
					// If commit fails, rollback
					if rollbackErr := d.RollbackTransaction(tx); rollbackErr != nil {
						clog.Trace("failed to rollback transaction after commit failure: %v", rollbackErr)
					}
					err = commitErr
				}
			}
		}
	}()

	// Run the callback function with the transaction context
	err = tFunc(InjectTx(transactionCtx, tx))
	if err != nil {
		tx.Error = err // Mark the transaction as needing a rollback
		return err
	}

	return nil
}
