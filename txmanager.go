// The txnmanager is a nested transation manager for database/sql.

package txmanager

import (
	"database/sql"
	"errors"
)

var ErrChildrenNotDone = errors.New("txmanager: children transactions are not done")

// an Executor executes SQL query.
type Executor interface {
	// Exec executes a query without returning any rows.
	// See sql.DB and sql.Tx for more information.
	Exec(query string, args ...interface{}) (sql.Result, error)

	// Prepare creates a prepared statement for later queries or executions.
	// See sql.DB and sql.Tx for more information.
	Prepare(query string) (*sql.Stmt, error)

	// Query executes a query that returns rows, typically a SELECT.
	// See sql.DB and sql.Tx for more information.
	Query(query string, args ...interface{}) (*sql.Rows, error)

	// QueryRow executes a query that is expected to return at most one row.
	// See sql.DB and sql.Tx for more information.
	QueryRow(query string, args ...interface{}) *sql.Row
}

// a Beginner starts a transaction.
type Beginner interface {
	// TxBegin starts a transaction.
	// If the DB is a transaction, TxBegin does't do BEGIN at here.
	// It just pushed transaction stack and do nothing.
	TxBegin() (Tx, error)
}

// a Committer finishes a transaction
type Committer interface {
	// TxCommit commits the transaction.
	// If the DB is in a nested transaction, TxCommit doesn't do COMMIT at here.
	// It just poped transaction stack and do nothing.
	TxCommit() error

	// TxRollback aborts the transaction.
	// TxRollback always does ROLLBACK at here.
	TxRollback() error

	// TxFinish aborts the transaction if it is not commited.
	TxFinish() error
}

// an EndHookAdder adds end hooks
type EndHookAdder interface {
	// TxAddEndHook add a hook function to txmanager.
	// Hooks are executed only all transactions are executed successfully.
	// If some transactions are failed, they aren't executed.
	TxAddEndHook(hook func() error) error
}

type DB interface {
	Executor
	Beginner
}

type Tx interface {
	Executor
	Beginner
	Committer
	EndHookAdder
}

type db struct {
	*sql.DB
}

type tx struct {
	*sql.Tx
	parent     *tx
	root       *tx
	childCount int
	done       bool
	hooks      []func() error
}

func NewDB(d *sql.DB) DB {
	return &db{d}
}

func (d *db) TxBegin() (Tx, error) {
	t, err := d.Begin()
	if err != nil {
		return nil, err
	}

	child := &tx{Tx: t}
	child.root = child
	return child, nil
}

func (t *tx) TxBegin() (Tx, error) {
	t.childCount++
	child := &tx{
		Tx:     t.Tx,
		parent: t,
		root:   t.root,
	}
	return child, nil
}

func (t *tx) TxCommit() error {
	if t.done || t.root.done {
		return sql.ErrTxDone
	}

	if t.childCount != 0 {
		t.TxRollback()
		return ErrChildrenNotDone
	}

	t.done = true
	if t.parent != nil {
		// t is a nested transaction.
		// just pop transaction stack
		t.parent.childCount--

		// ... and do nothing
		return nil
	}

	// Do COMMIT
	err := t.Commit()
	if err != nil {
		return t.Rollback()
	}

	// call end hooks
	if t.hooks != nil {
		for _, h := range t.hooks {
			if err := h(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *tx) TxRollback() error {
	if t.done || t.root.done {
		return sql.ErrTxDone
	}
	t.done = true
	t.root.done = true
	return t.Rollback()
}

func (t *tx) TxFinish() error {
	if t.done || t.root.done {
		return nil
	}
	return t.TxRollback()
}

func (t *tx) TxAddEndHook(hook func() error) error {
	if t.done || t.root.done {
		return sql.ErrTxDone
	}
	t.root.hooks = append(t.root.hooks, hook)
	return nil
}

// Do executes the function in a transaction.
func Do(d DB, f func(t Tx) error) error {
	t, err := d.TxBegin()
	if err != nil {
		return err
	}
	defer t.TxFinish()
	err = f(t)
	if err != nil {
		return err
	}
	return t.TxCommit()
}
