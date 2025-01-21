package transactors

import (
	"context"
	"fmt"
	"time"

	"github.com/billowdev/clog"
)

func (d *TransactorImpl) WithAtomic(ctx context.Context, timeout time.Duration, tFunc func(txCtx context.Context) error) error {
	// Create a new context with timeout
	transactionCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Extract or create transaction
	tx := ExtractTx(ctx)
	if tx == nil {
		newTx, err := d.BeginTransaction()
		if err != nil {
			return fmt.Errorf("begin transaction: %w", err) // Fixed: was using tx.Error instead of err
		}
		tx = newTx
	}

	var err error
	// Ensure transaction is finalized properly
	defer func() {
		// First check if there were any errors during execution
		if err != nil || tx.Error != nil {
			if rollbackErr := d.RollbackTransaction(tx); rollbackErr != nil {
				clog.Trace("failed to rollback transaction: %v", rollbackErr)
			}
			return
		}

		// Then check context status
		select {
		case <-transactionCtx.Done():
			// Rollback if context is done (timeout or cancel)
			if rollbackErr := d.RollbackTransaction(tx); rollbackErr != nil {
				clog.Trace("failed to rollback transaction: %v", rollbackErr)
			}
		default:
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
	}()

	// Run the callback function with the transaction context
	err = tFunc(InjectTx(transactionCtx, tx))
	if err != nil {
		tx.Error = err
		return err
	}

	return nil
}
