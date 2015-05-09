# txmanager

[![Build Status](https://travis-ci.org/shogo82148/txmanager.svg?branch=master)](https://travis-ci.org/shogo82148/txmanager)

The txnmanager package is a nested transation manager for database/sql.

## SYNOPSIS

Use `TxBegin` to start a transaction, and `TxCommit` or `TxRollback` to finish the transaction.

``` go
import (
	"database/sql"

	"githun.com/shogo82148/txmanager"
)

func Example(db *sql.DB) {
	dbm := txmanager.NewDbm(db)

	// start a transaction
	tx, _ := dbm.TxBegin()
	defer tx.TxFinish()

	// Exec INSERT in a transaction
	_, err := tx.Exec("INSERT INTO t1 (id) VALUES(1)")
	if err != nil {
		tx.TxRollback()
	}
	tx.TxCommit()
}
```

You can manage `txmanager.Dbm` with `txmanager.Do`.

``` go
import (
	"database/sql"

	"githun.com/shogo82148/txmanager"
)

func Example(db *sql.DB) {
	dbm := txmanager.NewDbm(db)
	txmanager.Do(dbm, func(tx txmanager.Dbm) error {
		// Exec INSERT in a transaction
		_, err := tx.Exec("INSERT INTO t1 (id) VALUES(1)")
		return err
	})
}
```

## NESTED TRANSACTION

``` go
import (
	"database/sql"

	"githun.com/shogo82148/txmanager"
)

func Foo(dbm *txmanager.Dbm) error {
	return txmanager.Do(dbm, func(tx Dbm) error {
		_, err := tx.Exec("INSERT INTO t1 (id) VALUES(1)")
		return err
	})
}

func Example(db *sql.DB) {
	dbm := txmanager.NewDbm(db)

	Foo(dbm)

	txmanager.Do(dbm, func(tx Dbm) error {
		return Foo(tx)
	})
}

```

## godoc

See [godoc](https://godoc.org/github.com/shogo82148/txnmanager) for more imformation.
