// The txnmanager is a nested transation manager for database/sql.

package txmanager

import (
	"database/sql"
	"errors"
)

var ErrChildrenNotDone = errors.New("txmanager: children transactions are not done")

type Dbm interface {
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

	// TxBegin starts a transaction.
	// If the Dbm is a transaction, TxBegin does't do BEGIN at here.
	// It just pushed transaction stack and do nothing.
	TxBegin() (Dbm, error)

	// TxCommit commits the transaction.
	// If the Dbm is in a nested transaction, TxCommit doesn't do COMMIT at here.
	// It just poped transaction stack and do nothing.
	TxCommit() error

	// TxRollback aborts the transaction.
	// TxRollback always does ROLLBACK at here.
	TxRollback() error

	// TxFinish aborts the transaction if it is not commited.
	TxFinish() error

	// TxAddEndHook add a hook function to txmanager.
	// Hooks are executed only all transactions are executed successfully.
	// If some transactions are failed, they aren't executed.
	TxAddEndHook(hook func() error) error
}

type dbm struct {
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

func NewDbm(db *sql.DB) Dbm {
	return &dbm{db}
}

func (d *dbm) TxBegin() (Dbm, error) {
	t, err := d.Begin()
	if err != nil {
		return nil, err
	}

	child := &tx{Tx: t}
	child.root = child
	return child, nil
}

func (d *dbm) TxCommit() error {
	return sql.ErrTxDone
}

func (d *dbm) TxRollback() error {
	return sql.ErrTxDone
}

func (d *dbm) TxFinish() error {
	return nil
}

func (d *dbm) TxAddEndHook(hook func() error) error {
	return sql.ErrTxDone
}

func (t *tx) TxBegin() (Dbm, error) {
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
func Do(d Dbm, f func(t Dbm) error) error {
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
