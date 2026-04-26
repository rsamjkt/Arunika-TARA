-- Migration 001 -- initial schema untuk APM (T.A.R.A) SQLite store.
-- Engine: SQLite 3 (CGO via mattn/go-sqlite3).
-- Skema ini juga dipakai oleh sqlc generate (lihat sqlc.yaml di root).

-- ============================================================
-- Tabel: antrian_lokal
-- Buffer antrian saat Khanza offline. Reconcile worker akan
-- mem-flush record dengan sync_status='pending' ke Khanza saat
-- koneksi pulih.
-- ============================================================
CREATE TABLE IF NOT EXISTS antrian_lokal (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    jenis       TEXT    NOT NULL,
    sub_jenis   TEXT,
    nomor       TEXT    NOT NULL,
    prefix      TEXT    NOT NULL,
    no_urut     INTEGER NOT NULL,
    no_rm       TEXT,
    no_poli     TEXT,
    created_at  DATETIME DEFAULT (datetime('now','localtime')),
    synced_at   DATETIME,
    sync_status TEXT     DEFAULT 'pending',
    -- Reconcile worker retry tracking (P-050).
    -- retry_count: increment tiap attempt sync gagal.
    -- last_error: pesan error terakhir untuk audit/debug.
    retry_count INTEGER DEFAULT 0,
    last_error  TEXT
);

CREATE INDEX IF NOT EXISTS idx_antrian_sync_status
    ON antrian_lokal (sync_status);

-- ============================================================
-- Tabel: pending_sep
-- SEP yang gagal di-submit ke Khanza atau menunggu konfirmasi
-- operator (mis. saat partial network failure). Dipakai admin
-- panel + reconcile worker.
-- ============================================================
CREATE TABLE IF NOT EXISTS pending_sep (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    no_kartu        TEXT    NOT NULL,
    kategori        TEXT    NOT NULL,
    payload_json    TEXT    NOT NULL,
    vclaim_response TEXT,
    status          TEXT    DEFAULT 'pending',
    retry_count     INTEGER DEFAULT 0,
    last_error      TEXT,
    created_at      DATETIME DEFAULT (datetime('now','localtime')),
    confirmed_by    TEXT,
    confirmed_at    DATETIME
);

CREATE INDEX IF NOT EXISTS idx_pending_sep_status
    ON pending_sep (status);

-- ============================================================
-- Tabel: print_history
-- Setiap dokumen yang dicetak. Dipakai untuk reprint dari
-- kiosk atau admin panel.
-- ============================================================
CREATE TABLE IF NOT EXISTS print_history (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    doc_type      TEXT    NOT NULL,
    ref_id        TEXT,
    escpos_bytes  BLOB    NOT NULL,
    printed_at    DATETIME DEFAULT (datetime('now','localtime')),
    reprint_count INTEGER  DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_print_history_doc_type
    ON print_history (doc_type, printed_at DESC);

-- ============================================================
-- Tabel: reconcile_log
-- Audit trail tiap aksi rekonsiliasi & operasi admin.
-- Append-only.
-- ============================================================
CREATE TABLE IF NOT EXISTS reconcile_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    table_name  TEXT    NOT NULL,
    record_id   INTEGER NOT NULL,
    action      TEXT    NOT NULL,
    operator_id TEXT,
    result      TEXT,
    timestamp   DATETIME DEFAULT (datetime('now','localtime'))
);

CREATE INDEX IF NOT EXISTS idx_reconcile_log_timestamp
    ON reconcile_log (timestamp DESC);

-- ============================================================
-- Tabel: config_cache
-- Cache key-value untuk lookup yang sering & jadwal dokter
-- (refresh harian).
-- ============================================================
CREATE TABLE IF NOT EXISTS config_cache (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at DATETIME DEFAULT (datetime('now','localtime'))
);
