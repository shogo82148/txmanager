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

## END HOOK

`TxCommit` necessarily does not do COMMIT SQL statemant.
So following code sometimes outputs wrong log.

``` go
func Foo(dbm *txmanager.Dbm) error {
	err := txmanager.Do(dbm, func(tx Dbm) error {
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
func Foo(dbm *txmanager.Dbm) error {
	return txmanager.Do(dbm, func(tx Dbm) error {
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

## godoc

See [godoc](https://godoc.org/github.com/shogo82148/txnmanager) for more imformation.
