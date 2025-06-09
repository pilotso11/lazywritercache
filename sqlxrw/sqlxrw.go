package sqlxrw

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

// Logger defines a simple logging interface that can be implemented by users
// to integrate with their existing logging infrastructure. It provides methods
// for logging informational messages, errors, and debug details.
type Logger interface {
	// Info logs an informational message.
	// `args` are optional and can be used for formatted strings.
	Info(ctx context.Context, msg string, args ...any)
	// Error logs an error message along with the error itself.
	// `args` are optional and can be used for formatted strings.
	Error(ctx context.Context, err error, msg string, args ...any)
	// Debug logs a debug message, typically for more verbose output.
	// `args` are optional and can be used for formatted strings.
	Debug(ctx context.Context, msg string, args ...any)
}

// defaultLogger provides a basic implementation of the Logger interface
// using the standard Go `log` package.
type defaultLogger struct{}

// Info logs an informational message using the standard log package.
func (dl *defaultLogger) Info(ctx context.Context, msg string, args ...any) {
	log.Printf("[INFO] "+msg, args...)
}

// Error logs an error message and the associated error using the standard log package.
func (dl *defaultLogger) Error(ctx context.Context, err error, msg string, args ...any) {
	fullMsg := fmt.Sprintf("[ERROR] "+msg, args...)
	if err != nil {
		fullMsg += fmt.Sprintf(" | error: %v", err)
	}
	log.Println(fullMsg)
}

// Debug logs a debug message using the standard log package.
func (dl *defaultLogger) Debug(ctx context.Context, msg string, args ...any) {
	log.Printf("[DEBUG] "+msg, args...)
}

var _ Logger = (*defaultLogger)(nil)

// Tx defines an interface for database operations that can be executed either
// directly on a database connection (*sqlx.DB) or within a transaction (*sqlx.Tx).
// It abstracts the common methods from `sqlx.QueryerContext`, `sqlx.ExecerContext`,
// and `sqlx.PreparerContext`, and adds methods specific to transactions (`Commit`, `Rollback`)
// as well as `GetContext`, `SelectContext`, and `NamedExecContext` which are available
// on both `*sqlx.DB` and `*sqlx.Tx`.
// This allows `ReadWriterImpl` methods to operate consistently whether an explicit
// transaction is provided or not (for single operations not within a caller-managed transaction).
type Tx interface {
	sqlx.QueryerContext  // Provides QueryContext, QueryRowContext, QueryxContext, QueryRowxContext
	sqlx.ExecerContext   // Provides ExecContext
	sqlx.PreparerContext // Provides PreparexContext

	// GetContext retrieves a single row and scans it into the dest.
	// `dest` must be a pointer.
	GetContext(ctx context.Context, dest interface{}, query string, args ...any) error
	// SelectContext retrieves multiple rows and scans them into the dest.
	// `dest` must be a pointer to a slice.
	SelectContext(ctx context.Context, dest interface{}, query string, args ...any) error
	// NamedExecContext executes a query that uses named bind variables.
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)

	// Commit commits the transaction.
	// This method is only meaningful if the underlying implementation is an actual transaction (*sqlx.Tx).
	// Calling Commit on a Tx instance that represents a *sqlx.DB may result in an error or no-op.
	Commit() error
	// Rollback aborts the transaction.
	// Similar to Commit, this is only meaningful for actual transactions (*sqlx.Tx).
	Rollback() error
}

// ReadWriter defines the core interface for reading and writing data of type T
// with a key of type K. It supports operations within or outside explicit transactions.
type ReadWriter[K comparable, T any] interface {
	// Find retrieves an item by its key.
	// If `tx` is provided, the operation is performed within that transaction.
	// If `tx` is nil, the operation is performed directly on the underlying database connection.
	// Returns the item of type T or an error if not found or if another error occurs.
	Find(ctx context.Context, key K, tx Tx) (T, error)

	// Save inserts or updates an item.
	// If `tx` is provided, the operation is performed within that transaction.
	// If `tx` is nil, the operation is performed directly on the underlying database connection.
	// The behavior (upsert vs. update-then-insert) depends on the Config.
	Save(ctx context.Context, item T, tx Tx) error

	// BeginTx starts a new database transaction and returns it as a Tx interface.
	// The concrete type returned is *sqlx.Tx, which implements the Tx interface.
	BeginTx(ctx context.Context) (Tx, error)

	// CommitTx commits the provided transaction.
	// The `tx` argument should be a transaction previously obtained from BeginTx.
	CommitTx(ctx context.Context, tx Tx) error

	// RollbackTx rolls back the provided transaction.
	// The `tx` argument should be a transaction previously obtained from BeginTx.
	RollbackTx(ctx context.Context, tx Tx) error
}

