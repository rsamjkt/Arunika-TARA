package detector

// maskID memendekkan identifier 16-digit (NIK / No Kartu / No RM)
// menjadi format "************XXXX" dengan hanya 4 digit terakhir
// terlihat. Untuk identifier <8 char, return "***" supaya tidak ada
// PHI yang bocor ke log.
//
// Helper ini lokal untuk paket detector. Solusi global PHI masking
// (slog handler) di-implement di P-051 — ketika itu siap, helper
// ini bisa di-deprecate dan delegate ke handler global.
func maskID(id string) string {
	if len(id) < 8 {
		return "***"
	}
	masked := make([]byte, len(id))
	for i := range masked {
		if i >= len(id)-4 {
			masked[i] = id[i]
		} else {
			masked[i] = '*'
		}
	}
	return string(masked)
}
