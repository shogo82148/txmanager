package txnmanager

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setup() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(
		"CREATE TABLE t1 (id INTEGER PRIMARY KEY)",
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func TestCommit(t *testing.T) {
	db, err := setup()
	if err != nil {
		t.Fatalf("opening database failed: %v", err)
	}
	defer db.Close()

	dbm := NewDbm(db)
	func() {
		tx, err := dbm.TxBegin()
		if err != nil {
			t.Fatalf("beginning transaction failed: %v", err)
		}
		defer tx.TxFinish()

		_, err = tx.Exec("INSERT INTO t1 (id) VALUES(1)")
		if err != nil {
			t.Fatalf("inserting failed: %v", err)
		}

		if err := tx.TxCommit(); err != nil {
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
	db, err := setup()
	if err != nil {
		t.Fatalf("opening database failed: %v", err)
	}
	defer db.Close()

	dbm := NewDbm(db)
	func() {
		tx, err := dbm.TxBegin()
		if err != nil {
			t.Fatalf("beginning transaction failed: %v", err)
		}
		defer tx.TxFinish()

		_, err = tx.Exec("INSERT INTO t1 (id) VALUES(1)")
		if err != nil {
			t.Fatalf("inserting failed: %v", err)
		}

		if err := tx.TxRollback(); err != nil {
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
	db, err := setup()
	if err != nil {
		t.Fatalf("opening database failed: %v", err)
	}
	defer db.Close()

	dbm := NewDbm(db)
	err = Do(dbm, func(tx Dbm) error {
		_, err := tx.Exec("INSERT INTO t1 (id) VALUES(1)")
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