// Config holds the SQL queries and data extraction functions required by a ReadWriter.
// It allows users to define how data is fetched, inserted, updated, or upserted
// for a generic type T with a key of type K.
// SQL queries use the bind variable format supported by the underlying driver (e.g., "?", "$1").
// `sqlx` will typically rebind these to the correct format if necessary when using `sqlx.DB.Rebind`.
type Config[K comparable, T any] struct {
	// FindQuery is the SQL query to find an item by its key.
	// It should expect a single bind variable (e.g., "?") for the key.
	// Example: "SELECT * FROM users WHERE id = ?"
	FindQuery string

	// InsertQuery is the SQL query to insert a new item.
	// The order of bind variables should match the order of values returned by InsertArgsExtractor.
	// Example: "INSERT INTO users (name, email) VALUES (?, ?)"
	InsertQuery string

	// UpdateQuery is the SQL query to update an existing item.
	// The order of bind variables should match the order of values returned by UpdateArgsExtractor.
	// Typically, the key for the WHERE clause is the last bind variable.
	// Example: "UPDATE users SET name = ?, email = ? WHERE id = ?"
	UpdateQuery string

	// UpsertQuery is an optional SQL query to perform an "upsert" operation (insert or update).
	// This query is database-specific (e.g., using "ON CONFLICT" for PostgreSQL/SQLite or "ON DUPLICATE KEY UPDATE" for MySQL).
	// If provided, this query is used by Save. Otherwise, Save uses an update-then-insert strategy.
	// The order of bind variables should match UpsertArgsExtractor (or InsertArgsExtractor if UpsertArgsExtractor is nil).
	// Example (SQLite): "INSERT INTO users (id, name, email) VALUES (?, ?, ?) ON CONFLICT(id) DO UPDATE SET name=excluded.name, email=excluded.email"
	UpsertQuery string

	// KeyExtractor is a function that extracts the key (of type K) from an item (of type T).
	// This is crucial for update operations and the update-then-insert strategy in Save.
	KeyExtractor func(item T) K

	// InsertArgsExtractor is a function that extracts a slice of arguments for the InsertQuery from an item.
	// The order of arguments in the slice must match the order of bind variables in InsertQuery.
	InsertArgsExtractor func(item T) ([]any, error)

	// UpdateArgsExtractor is a function that extracts a slice of arguments for the UpdateQuery from an item.
	// The order of arguments in the slice must match the order of bind variables in UpdateQuery.
	// Typically, the key for the WHERE clause is the last argument in the slice.
	UpdateArgsExtractor func(item T) ([]any, error)

	// UpsertArgsExtractor is an optional function to extract arguments for the UpsertQuery.
	// If UpsertQuery is provided and UpsertArgsExtractor is nil, InsertArgsExtractor is used as a fallback.
	// This is useful if the UpsertQuery requires a different set or order of arguments than a simple insert.
	UpsertArgsExtractor func(item T) ([]any, error)
}

// dbExecutor is an internal interface that consolidates methods common to *sqlx.DB and *sqlx.Tx.
// This is used to type the 'executor' variable within Find and Save methods, allowing them
// to operate on either a direct DB connection or a transaction object.
type dbExecutor interface {
	GetContext(ctx context.Context, dest interface{}, query string, args ...any) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...any) error
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error)
	QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row
}

// ReadWriterImpl provides a concrete implementation of the ReadWriter interface
// using `sqlx` for database interactions.
// It is generic for item type T and key type K.
type ReadWriterImpl[K comparable, T any] struct {
	db     *sqlx.DB     // The underlying database connection pool.
	config Config[K, T] // Configuration for SQL queries and argument extractors.
	logger Logger       // Logger for internal operations.
}

