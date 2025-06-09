package sqlxrw

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/pilotso11/lazywritercache" // For CacheableLF and CacheReaderWriterLF, CacheAction
)

// SqlxRwToLazyCacheAdapter adapts an [ReadWriterImpl] to the
// [lazywritercache.CacheReaderWriterLF] interface.
//
// This allows the generic sqlx-based ReadWriter to be used as a backend
// for the [lazywritercache.LazyWriterCacheLF].
//
// The generic type K is comparable and represents the key type for cache items.
// The generic type T must implement [lazywritercache.CacheableLF[K]], which includes
// methods for Key() to return its cache key, and CopyKeyDataFrom() to merge
// database-generated fields (like auto-incremented IDs or updated timestamps)
// from a database-fresh instance into a cached instance.
type SqlxRwToLazyCacheAdapter[K comparable, T lazywritercache.CacheableLF[K]] struct {
	// sqlxRW is the underlying ReadWriterImpl instance that performs the database operations.
	sqlxRW *ReadWriterImpl[K, T]
}

// NewSqlxRwToLazyCacheAdapter creates a new adapter that wraps the provided
// [ReadWriterImpl].
//
// The `rw` argument is the underlying sqlx-based reader/writer that will perform
// the actual database operations. It must not be nil.
//
// Returns a pointer to the new adapter or an error if `rw` is nil.
func NewSqlxRwToLazyCacheAdapter[K comparable, T lazywritercache.CacheableLF[K]](
	rw *ReadWriterImpl[K, T],
) (*SqlxRwToLazyCacheAdapter[K, T], error) {
	if rw == nil {
		return nil, errors.New("sqlxrw_adapter: provided ReadWriterImpl cannot be nil")
	}
	return &SqlxRwToLazyCacheAdapter[K, T]{sqlxRW: rw}, nil
}

// Find retrieves an item of type T identified by `key` using the underlying [ReadWriterImpl].
// It conforms to the [lazywritercache.CacheReaderWriterLF] interface.
//
// The `txParam` argument, if non-nil, is expected to be of type [Tx] (typically `*sqlx.Tx`),
// which is then passed to the underlying ReadWriter's Find method. If `txParam` is nil,
// the operation is performed without an explicit transaction.
// An error is returned if `txParam` is non-nil and fails the type assertion to [Tx],
// or if the underlying Find operation fails.
func (a *SqlxRwToLazyCacheAdapter[K, T]) Find(ctx context.Context, key K, txParam any) (T, error) {
	var sqlxTx Tx
	if txParam != nil {
		var ok bool
		sqlxTx, ok = txParam.(Tx)
		if !ok {
			var zeroT T
			err := fmt.Errorf("sqlxrw_adapter: transaction type mismatch in Find, expected sqlxrw.Tx, got %T", txParam)
			a.sqlxRW.logger.Error(ctx, err, "Find type assertion failed for key %v", key)
			return zeroT, err
		}
	}
	return a.sqlxRW.Find(ctx, key, sqlxTx)
}

// Save stores the given `item` using the underlying [ReadWriterImpl].
// It conforms to the [lazywritercache.CacheReaderWriterLF] interface.
// The underlying Save method of [ReadWriterImpl] is expected to handle upsert logic
// based on its configuration.
//
// The `txParam` argument, if non-nil, is asserted to type [Tx] and passed to the
// underlying Save method. If `txParam` is nil, the operation occurs outside an explicit transaction.
// An error is returned if `txParam` is non-nil and fails the type assertion, or if the
// underlying Save operation fails.
func (a *SqlxRwToLazyCacheAdapter[K, T]) Save(ctx context.Context, item T, txParam any) error {
	var sqlxTx Tx
	if txParam != nil {
		var ok bool
		sqlxTx, ok = txParam.(Tx)
		if !ok {
			err := fmt.Errorf("sqlxrw_adapter: transaction type mismatch in Save, expected sqlxrw.Tx, got %T", txParam)
			// item.Key() is available because T is constrained by lazywritercache.CacheableLF[K]
			a.sqlxRW.logger.Error(ctx, err, "Save type assertion failed for item key %v", item.Key())
			return err
		}
	}
	return a.sqlxRW.Save(ctx, item, sqlxTx)
}

