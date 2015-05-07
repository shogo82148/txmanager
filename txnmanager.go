package txnmanager

import (
	"database/sql"
	"errors"
)

type Dbm interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row

	TxBegin() (Dbm, error)
	TxCommit() error
	TxRollback() error
	TxFinish() error
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
		return errors.New("txmanager: child transactions are not done")
	}

	t.done = true
	if t.parent == nil {
		err := t.Commit()
		if err != nil {
			return t.Rollback()
		}
	} else {
		t.parent.childCount--
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
