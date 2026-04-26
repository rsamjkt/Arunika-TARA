-- ============================================================
-- antrian_lokal -- buffer offline antrian
-- ============================================================

-- name: InsertAntrian :one
INSERT INTO antrian_lokal (jenis, sub_jenis, nomor, prefix, no_urut, no_rm, no_poli)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetPendingAntrian :many
SELECT *
FROM antrian_lokal
WHERE sync_status = 'pending'
ORDER BY created_at ASC
LIMIT ?;

-- name: MarkAntrianSynced :exec
UPDATE antrian_lokal
SET sync_status = 'synced',
    synced_at   = datetime('now','localtime')
WHERE id = ?;

-- name: MarkAntrianFailed :exec
UPDATE antrian_lokal
SET sync_status = 'failed'
WHERE id = ?;

-- ============================================================
-- pending_sep -- SEP yang menunggu sync ke Khanza
-- ============================================================

-- name: InsertPendingSEP :one
INSERT INTO pending_sep (no_kartu, kategori, payload_json, vclaim_response)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetPendingSEPs :many
SELECT *
FROM pending_sep
WHERE status = ?
ORDER BY created_at ASC
LIMIT ?;

-- name: ConfirmSEP :exec
UPDATE pending_sep
SET status       = 'awaiting_sync',
    confirmed_by = ?,
    confirmed_at = datetime('now','localtime')
WHERE id = ?;

-- name: MarkSEPSynced :exec
UPDATE pending_sep
SET status = 'synced'
WHERE id = ?;

-- name: IncrementSEPRetry :exec
UPDATE pending_sep
SET retry_count = retry_count + 1,
    last_error  = ?
WHERE id = ?;

-- ============================================================
-- print_history -- log dokumen yang dicetak (untuk reprint)
-- ============================================================

-- name: InsertPrintHistory :one
INSERT INTO print_history (doc_type, ref_id, escpos_bytes)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetPrintHistory :one
SELECT *
FROM print_history
WHERE id = ?;

-- name: GetPrintHistoryByRefID :one
SELECT *
FROM print_history
WHERE doc_type = ? AND ref_id = ?
ORDER BY printed_at DESC
LIMIT 1;

-- name: IncrementReprintCount :exec
UPDATE print_history
SET reprint_count = reprint_count + 1
WHERE id = ?;

-- ============================================================
-- reconcile_log -- audit trail
-- ============================================================

-- name: InsertReconcileLog :one
INSERT INTO reconcile_log (table_name, record_id, action, operator_id, result)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetRecentLogs :many
-- Tiebreaker id DESC karena datetime('now','localtime') hanya resolusi detik,
-- multiple insert dalam detik yang sama bisa punya timestamp identik.
SELECT *
FROM reconcile_log
ORDER BY timestamp DESC, id DESC
LIMIT ?;