// NewReadWriterImpl creates a new instance of ReadWriterImpl.
//
// Parameters:
//   - db: An initialized *sqlx.DB database connection. Cannot be nil.
//   - config: A Config struct defining SQL queries and data extraction functions.
//     It must contain a valid FindQuery. For Save operations, it needs either
//     a complete set for upsert (UpsertQuery and its args extractor or InsertArgsExtractor)
//     or a complete set for update-then-insert (InsertQuery, UpdateQuery, and their
//     respective args extractors, plus KeyExtractor).
//   - logger: An optional variadic Logger interface. If provided and not nil, the first logger
//     is used. Otherwise, a default logger (using the standard `log` package) is initialized.
//
// Returns:
//
//	A pointer to the new ReadWriterImpl instance or an error if validation of `db` or `config` fails.
func NewReadWriterImpl[K comparable, T any](db *sqlx.DB, config Config[K, T], logger ...Logger) (*ReadWriterImpl[K, T], error) {
	if db == nil {
		return nil, errors.New("sqlxrw: db cannot be nil")
	}
	if config.FindQuery == "" {
		return nil, errors.New("sqlxrw: Config.FindQuery cannot be empty")
	}
	// Validation for Save-related fields
	hasInsert := config.InsertQuery != "" && config.InsertArgsExtractor != nil
	hasUpdate := config.UpdateQuery != "" && config.UpdateArgsExtractor != nil && config.KeyExtractor != nil
	hasUpsert := config.UpsertQuery != "" && (config.UpsertArgsExtractor != nil || config.InsertArgsExtractor != nil)

	if !((hasInsert && hasUpdate) || hasUpsert) {
		return nil, errors.New("sqlxrw: Config must provide either (InsertQuery & InsertArgsExtractor AND UpdateQuery & UpdateArgsExtractor & KeyExtractor) OR (UpsertQuery & (UpsertArgsExtractor or InsertArgsExtractor))")
	}
	if (hasUpdate && !hasUpsert) && config.KeyExtractor == nil {
		return nil, errors.New("sqlxrw: Config.KeyExtractor cannot be nil if using UpdateQuery without a native UpsertQuery")
	}

	var actualLogger Logger = &defaultLogger{}
	if len(logger) > 0 && logger[0] != nil {
		actualLogger = logger[0]
	}

	return &ReadWriterImpl[K, T]{
		db:     db,
		config: config,
		logger: actualLogger,
	}, nil
}

// BeginTx starts a new database transaction using the underlying *sqlx.DB.
// It returns a Tx interface, which is implemented by *sqlx.Tx.
// Errors from `rwi.db.BeginTxx` are logged and returned.
func (rwi *ReadWriterImpl[K, T]) BeginTx(ctx context.Context) (Tx, error) {
	rwi.logger.Debug(ctx, "BeginTx started")
	tx, err := rwi.db.BeginTxx(ctx, nil)
	if err != nil {
		rwi.logger.Error(ctx, err, "BeginTx failed")
		return nil, fmt.Errorf("sqlxrw: BeginTx failed: %w", err)
	}
	rwi.logger.Debug(ctx, "BeginTx successful")
	return tx, nil // *sqlx.Tx implements Tx
}

// CommitTx commits the provided transaction `tx`.
// It logs the attempt and success or failure.
// It handles cases where `tx` is nil or if the commit operation itself fails,
// ignoring `sql.ErrTxDone` as it indicates the transaction is already completed.
func (rwi *ReadWriterImpl[K, T]) CommitTx(ctx context.Context, tx Tx) error {
	rwi.logger.Debug(ctx, "CommitTx started")
	if tx == nil {
		err := errors.New("sqlxrw: cannot commit a nil transaction")
		rwi.logger.Error(ctx, err, "CommitTx failed due to nil transaction")
		return err
	}
	// sqlx.Tx.Commit() doesn't take context
	err := tx.Commit()
	if err != nil && !errors.Is(err, sql.ErrTxDone) {
		rwi.logger.Error(ctx, err, "CommitTx failed")
		return fmt.Errorf("sqlxrw: CommitTx failed: %w", err)
	}
	rwi.logger.Debug(ctx, "CommitTx successful")
	return nil
}

