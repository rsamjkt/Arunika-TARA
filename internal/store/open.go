package store

import (
	"context"
	"database/sql"
	"fmt"

	// Driver SQLite via CGO. mattn/go-sqlite3 mendaftarkan dirinya
	// sebagai "sqlite3" via init().
	_ "github.com/mattn/go-sqlite3"
)

// Open membuka koneksi SQLite di path yang diberikan dan menjalankan
// schemaSQL untuk memastikan tabel sudah ada. Path ":memory:" dipakai
// untuk testing dengan in-memory database.
//
// schemaSQL biasanya berisi isi migrations/001_initial.sql yang dibaca
// caller (main, test) via os.ReadFile. Pisahkan baca file dari open
// supaya store tidak memaksakan layout filesystem tertentu.
//
// PRAGMA yang di-set lewat DSN:
//   - _foreign_keys=on
//   - _busy_timeout=5000ms
//   - _journal_mode=WAL  (hanya untuk file-backed DB)
func Open(ctx context.Context, path, schemaSQL string) (*sql.DB, *Queries, error) {
	dsn := buildDSN(path)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("buka sqlite di %q: %w", path, err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if schemaSQL != "" {
		if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
			_ = db.Close()
			return nil, nil, fmt.Errorf("apply schema: %w", err)
		}
	}
	return db, New(db), nil
}

func buildDSN(path string) string {
	if path == ":memory:" {
		return ":memory:?_foreign_keys=on&_busy_timeout=5000"
	}
	return path + "?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000"
}
