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

	TxnBegin() (Dbm, error)
	TxnCommit() error
	TxnRollback() error
	TxnFinish() error
}

type dbm struct {
	*sql.DB
}

type txn struct {
	*sql.Tx
	parent     *txn
	childCount int
	isDone     bool
}

func NewDbm(db *sql.DB) Dbm {
	return &dbm{db}
}

func (dbm *dbm) TxnBegin() (Dbm, error) {
	t, err := dbm.Begin()
	if err != nil {
		return nil, err
	}
	return &txn{Tx: t}, nil
}

func (dbm *dbm) TxnCommit() error {
	return sql.ErrTxDone
}

func (dbm *dbm) TxnRollback() error {
	return sql.ErrTxDone
}

func (dbm *dbm) TxnFinish() error {
	return nil
}

func (txn *txn) TxnBegin() (Dbm, error) {
	txn.childCount++
	return &txn{
		Tx:     txn.Tx,
		parent: txn,
	}, nil
}

func (txn *txn) TxnCommit() error {
	if txn.isDone {
		return nil
	}

	if txn.childCount != 0 {
		txn.TxnRollback()
		return errors.New("txnmanager: child transactions are not done")
	}

	if txn.parent == nil {
		err := txn.Commit()
		if err != nil {
			return txn.TxnRollback()
		}
		txn.isDone = true
	} else {
		txn.parent.childCount--
	}
	return nil
}

func (txn *txn) TxnRollback() error {
	if txn.isDone {
		return nil
	}
	txn.isDone = true
	return txn.Rollback()
}

func (txn *txn) TxnFinish() error {
	if txn.isDone {
		return nil
	}
	return txn.TxnRollback()
}

func Do(dbm Dbm, f func(txn Dbm) error) error {
	txn, err := dbm.TxnBegin()
	if err != nil {
		return err
	}
	defer txn.TxnFinish()

	err = f(txn)
	if err != nil {
		return err
	}
	return txn.TxnCommit()
}