// RollbackTx rolls back the provided transaction `tx`.
// It logs the attempt and success or failure.
// It handles cases where `tx` is nil or if the rollback operation itself fails,
// ignoring `sql.ErrTxDone` as it indicates the transaction is already completed.
func (rwi *ReadWriterImpl[K, T]) RollbackTx(ctx context.Context, tx Tx) error {
	rwi.logger.Debug(ctx, "RollbackTx started")
	if tx == nil {
		err := errors.New("sqlxrw: cannot rollback a nil transaction")
		rwi.logger.Error(ctx, err, "RollbackTx failed due to nil transaction")
		return err
	}
	// sqlx.Tx.Rollback() doesn't take context
	err := tx.Rollback()
	if err != nil && !errors.Is(err, sql.ErrTxDone) {
		rwi.logger.Error(ctx, err, "RollbackTx failed")
		return fmt.Errorf("sqlxrw: RollbackTx failed: %w", err)
	}
	rwi.logger.Debug(ctx, "RollbackTx successful")
	return nil
}

// Find retrieves an item of type T by its key of type K.
// If `tx` is provided (non-nil), the find operation is performed within that transaction.
// Otherwise, it's performed on the main database connection.
// It uses the `FindQuery` from the `Config`.
// Returns the found item or an error. If no item is found, an error wrapping `sql.ErrNoRows` is returned.
// Logs the operation details, including success or failure.
func (rwi *ReadWriterImpl[K, T]) Find(ctx context.Context, key K, tx Tx) (T, error) {
	var item T // Zero value for T
	rwi.logger.Debug(ctx, "Find started for key: %v", key)

	if rwi.config.FindQuery == "" {
		// This check is also in NewReadWriterImpl, but as a runtime safeguard.
		err := errors.New("sqlxrw: FindQuery is not configured")
		rwi.logger.Error(ctx, err, "Find configuration error for key: %v", key)
		return item, err
	}

	var executor dbExecutor
	if tx != nil {
		executor = tx // Use the provided transaction
	} else {
		executor = rwi.db // Use the main DB connection
	}

	err := executor.GetContext(ctx, &item, rwi.config.FindQuery, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			rwi.logger.Info(ctx, "Item not found for key: %v", key)
			// Standardize error for not found, wrapping sql.ErrNoRows
			return item, fmt.Errorf("sqlxrw: item with key %v not found: %w", key, err)
		}
		rwi.logger.Error(ctx, err, "Find failed for key: %v", key)
		return item, fmt.Errorf("sqlxrw: Find failed for key %v: %w", key, err)
	}
	rwi.logger.Debug(ctx, "Find successful for key: %v", key)
	return item, nil
}

