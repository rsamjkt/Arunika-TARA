# Templates

Template print dokumen (TIKET, SEP, REGISTRASI) di-embed di binary
dari folder:

```
internal/hardware/printer/templates/
```

Folder ini ada di dalam package `printer` supaya `//go:embed` bisa
mereach-nya — directive embed Go tidak boleh keluar dari package
direktori.

## Edit template

Edit file `.tmpl` di folder embed di atas, lalu rebuild:

```bash
make build-mac        # atau build-windows untuk production
```

## Template engine

Pakai `text/template` standard library Go. Field di-resolve dari
struct yang dipasok ke `printer.Print(ctx, docType, data any)`.

| docType       | Data shape (field yang harus ada)                    |
|---------------|------------------------------------------------------|
| `TIKET`       | `RSName`, `Tanggal`, `JenisAntrian`, `Nomor`         |
| `SEP`         | `NoSEP`, `Nama`, `NoKartu`, `NmPoli`, `NmDokter`,    |
|               | `TglSEP`, `KelasRawat`                               |
| `REGISTRASI`  | `NoRawat`, `Nama`, `NmPoli`, `NmDokter`,             |
|               | `TglKunjungan`, `NoAntrian`                          |

## Override runtime (future)

Saat ini template hanya embedded — tidak ada override runtime. Iterasi
berikutnya bisa support overlay dari path eksternal (mis.
`config.toml` `[printer] templates_dir = "./templates"`) untuk RS
yang mau customize tanpa rebuild binary.
