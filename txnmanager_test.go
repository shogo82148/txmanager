package txnmanager

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestCommit(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening database failed: %v", err)
	}
	defer db.Close()

	dbm := NewDbm(db)
	_, err = dbm.Exec(
		"CREATE TABLE t1 (id INTEGER PRIMARY KEY)",
	)
	if err != nil {
		t.Fatalf("creating test table failed: %v", err)
	}

	func() {
		txn, err := dbm.TxnBegin()
		if err != nil {
			t.Fatalf("beginning transaction failed: %v", err)
		}
		defer txn.TxnFinish()

		_, err = txn.Exec("INSERT INTO t1 (id) VALUES(1)")
		if err != nil {
			t.Fatalf("inserting failed: %v", err)
		}

		if err := txn.TxnCommit(); err != nil {
			t.Fatalf("commiting failed: %v", err)
		}
	}()

	row := dbm.QueryRow("SELECT id FROM t1 WHERE id = ?", 1)
	var id int
	if err = row.Scan(&id); err != nil {
		t.Fatalf("selecting row failed: %v", err)
	}
	if id != 1 {
		t.Errorf("got %d\nwant 1", id)
	}
}

func TestRollback(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening database failed: %v", err)
	}
	defer db.Close()

	dbm := NewDbm(db)
	_, err = dbm.Exec(
		"CREATE TABLE t1 (id INTEGER PRIMARY KEY)",
	)
	if err != nil {
		t.Fatalf("creating test table failed: %v", err)
	}

	func() {
		txn, err := dbm.TxnBegin()
		if err != nil {
			t.Fatalf("beginning transaction failed: %v", err)
		}
		defer txn.TxnFinish()

		_, err = txn.Exec("INSERT INTO t1 (id) VALUES(1)")
		if err != nil {
			t.Fatalf("inserting failed: %v", err)
		}

		if err := txn.TxnRollback(); err != nil {
			t.Fatalf("rollback failed: %v", err)
		}
	}()

	row := dbm.QueryRow("SELECT id FROM t1 WHERE id = ?", 1)
	var id int
	if err = row.Scan(&id); err != sql.ErrNoRows {
		t.Fatalf("got %v\nwant ErrNoRows", err)
	}
}

func TestDo(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening database failed: %v", err)
	}
	defer db.Close()

	dbm := NewDbm(db)
	_, err = dbm.Exec(
		"CREATE TABLE t1 (id INTEGER PRIMARY KEY)",
	)
	if err != nil {
		t.Fatalf("creating test table failed: %v", err)
	}

	err = Do(dbm, func(txn Dbm) error {
		_, err := txn.Exec("INSERT INTO t1 (id) VALUES(1)")
		return err
	})
	if err != nil {
		t.Fatalf("do failed: %v", err)
	}

	row := dbm.QueryRow("SELECT id FROM t1 WHERE id = ?", 1)
	var id int
	if err = row.Scan(&id); err != nil {
		t.Fatalf("selecting row failed: %v", err)
	}
	if id != 1 {
		t.Errorf("got %d\nwant 1", id)
	}
}
