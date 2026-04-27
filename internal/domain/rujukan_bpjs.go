package domain

// RujukanBPJS adalah audit trail rujukan FKTP yang dipakai pasien BPJS
// untuk SEP — disimpan di tabel `bridging_rujukan_bpjs` (terpisah dari
// `rujuk_masuk` yang link ke no_rawat).
//
// Schema vendor (KhanzaHMSAnjunganSEP_RSAMXIP):
//   no_sep         FK ke bridging_sep
//   tglRujukan     date — tgl issue rujukan FKTP
//   tglRencanaKunjungan  date — tgl rencana ke RS rujukan
//   ppkDirujuk     varchar — kode RS yang dirujuk (kita)
//   nm_ppkDirujuk  varchar — nama RS
//   jnsPelayanan   enum '1' RJ / '2' RI
//   catatan        varchar — keterangan
//   diagRujukan    varchar — ICD-10 diagnosa rujukan
//   nama_diagRujukan varchar — deskripsi
//   tipeRujukan    enum '0. Penuh' / '1. Partial' / '2. Rujuk Balik'
//   poliRujukan    varchar — kode poli rujukan
//   nama_poliRujukan varchar
//   no_rujukan     PRIMARY KEY
//   user           varchar — petugas
type RujukanBPJS struct {
	NoSEP            string
	NoRujukan        string
	TglRujukan       string // "2006-01-02"
	TglRencana       string // "2006-01-02" — opsional
	PPKDirujuk       string // kode RS tujuan
	NmPPKDirujuk     string
	JnsPelayanan     string // "1" / "2"
	Catatan          string
	DiagRujukan      string // ICD-10
	NmDiagRujukan    string
	TipeRujukan      string // "0. Penuh" / "1. Partial" / "2. Rujuk Balik"
	PoliRujukan      string
	NmPoliRujukan    string
	User             string
}
