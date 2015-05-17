// The savepoint is a nested transation manager which suppurts partial rollback.
package savepoint

import (
	"database/sql"
	"fmt"

	"github.com/shogo82148/txmanager"
)

type db struct {
	*sql.DB
}

type tx struct {
	*sql.Tx
	parent     *tx
	root       *tx
	childCount int
	saveCount  int
	done       bool
	name       string
	hooks      []func() error
}

// NewDB creates new transaction manager which suppurts partial rollback.
// TxBegin does SAVEPOINT, and TxCommit does RELEASE SAVEPOINT in a transaction.
// TxRollback does ROLLBACK TO SAVEPOINT and finish transactions excludes parents.
func NewDB(d *sql.DB) txmanager.DB {
	return &db{d}
}

func (d *db) TxBegin() (txmanager.Tx, error) {
	t, err := d.Begin()
	if err != nil {
		return nil, err
	}

	child := &tx{Tx: t}
	child.root = child
	return child, nil
}

func (t *tx) TxBegin() (txmanager.Tx, error) {
	t.root.saveCount++
	name := fmt.Sprintf("savepoint_%d", t.root.saveCount)
	if _, err := t.Exec("SAVEPOINT " + name); err != nil {
		return nil, err
	}

	t.childCount++
	child := &tx{
		Tx:     t.Tx,
		parent: t,
		root:   t.root,
		name:   name,
	}
	return child, nil
}

func (t *tx) TxCommit() error {
	if t.done || t.root.done {
		return sql.ErrTxDone
	}

	if t.childCount != 0 {
		t.TxRollback()
		return txmanager.ErrChildrenNotDone
	}

	t.done = true
	if t.parent != nil {
		// t is a nested transaction.
		t.parent.childCount--

		_, err := t.Exec("RELEASE SAVEPOINT " + t.name)
		if err != nil {
			return err
		}
		t.parent.hooks = append(t.parent.hooks, t.hooks...)
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

	if t.parent != nil {
		// t is a nested transaction.
		t.parent.childCount--

		_, err := t.Exec("ROLLBACK TO SAVEPOINT " + t.name)
		return err
	}
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
	t.hooks = append(t.hooks, hook)
	return nil
}
