# txmanager

[![Build Status](https://travis-ci.org/shogo82148/txmanager.svg?branch=master)](https://travis-ci.org/shogo82148/txmanager)

The txnmanager package is a nested transation manager for database/sql.

## SYNOPSIS

Use `TxBegin` to start a transaction, and `TxCommit` or `TxRollback` to finish the transaction.

``` go
import (
	"database/sql"

	"github.com/shogo82148/txmanager"
)

func Example(db *sql.DB) {
	dbm := txmanager.NewDB(db)

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

You can manage `txmanager.DB` with `txmanager.Do`.

``` go
import (
	"database/sql"

	"github.com/shogo82148/txmanager"
)

func Example(db *sql.DB) {
	dbm := txmanager.NewDB(db)
	txmanager.Do(dbm, func(tx txmanager.Tx) error {
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

	"github.com/shogo82148/txmanager"
)

func Foo(dbm *txmanager.DB) error {
	return txmanager.Do(dbm, func(tx txmanager.Tx) error {
		_, err := tx.Exec("INSERT INTO t1 (id) VALUES(1)")
		return err
	})
}

func Example(db *sql.DB) {
	dbm := txmanager.NewDB(db)

	Foo(dbm)

	txmanager.Do(dbm, func(tx txmanager.Tx) error {
		return Foo(tx)
	})
}

```

## END HOOK

`TxCommit` necessarily does not do COMMIT SQL statemant.
So following code sometimes outputs wrong log.

``` go
func Foo(dbm *txmanager.DB) error {
	err := txmanager.Do(dbm, func(tx txmanager.Tx) error {
		_, err := tx.Exec("INSERT INTO t1 (id) VALUES(1)"); err != nil {
		return err
	})
	if err != nil {
		return err
	}

	// TxCommit is success, while the transaction might fail
	log.Println("COMMIT is success!!!")
	return nil
}
```

Use `TxAddEndHook` to avoid it.
It is inspired by [DBIx::TransactionManager::EndHook](https://github.com/soh335/DBIx-TransactionManager-EndHook).

``` go
func Foo(dbm *txmanager.DB) error {
	return txmanager.Do(dbm, func(tx txmanager.Tx) error {
		if _, err := tx.Exec("INSERT INTO t1 (id) VALUES(1)"); err != nil {
			return err
		}
		tx.TxAddEndHook(func() error {
			// It is called if all transactions are executed successfully.
			log.Println("COMMIT is success!!!")
		})
		return nil
	})
}
```

## LICENSE

This software is released under the MIT License, see LICENSE.txt.

## godoc

See [godoc](https://godoc.org/github.com/shogo82148/txmanager) for more imformation.
