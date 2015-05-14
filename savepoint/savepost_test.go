package savepoint

import (
	"database/sql"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/shogo82148/txmanager"
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

func TestDo(t *testing.T) {
	db, err := setup()
	if err != nil {
		t.Fatalf("opening database failed: %v", err)
	}
	defer db.Close()

	dbm := NewDB(db)
	err = txmanager.Do(dbm, func(tx txmanager.Tx) error {
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

func TestDoRollback(t *testing.T) {
	db, err := setup()
	if err != nil {
		t.Fatalf("opening database failed: %v", err)
	}
	defer db.Close()

	dbm := NewDB(db)
	err = txmanager.Do(dbm, func(tx txmanager.Tx) error {
		_, err := tx.Exec("INSERT INTO t1 (id) VALUES(1)")
		if err != nil {
			return err
		}
		return errors.New("something wrong. rollback all change.")
	})
	if err == nil {
		t.Fatalf("got no error\nwant fail")
	}

	row := dbm.QueryRow("SELECT id FROM t1 WHERE id = ?", 1)
	var id int
	if err = row.Scan(&id); err != sql.ErrNoRows {
		t.Errorf("got %v\nwant ErrNoRows", err)
	}
}

func TestNestCommit(t *testing.T) {
	db, err := setup()
	if err != nil {
		t.Fatalf("opening database failed: %v", err)
	}
	defer db.Close()

	dbm := NewDB(db)
	err = txmanager.Do(dbm, func(tx1 txmanager.Tx) error {
		_, err := tx1.Exec("INSERT INTO t1 (id) VALUES(1)")
		if err != nil {
			return err
		}

		return txmanager.Do(tx1, func(tx2 txmanager.Tx) error {
			_, err := tx2.Exec("INSERT INTO t1 (id) VALUES(2)")
			return err
		})
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

	row = dbm.QueryRow("SELECT id FROM t1 WHERE id = ?", 2)
	if err = row.Scan(&id); err != nil {
		t.Fatalf("selecting row failed: %v", err)
	}
	if id != 2 {
		t.Errorf("got %d\nwant 2", id)
	}
}

func TestNestRollback(t *testing.T) {
	db, err := setup()
	if err != nil {
		t.Fatalf("opening database failed: %v", err)
	}
	defer db.Close()

	dbm := NewDB(db)
	err = txmanager.Do(dbm, func(tx1 txmanager.Tx) error {
		_, err := tx1.Exec("INSERT INTO t1 (id) VALUES(1)")
		if err != nil {
			t.Fatalf("intert failed: %v", err)
		}

		txmanager.Do(tx1, func(tx2 txmanager.Tx) error {
			_, err := tx2.Exec("INSERT INTO t1 (id) VALUES(2)")
			if err != nil {
				t.Fatalf("insert failed: %v", err)
			}
			return errors.New("something wrong. rollback change.")
		})

		_, err = tx1.Exec("INSERT INTO t1 (id) VALUES(3)")
		if err != nil {
			t.Fatalf("intert failed: %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("do failed: %v", err)
	}

	row := dbm.QueryRow("SELECT id FROM t1 WHERE id = ?", 1)
	var id int
	if err = row.Scan(&id); err != nil {
		t.Errorf("got %v\nwant no error", err)
	}

	row = dbm.QueryRow("SELECT id FROM t1 WHERE id = ?", 2)
	if err = row.Scan(&id); err != sql.ErrNoRows {
		t.Errorf("got %v\nwant ErrNoRows", err)
	}

	row = dbm.QueryRow("SELECT id FROM t1 WHERE id = ?", 3)
	if err = row.Scan(&id); err != nil {
		t.Errorf("got %v\nwant no error", err)
	}
}
