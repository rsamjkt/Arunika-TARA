// Live integration test untuk VClaim client — hit server BPJS dev langsung.
// Jalankan: go run ./cmd/vclaim-live
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/arunika/apm-go/internal/config"
	"github.com/arunika/apm-go/internal/integration/vclaim"
)

var devNoKartu = []string{
	"0002076061241",
	"0002053875677",
	"0002076678325",
	"0002076680182",
	"0002070402456",
}

func main() {
	cfgPath := "config.toml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
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

	fmt.Printf("VClaim URL : %s\n", cfg.BPJS.VClaimURL)
	fmt.Printf("cons_id    : %s\n", cfg.BPJS.ConsID)
	fmt.Println()

	for _, noka := range devNoKartu {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		peserta, err := client.GetPeserta(ctx, noka, time.Now())
		cancel()

		if err != nil {
			fmt.Printf("[%s] ERROR: %v\n", noka, err)
		} else {
			fmt.Printf("[%s] OK: %s | NIK: %s | Kelas: %s | Status: %s\n",
				noka, peserta.Nama, peserta.NIK, peserta.KelasHak, peserta.StatusAktif)
			return // satu sukses sudah cukup
		}
	}
}
