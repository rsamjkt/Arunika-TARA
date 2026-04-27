package domain

// RujukanInternalPoli adalah hasil dari `rujukan_internal_poli` Khanza —
// pasien yang baru selesai kunjungan di satu poli kemudian dirujuk
// ke poli lain (kontrol antar-poli internal RS).
//
// Berbeda dengan SuratKontrol (yang BPJS-side, dari bridging_surat_kontrol_bpjs):
// rujukan internal poli = dorongan dari dokter ke poli lain dalam RS yang sama,
// tanpa perlu SEP baru dari BPJS (selama masih dalam episode pelayanan).
//
// Dipakai detector PostRAJAL untuk auto-isi poli tujuan + dokter tujuan
// di screen pendaftaran lanjutan.
type RujukanInternalPoli struct {
	NoRawatAsal      string // no_rawat dari kunjungan poli asal
	TglKunjunganAsal string // tgl_registrasi dari kunjungan asal — "2006-01-02"
	KdPoliAsal       string
	NmPoliAsal       string
	KdPoliTujuan     string // poli baru yang dirujuk
	NmPoliTujuan     string
	KdDokterTujuan   string
	NmDokterTujuan   string
}
