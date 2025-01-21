# Atomic Transactors

A Go package that provides robust transaction management functionality for GORM-based applications, supporting atomic operations with various commit conditions and transaction handling strategies.

## Features

- Atomic transaction management with timeout support
- Context-based transaction propagation
- Basic and dynamic commit conditions
- WaitGroup integration for concurrent operations
- Comprehensive error handling and logging
- Transaction extraction and injection utilities

## Installation

```bash
go get github.com/billowdev/atomic-transactors
```

## Core Components

### 1. TransactorImpl

The main implementation of the transaction manager that provides various transaction handling methods:

```go
type TransactorImpl struct {
    db *gorm.DB
}
```

### 2. Interface Definition

```go
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
```

## Usage Examples

### 1. Basic Atomic Transaction

```go
db, _ := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
transactor := transactors.NewTransactorRepo(db)

ctx := context.Background()
err := transactor.WithAtomic(ctx, 5*time.Second, func(txCtx context.Context) error {
    // Your transaction logic here
    tx := transactors.ExtractTx(txCtx)
    return tx.Create(&User{Name: "John"}).Error
})
```

### 2. Conditional Transaction

```go
err := transactor.WithAtomicBasicCondition(ctx, true, 5*time.Second, func(txCtx context.Context) error {
    // Transaction will commit only if the second parameter is true
    tx := transactors.ExtractTx(txCtx)
    return tx.Create(&User{Name: "Alice"}).Error
})
```

### 3. Dynamic Commit Condition

```go
err := transactor.WithAtomicCommitCondition(ctx, func() bool {
    // Dynamic condition for commit
    return someCondition
}, 5*time.Second, func(txCtx context.Context) error {
    // Transaction logic
    return nil
})
```

## Transaction Utilities

### Context Injection/Extraction

```go
// Inject transaction into context
txCtx := transactors.InjectTx(ctx, tx)

// Extract transaction from context
tx := transactors.ExtractTx(ctx)
```

### WaitGroup Integration

```go
// Create context with WaitGroup
ctx, wg := transactors.WithWaitGroup(context.Background())

// Use WaitAndClose for managed goroutine execution
transactors.WaitAndClose(ctx, func() {
    // Your goroutine logic here
}, func() {
    // Cleanup logic here
})
```

## Error Handling

The package provides comprehensive error handling:
- Transaction timeouts
- Rollback on errors
- Commit failures
- Context cancellation

All errors are properly logged using the `clog` package.

## Best Practices

1. Always use timeouts with transactions:
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
```

2. Handle transaction errors:
```go
if err := transactor.WithAtomic(ctx, timeout, func(txCtx context.Context) error {
    // Your logic
    if err != nil {
        return err // Will trigger rollback
    }
    return nil
}); err != nil {
    // Handle error
}
```

3. Use trace IDs for debugging:
```go
ctx = context.WithValue(ctx, "trace_id", "unique-trace-id")
```

## Dependencies

- `gorm.io/gorm`: v1.25.12
- `github.com/billowdev/clog`: v1.0.0
- Go version: 1.23.5 or higher

## License

This package is available under the MIT license.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.