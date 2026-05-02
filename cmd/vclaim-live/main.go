// Live integration test untuk VClaim client — hit server BPJS langsung.
// Jalankan: go run ./cmd/vclaim-live
// Atau:     go run ./cmd/vclaim-live <config.toml> [noKartu]
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/arunika/apm-go/internal/config"
	"github.com/arunika/apm-go/internal/integration/vclaim"
)

func main() {
	cfgPath := "config.toml"
	noKartu := "0002259275861"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	if len(os.Args) > 2 {
		noKartu = os.Args[2]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	if cfg.BPJS.Mock {
		fmt.Fprintln(os.Stderr, "config: bpjs.mock = true — ubah ke false")
		os.Exit(1)
	}

	client := vclaim.New(cfg.BPJS)
	fmt.Printf("URL     : %s\n", cfg.BPJS.VClaimURL)
	fmt.Printf("cons_id : %s\n", cfg.BPJS.ConsID)
	fmt.Printf("noKartu : %s\n\n", noKartu)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	peserta, err := client.GetPeserta(ctx, noKartu, time.Now())
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== BERHASIL ===")
	fmt.Printf("Nama       : %s\n", peserta.Nama)
	fmt.Printf("NIK        : %s\n", peserta.NIK)
	fmt.Printf("NoKartu    : %s\n", peserta.NoKartu)
	fmt.Printf("Kelas Hak  : %s\n", peserta.KelasHak)
	fmt.Printf("Status     : %s\n", peserta.StatusAktif)
	fmt.Printf("Jenis      : %s\n", peserta.JenisPeserta)
}
