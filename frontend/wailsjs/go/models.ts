export namespace domain {
	
	export class Peserta {
	    NoKartu: string;
	    NoRM: string;
	    NIK: string;
	    Nama: string;
	    TglLahir: string;
	    StatusAktif: string;
	    KelasHak: string;
	    JenisPeserta: string;
	
	    static createFrom(source: any = {}) {
	        return new Peserta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.NoKartu = source["NoKartu"];
	        this.NoRM = source["NoRM"];
	        this.NIK = source["NIK"];
	        this.Nama = source["Nama"];
	        this.TglLahir = source["TglLahir"];
	        this.StatusAktif = source["StatusAktif"];
	        this.KelasHak = source["KelasHak"];
	        this.JenisPeserta = source["JenisPeserta"];
	    }
	}
	export class DetectionResult {
	    Type: number;
	    Peserta?: Peserta;
	    Data: any;
	    Err: any;
	    // Go type: time
	    DetectedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new DetectionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Type = source["Type"];
	        this.Peserta = this.convertValues(source["Peserta"], Peserta);
	        this.Data = source["Data"];
	        this.Err = source["Err"];
	        this.DetectedAt = this.convertValues(source["DetectedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class JadwalDokter {
	    KdDokter: string;
	    NmDokter: string;
	    KdPoli: string;
	    NmPoli: string;
	    Hari: string;
	    JamMulai: string;
	    JamSelesai: string;
	    Kuota: number;
	    Sisa: number;
	    Aktif: boolean;
	
	    static createFrom(source: any = {}) {
	        return new JadwalDokter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.KdDokter = source["KdDokter"];
	        this.NmDokter = source["NmDokter"];
	        this.KdPoli = source["KdPoli"];
	        this.NmPoli = source["NmPoli"];
	        this.Hari = source["Hari"];
	        this.JamMulai = source["JamMulai"];
	        this.JamSelesai = source["JamSelesai"];
	        this.Kuota = source["Kuota"];
	        this.Sisa = source["Sisa"];
	        this.Aktif = source["Aktif"];
	    }
	}
	export class Pasien {
	    NoRM: string;
	    Nama: string;
	    NIK: string;
	    NoKartu: string;
	    TglLahir: string;
	    JK: string;
	    Alamat: string;
	    NoTelp: string;
	    IhsNumber: string;
	
	    static createFrom(source: any = {}) {
	        return new Pasien(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.NoRM = source["NoRM"];
	        this.Nama = source["Nama"];
	        this.NIK = source["NIK"];
	        this.NoKartu = source["NoKartu"];
	        this.TglLahir = source["TglLahir"];
	        this.JK = source["JK"];
	        this.Alamat = source["Alamat"];
	        this.NoTelp = source["NoTelp"];
	        this.IhsNumber = source["IhsNumber"];
	    }
	}
	export class Pendaftaran {
	    NoRawat: string;
	    NoRM: string;
	    KdPoli: string;
	    NmPoli: string;
	    KdDokter: string;
	    NmDokter: string;
	    TglPeriksa: string;
	    NoUrut: number;
	
	    static createFrom(source: any = {}) {
	        return new Pendaftaran(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.NoRawat = source["NoRawat"];
	        this.NoRM = source["NoRM"];
	        this.KdPoli = source["KdPoli"];
	        this.NmPoli = source["NmPoli"];
	        this.KdDokter = source["KdDokter"];
	        this.NmDokter = source["NmDokter"];
	        this.TglPeriksa = source["TglPeriksa"];
	        this.NoUrut = source["NoUrut"];
	    }
	}
	export class PendaftaranRequest {
	    NoRM: string;
	    KdPoli: string;
	    KdDokter: string;
	    TglPeriksa: string;
	    JamPeriksa: string;
	    Penjamin: string;
	    NoSEP: string;
	    Catatan: string;
	    PJawab: string;
	    AlmtPJ: string;
	    HubunganPJ: string;
	
	    static createFrom(source: any = {}) {
	        return new PendaftaranRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.NoRM = source["NoRM"];
	        this.KdPoli = source["KdPoli"];
	        this.KdDokter = source["KdDokter"];
	        this.TglPeriksa = source["TglPeriksa"];
	        this.JamPeriksa = source["JamPeriksa"];
	        this.Penjamin = source["Penjamin"];
	        this.NoSEP = source["NoSEP"];
	        this.Catatan = source["Catatan"];
	        this.PJawab = source["PJawab"];
	        this.AlmtPJ = source["AlmtPJ"];
	        this.HubunganPJ = source["HubunganPJ"];
	    }
	}
	
	export class Poliklinik {
	    kd_poli: string;
	    nm_poli: string;
	    registrasi: number;
	    registrasi_lama: number;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new Poliklinik(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kd_poli = source["kd_poli"];
	        this.nm_poli = source["nm_poli"];
	        this.registrasi = source["registrasi"];
	        this.registrasi_lama = source["registrasi_lama"];
	        this.status = source["status"];
	    }
	}
	export class SEP {
	    NoSEP: string;
	    NoKartu: string;
	    TglSEP: string;
	    KdPoli: string;
	    NmPoli: string;
	    KdDokter: string;
	    NmDokter: string;
	    // Go type: time
	    CreatedAt: any;
	    NoRujukan: string;
	    TglRujukan: string;
	    KdPPKRujukan: string;
	    NmPPKRujukan: string;
	    AsalRujukan: string;
	    DiagnosaAwal: string;
	    NamaDiagnosa: string;
	    JenisPelayanan: string;
	    KelasRawat: string;
	    NoSKDP: string;
	    KdDPJP: string;
	    NmDPJP: string;
	    NoMR: string;
	    NamaPasien: string;
	    PRBCode: string;
	    LakaLantas: string;
	    TglKejadian: string;
	    KetKecelakaan: string;
	    KdPropinsi: string;
	    NmPropinsi: string;
	    KdKabupaten: string;
	    NmKabupaten: string;
	    KdKecamatan: string;
	    NmKecamatan: string;
	    COB: string;
	    Eksekutif: string;
	    TujuanKunjungan: string;
	    AsesmenPelayanan: string;
	
	    static createFrom(source: any = {}) {
	        return new SEP(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.NoSEP = source["NoSEP"];
	        this.NoKartu = source["NoKartu"];
	        this.TglSEP = source["TglSEP"];
	        this.KdPoli = source["KdPoli"];
	        this.NmPoli = source["NmPoli"];
	        this.KdDokter = source["KdDokter"];
	        this.NmDokter = source["NmDokter"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.NoRujukan = source["NoRujukan"];
	        this.TglRujukan = source["TglRujukan"];
	        this.KdPPKRujukan = source["KdPPKRujukan"];
	        this.NmPPKRujukan = source["NmPPKRujukan"];
	        this.AsalRujukan = source["AsalRujukan"];
	        this.DiagnosaAwal = source["DiagnosaAwal"];
	        this.NamaDiagnosa = source["NamaDiagnosa"];
	        this.JenisPelayanan = source["JenisPelayanan"];
	        this.KelasRawat = source["KelasRawat"];
	        this.NoSKDP = source["NoSKDP"];
	        this.KdDPJP = source["KdDPJP"];
	        this.NmDPJP = source["NmDPJP"];
	        this.NoMR = source["NoMR"];
	        this.NamaPasien = source["NamaPasien"];
	        this.PRBCode = source["PRBCode"];
	        this.LakaLantas = source["LakaLantas"];
	        this.TglKejadian = source["TglKejadian"];
	        this.KetKecelakaan = source["KetKecelakaan"];
	        this.KdPropinsi = source["KdPropinsi"];
	        this.NmPropinsi = source["NmPropinsi"];
	        this.KdKabupaten = source["KdKabupaten"];
	        this.NmKabupaten = source["NmKabupaten"];
	        this.KdKecamatan = source["KdKecamatan"];
	        this.NmKecamatan = source["NmKecamatan"];
	        this.COB = source["COB"];
	        this.Eksekutif = source["Eksekutif"];
	        this.TujuanKunjungan = source["TujuanKunjungan"];
	        this.AsesmenPelayanan = source["AsesmenPelayanan"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SEPRequest {
	    NoKartu: string;
	    TglSEP: string;
	    KdPoli: string;
	    KdDokter: string;
	    JnsPelayanan: string;
	    KelasRawat: string;
	    NoRujukan: string;
	    CatatanPelayanan: string;
	    FPToken: string;
	    BiometrikToken: string;
	    TglRujukan: string;
	    KdPPKRujukan: string;
	    NmPPKRujukan: string;
	    AsalRujukan: string;
	    DiagnosaAwal: string;
	    NamaDiagnosa: string;
	    NoMR: string;
	    NoTelp: string;
	    NoSKDP: string;
	    KdDPJP: string;
	    Eksekutif: string;
	    COB: string;
	    Katarak: string;
	    LakaLantas: string;
	    TglKejadian: string;
	    KetKecelakaan: string;
	    Suplesi: string;
	    NoSepSuplesi: string;
	    KdPropinsi: string;
	    NmPropinsi: string;
	    KdKabupaten: string;
	    NmKabupaten: string;
	    KdKecamatan: string;
	    NmKecamatan: string;
	    TujuanKunjungan: string;
	    FlagProcedure: string;
	    KdPenunjang: string;
	    AsesmenPelayanan: string;
	    User: string;
	
	    static createFrom(source: any = {}) {
	        return new SEPRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.NoKartu = source["NoKartu"];
	        this.TglSEP = source["TglSEP"];
	        this.KdPoli = source["KdPoli"];
	        this.KdDokter = source["KdDokter"];
	        this.JnsPelayanan = source["JnsPelayanan"];
	        this.KelasRawat = source["KelasRawat"];
	        this.NoRujukan = source["NoRujukan"];
	        this.CatatanPelayanan = source["CatatanPelayanan"];
	        this.FPToken = source["FPToken"];
	        this.BiometrikToken = source["BiometrikToken"];
	        this.TglRujukan = source["TglRujukan"];
	        this.KdPPKRujukan = source["KdPPKRujukan"];
	        this.NmPPKRujukan = source["NmPPKRujukan"];
	        this.AsalRujukan = source["AsalRujukan"];
	        this.DiagnosaAwal = source["DiagnosaAwal"];
	        this.NamaDiagnosa = source["NamaDiagnosa"];
	        this.NoMR = source["NoMR"];
	        this.NoTelp = source["NoTelp"];
	        this.NoSKDP = source["NoSKDP"];
	        this.KdDPJP = source["KdDPJP"];
	        this.Eksekutif = source["Eksekutif"];
	        this.COB = source["COB"];
	        this.Katarak = source["Katarak"];
	        this.LakaLantas = source["LakaLantas"];
	        this.TglKejadian = source["TglKejadian"];
	        this.KetKecelakaan = source["KetKecelakaan"];
	        this.Suplesi = source["Suplesi"];
	        this.NoSepSuplesi = source["NoSepSuplesi"];
	        this.KdPropinsi = source["KdPropinsi"];
	        this.NmPropinsi = source["NmPropinsi"];
	        this.KdKabupaten = source["KdKabupaten"];
	        this.NmKabupaten = source["NmKabupaten"];
	        this.KdKecamatan = source["KdKecamatan"];
	        this.NmKecamatan = source["NmKecamatan"];
	        this.TujuanKunjungan = source["TujuanKunjungan"];
	        this.FlagProcedure = source["FlagProcedure"];
	        this.KdPenunjang = source["KdPenunjang"];
	        this.AsesmenPelayanan = source["AsesmenPelayanan"];
	        this.User = source["User"];
	    }
	}
	export class Ticket {
	    ID: string;
	    Nomor: string;
	    Jenis: string;
	    SubJenis: string;
	    Prefix: string;
	    NoUrut: number;
	    NoRM: string;
	    NoPoli: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    PrintedAt?: any;
	    IsOffline: boolean;
	    PrintHistoryID: number;
	
	    static createFrom(source: any = {}) {
	        return new Ticket(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Nomor = source["Nomor"];
	        this.Jenis = source["Jenis"];
	        this.SubJenis = source["SubJenis"];
	        this.Prefix = source["Prefix"];
	        this.NoUrut = source["NoUrut"];
	        this.NoRM = source["NoRM"];
	        this.NoPoli = source["NoPoli"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.PrintedAt = this.convertValues(source["PrintedAt"], null);
	        this.IsOffline = source["IsOffline"];
	        this.PrintHistoryID = source["PrintHistoryID"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class UpdateStatus {
	    enabled: boolean;
	    available: boolean;
	    current_version: string;
	    latest_version: string;
	    release_notes: string;
	    asset_size: number;
	    published_at: string;

	    static createFrom(source: any = {}) {
	        return new UpdateStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.available = source["available"];
	        this.current_version = source["current_version"];
	        this.latest_version = source["latest_version"];
	        this.release_notes = source["release_notes"];
	        this.asset_size = source["asset_size"];
	        this.published_at = source["published_at"];
	    }
	}
	export class AdminLogEntry {
	    id: number;
	    table_name: string;
	    record_id: number;
	    action: string;
	    operator_id: string;
	    result: string;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new AdminLogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.table_name = source["table_name"];
	        this.record_id = source["record_id"];
	        this.action = source["action"];
	        this.operator_id = source["operator_id"];
	        this.result = source["result"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class AdminStats {
	    antrian_hari_ini: number;
	    sep_hari_ini: number;
	    pending_sync: number;
	    uptime_sec: number;
	    started_at: string;
	
	    static createFrom(source: any = {}) {
	        return new AdminStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.antrian_hari_ini = source["antrian_hari_ini"];
	        this.sep_hari_ini = source["sep_hari_ini"];
	        this.pending_sync = source["pending_sync"];
	        this.uptime_sec = source["uptime_sec"];
	        this.started_at = source["started_at"];
	    }
	}
	export class Branding {
	    hospital_name: string;
	    hospital_tagline: string;
	    logo_path: string;
	    logo_data_url: string;
	    primary_color: string;
	    primary_color_dark: string;
	    accent_color: string;
	    bpjs_logo_path: string;
	    bpjs_logo_data_url: string;
	    audio_enabled: boolean;
	    audio_volume: number;
	
	    static createFrom(source: any = {}) {
	        return new Branding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hospital_name = source["hospital_name"];
	        this.hospital_tagline = source["hospital_tagline"];
	        this.logo_path = source["logo_path"];
	        this.logo_data_url = source["logo_data_url"];
	        this.primary_color = source["primary_color"];
	        this.primary_color_dark = source["primary_color_dark"];
	        this.accent_color = source["accent_color"];
	        this.bpjs_logo_path = source["bpjs_logo_path"];
	        this.bpjs_logo_data_url = source["bpjs_logo_data_url"];
	        this.audio_enabled = source["audio_enabled"];
	        this.audio_volume = source["audio_volume"];
	    }
	}
	export class CheckResult {
	    component: string;
	    status: string;
	    message: string;
	    critical: boolean;
	    duration_ms: number;
	
	    static createFrom(source: any = {}) {
	        return new CheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.component = source["component"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.critical = source["critical"];
	        this.duration_ms = source["duration_ms"];
	    }
	}
	export class HardwareStatus {
	    frista: boolean;
	    fingerprint: boolean;
	    printer: boolean;
	
	    static createFrom(source: any = {}) {
	        return new HardwareStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.frista = source["frista"];
	        this.fingerprint = source["fingerprint"];
	        this.printer = source["printer"];
	    }
	}
	export class SystemStatus {
	    hardware: HardwareStatus;
	    online: boolean;
	    platform: string;
	    version: string;
	    uptime_sec: number;
	    started_at: string;
	
	    static createFrom(source: any = {}) {
	        return new SystemStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hardware = this.convertValues(source["hardware"], HardwareStatus);
	        this.online = source["online"];
	        this.platform = source["platform"];
	        this.version = source["version"];
	        this.uptime_sec = source["uptime_sec"];
	        this.started_at = source["started_at"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace sql {
	
	export class NullInt64 {
	    Int64: number;
	    Valid: boolean;
	
	    static createFrom(source: any = {}) {
	        return new NullInt64(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Int64 = source["Int64"];
	        this.Valid = source["Valid"];
	    }
	}
	export class NullString {
	    String: string;
	    Valid: boolean;
	
	    static createFrom(source: any = {}) {
	        return new NullString(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.String = source["String"];
	        this.Valid = source["Valid"];
	    }
	}
	export class NullTime {
	    // Go type: time
	    Time: any;
	    Valid: boolean;
	
	    static createFrom(source: any = {}) {
	        return new NullTime(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Time = this.convertValues(source["Time"], null);
	        this.Valid = source["Valid"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace store {
	
	export class PendingSep {
	    id: number;
	    no_kartu: string;
	    kategori: string;
	    payload_json: string;
	    vclaim_response: sql.NullString;
	    status: sql.NullString;
	    retry_count: sql.NullInt64;
	    last_error: sql.NullString;
	    created_at: sql.NullTime;
	    confirmed_by: sql.NullString;
	    confirmed_at: sql.NullTime;
	
	    static createFrom(source: any = {}) {
	        return new PendingSep(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.no_kartu = source["no_kartu"];
	        this.kategori = source["kategori"];
	        this.payload_json = source["payload_json"];
	        this.vclaim_response = this.convertValues(source["vclaim_response"], sql.NullString);
	        this.status = this.convertValues(source["status"], sql.NullString);
	        this.retry_count = this.convertValues(source["retry_count"], sql.NullInt64);
	        this.last_error = this.convertValues(source["last_error"], sql.NullString);
	        this.created_at = this.convertValues(source["created_at"], sql.NullTime);
	        this.confirmed_by = this.convertValues(source["confirmed_by"], sql.NullString);
	        this.confirmed_at = this.convertValues(source["confirmed_at"], sql.NullTime);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