// BeginTx initializes a new database transaction using the underlying [ReadWriterImpl].
// It conforms to the [lazywritercache.CacheReaderWriterLF] interface by returning the
// transaction object as `any`. The actual concrete type of the returned transaction
// is `*sqlx.Tx`, which implements the [Tx] interface.
func (a *SqlxRwToLazyCacheAdapter[K, T]) BeginTx(ctx context.Context) (any, error) {
	return a.sqlxRW.BeginTx(ctx)
}

// CommitTx commits the provided transaction.
// It conforms to the [lazywritercache.CacheReaderWriterLF] interface.
// The `txParam` argument is asserted to type [Tx] before committing via the
// underlying [ReadWriterImpl].
// An error is returned if `txParam` is nil, if the type assertion fails, or if the commit fails.
func (a *SqlxRwToLazyCacheAdapter[K, T]) CommitTx(ctx context.Context, txParam any) error {
	if txParam == nil {
		err := errors.New("sqlxrw_adapter: cannot commit a nil transaction")
		a.sqlxRW.logger.Error(ctx, err, "CommitTx failed due to nil transaction object")
		return err
	}
	sqlxTx, ok := txParam.(Tx)
	if !ok {
		err := fmt.Errorf("sqlxrw_adapter: transaction type mismatch in CommitTx, expected sqlxrw.Tx, got %T", txParam)
		a.sqlxRW.logger.Error(ctx, err, "CommitTx type assertion failed")
		return err
	}
	return a.sqlxRW.CommitTx(ctx, sqlxTx)
}

// RollbackTx rolls back the provided transaction.
// It conforms to the [lazywritercache.CacheReaderWriterLF] interface.
// The `txParam` argument is asserted to type [Tx] before rolling back via the
// underlying [ReadWriterImpl].
// An error is returned if `txParam` is nil, if the type assertion fails, or if the rollback fails.
func (a *SqlxRwToLazyCacheAdapter[K, T]) RollbackTx(ctx context.Context, txParam any) error {
	if txParam == nil {
		err := errors.New("sqlxrw_adapter: cannot rollback a nil transaction")
		a.sqlxRW.logger.Error(ctx, err, "RollbackTx failed due to nil transaction object")
		return err
	}
	sqlxTx, ok := txParam.(Tx)
	if !ok {
		err := fmt.Errorf("sqlxrw_adapter: transaction type mismatch in RollbackTx, expected sqlxrw.Tx, got %T", txParam)
		a.sqlxRW.logger.Error(ctx, err, "RollbackTx type assertion failed")
		return err
	}
	return a.sqlxRW.RollbackTx(ctx, sqlxTx)
}

// Info logs an informational message using the underlying [ReadWriterImpl]'s logger.
// It conforms to the [lazywritercache.CacheReaderWriterLF] interface.
// The message is prefixed with the string representation of the `action` parameter
// (e.g., "FLUSH_SUCCESS"). Keys of provided `items` are logged for context.
func (a *SqlxRwToLazyCacheAdapter[K, T]) Info(ctx context.Context, msg string, action lazywritercache.CacheAction, items ...T) {
	logArgs := make([]any, 0, len(items))
	for _, item := range items {
		logArgs = append(logArgs, item.Key())
	}
	fullMsg := fmt.Sprintf("[%s] %s", action.String(), msg)
	a.sqlxRW.logger.Info(ctx, fullMsg, logArgs...)
}