// Save inserts or updates an item of type T in the database.
// If `tx` is provided (non-nil), the save operation is performed within that transaction.
// Otherwise, it's performed on the main database connection.
//
// Behavior:
//  1. If `UpsertQuery` is configured, it attempts an upsert operation using `UpsertArgsExtractor`
//     (or `InsertArgsExtractor` as a fallback if `UpsertArgsExtractor` is nil).
//  2. If `UpsertQuery` is not configured, it attempts an update-then-insert strategy:
//     a. It first tries to update the item using `UpdateQuery` and `UpdateArgsExtractor`.
//     b. If the update affects 0 rows (meaning the item likely doesn't exist),
//     it then tries to insert the item using `InsertQuery` and `InsertArgsExtractor`.
//
// Logs the operation details, chosen strategy, and success or failure.
// Returns an error if any step fails or if configuration is inadequate for the chosen strategy.
func (rwi *ReadWriterImpl[K, T]) Save(ctx context.Context, item T, tx Tx) error {
	var itemKeyDescription string
	if rwi.config.KeyExtractor != nil {
		// Deferred function to catch panics from KeyExtractor, e.g., if item is not initialized.
		defer func() {
			if r := recover(); r != nil {
				rwi.logger.Error(ctx, fmt.Errorf("panic in KeyExtractor: %v", r), "Panic during key extraction for logging in Save")
				// Depending on desired robustness, could re-panic or return a specific error.
				// For now, it just logs.
			}
		}()
		itemKeyDescription = fmt.Sprintf("item with key: %v", rwi.config.KeyExtractor(item))
	} else {
		itemKeyDescription = "item (key not extracted for log)"
	}
	rwi.logger.Debug(ctx, "Save started for %s", itemKeyDescription)

	var executor dbExecutor
	if tx != nil {
		executor = tx // Use the provided transaction
	} else {
		executor = rwi.db // Use the main DB connection
	}

	// Strategy 1: Use dedicated UpsertQuery if available
	if rwi.config.UpsertQuery != "" {
		rwi.logger.Debug(ctx, "Attempting upsert strategy for %s", itemKeyDescription)
		var args []any
		var err error

		if rwi.config.UpsertArgsExtractor != nil {
			args, err = rwi.config.UpsertArgsExtractor(item)
		} else if rwi.config.InsertArgsExtractor != nil { // Fallback to InsertArgsExtractor
			args, err = rwi.config.InsertArgsExtractor(item)
		} else {
			// This state should ideally be caught by NewReadWriterImpl validation.
			err := errors.New("sqlxrw: UpsertQuery is configured but no UpsertArgsExtractor or InsertArgsExtractor found")
			rwi.logger.Error(ctx, err, "Upsert configuration error for %s", itemKeyDescription)
			return err
		}
		if err != nil {
			rwi.logger.Error(ctx, err, "Failed to extract upsert args for %s", itemKeyDescription)
			return fmt.Errorf("sqlxrw: failed to extract upsert args: %w", err)
		}

		_, err = executor.ExecContext(ctx, rwi.config.UpsertQuery, args...)
		if err != nil {
			rwi.logger.Error(ctx, err, "Upsert failed for %s", itemKeyDescription)
			return fmt.Errorf("sqlxrw: upsert failed: %w", err)
		}
		rwi.logger.Info(ctx, "Upsert successful for %s", itemKeyDescription)
		return nil
	}

	// Strategy 2: Update then Insert
	rwi.logger.Debug(ctx, "Attempting update-then-insert strategy for %s", itemKeyDescription)
	// This configuration validity is also checked in NewReadWriterImpl.
	if rwi.config.UpdateQuery == "" || rwi.config.InsertQuery == "" || rwi.config.UpdateArgsExtractor == nil || rwi.config.InsertArgsExtractor == nil || rwi.config.KeyExtractor == nil {
		err := errors.New("sqlxrw: Save configuration incomplete for update-then-insert strategy")
		rwi.logger.Error(ctx, err, "Save configuration error for %s", itemKeyDescription)
		return err
	}

	updateArgs, err := rwi.config.UpdateArgsExtractor(item)
	if err != nil {
		rwi.logger.Error(ctx, err, "Failed to extract update args for %s", itemKeyDescription)
		return fmt.Errorf("sqlxrw: failed to extract update args: %w", err)
	}

	res, err := executor.ExecContext(ctx, rwi.config.UpdateQuery, updateArgs...)
	if err != nil {
		rwi.logger.Error(ctx, err, "Update attempt failed for %s", itemKeyDescription)
		return fmt.Errorf("sqlxrw: update attempt failed: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		rwi.logger.Error(ctx, err, "Failed to get rows affected after update for %s", itemKeyDescription)
		return fmt.Errorf("sqlxrw: failed to get rows affected after update: %w", err)
	}

	if rowsAffected > 0 {
		rwi.logger.Info(ctx, "Update successful for %s", itemKeyDescription)
		return nil // Update successful
	}

	// If update affected 0 rows, item likely doesn't exist, so try to insert.
	rwi.logger.Debug(ctx, "Update affected 0 rows, attempting insert for %s", itemKeyDescription)
	insertArgs, err := rwi.config.InsertArgsExtractor(item)
	if err != nil {
		rwi.logger.Error(ctx, err, "Failed to extract insert args for %s", itemKeyDescription)
		return fmt.Errorf("sqlxrw: failed to extract insert args: %w", err)
	}

	_, err = executor.ExecContext(ctx, rwi.config.InsertQuery, insertArgs...)
	if err != nil {
		rwi.logger.Error(ctx, err, "Insert attempt failed for %s", itemKeyDescription)
		return fmt.Errorf("sqlxrw: insert attempt failed after update affected 0 rows: %w", err)
	}
	rwi.logger.Info(ctx, "Insert successful for %s", itemKeyDescription)
	return nil
}