// Warn logs a warning message using the underlying [ReadWriterImpl]'s logger.
// It conforms to the [lazywritercache.CacheReaderWriterLF] interface.
// The message is prefixed with "[WARN]" and the string representation of the `action`.
// Keys of provided `items` are logged for context.
// This adapter uses the underlying logger's Info level for warnings from the cache layer,
// as the [lazywritercache.CacheReaderWriterLF.Warn] method does not pass a Go `error` object.
func (a *SqlxRwToLazyCacheAdapter[K, T]) Warn(ctx context.Context, msg string, action lazywritercache.CacheAction, items ...T) {
	logArgs := make([]any, 0, len(items))
	for _, item := range items {
		logArgs = append(logArgs, item.Key())
	}
	fullMsg := fmt.Sprintf("[%s] %s", action.String(), msg)
	a.sqlxRW.logger.Info(ctx, "[WARN] "+fullMsg, logArgs...)
}

// IsRecoverable determines if a database error `err` is transient, suggesting that
// the operation might succeed if retried.
// It conforms to the [lazywritercache.CacheReaderWriterLF] interface.
//
// This implementation performs basic string matching for common recoverable error
// messages related to deadlocks, lock timeouts, and serialization failures for
// databases like PostgreSQL, MySQL, and SQLite. For example, it checks for substrings
// like "deadlock", "lock request time out", "serialization failure", "sqlite_busy".
//
// It specifically treats [sql.ErrTxDone] as not recoverable for the purpose of retrying
// the same operation on an already finalized transaction.
//
// This method is not exhaustive and may need to be extended or customized for specific
// database drivers or more nuanced error handling requirements.
func (a *SqlxRwToLazyCacheAdapter[K, T]) IsRecoverable(_ context.Context, err error) bool {
	if err == nil {
		return false
	}
	// Normalize error message for case-insensitive string matching.
	errStr := strings.ToLower(err.Error())

	// Common patterns for recoverable errors.
	recoverablePatterns := []string{
		"deadlock",                   // General deadlock keyword
		"lock request time out",      // Lock timeout
		"serialization failure",      // PostgreSQL serialization error (e.g., "could not serialize access due to concurrent update")
		"try restarting transaction", // Common advice for recoverable errors
		"sqlite_busy",                // SQLite busy error
		"database is locked",         // Another SQLite busy error
		"40p01",                      // PostgreSQL deadlock_detected error code
		"1213",                       // MySQL ER_LOCK_DEADLOCK error code
	}

	for _, pattern := range recoverablePatterns {
		if strings.Contains(errStr, pattern) {
			// Log at debug level if a recoverable error is identified.
			// Using a background context for adapter's internal logging decision if original context might be done.
			a.sqlxRW.logger.Debug(context.Background(), "Identified error as recoverable by IsRecoverable: %v", err)
			return true
		}
	}

	// sql.ErrTxDone indicates an operation was attempted on a transaction that is already
	// committed or rolled back. This is generally not recoverable by simply retrying the
	// operation on the same transaction; a new transaction would be needed.
	if errors.Is(err, sql.ErrTxDone) {
		a.sqlxRW.logger.Debug(context.Background(), "Identified sql.ErrTxDone as not typically recoverable for the same operation by IsRecoverable: %v", err)
		return false
	}

	// Log at debug level if an error is not identified as recoverable.
	a.sqlxRW.logger.Debug(context.Background(), "Error not identified as recoverable by IsRecoverable: %v", err)
	return false // Default to not recoverable
}

// Fail logs an irrecoverable error that occurred during a cache flush operation,
// as reported by the [lazywritercache.LazyWriterCacheLF].
// It conforms to the [lazywritercache.CacheReaderWriterLF] interface.
// This method uses the underlying [ReadWriterImpl]'s Error logger, including the
// original `err` and the keys of the affected `items` for diagnostic purposes.
func (a *SqlxRwToLazyCacheAdapter[K, T]) Fail(ctx context.Context, err error, items ...T) {
	itemKeys := make([]any, len(items))
	for i, item := range items {
		itemKeys[i] = item.Key()
	}
	a.sqlxRW.logger.Error(ctx, err, "Fail called by lazywritercache for items (keys logged)", itemKeys...)
}
