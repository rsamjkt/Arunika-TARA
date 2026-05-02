//go:build windows

// Quick smoke-test untuk Frista dan After.exe hardware integration.
// Jalankan: go run ./cmd/test-hw
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/arunika/apm-go/internal/config"
	"github.com/arunika/apm-go/internal/hardware/frista"
	"github.com/arunika/apm-go/internal/hardware/fingerprint"
)

const testNoPeserta = "0001234567890012"

func main() {
	mode := "frista"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	switch mode {
	case "frista":
		testFrista()
	case "fp":
		testFingerprint()
	default:
		fmt.Println("Usage: test-hw [frista|fp]")
	}
}

func testFrista() {
	fmt.Println("=== TEST FRISTA ===")
	cfg := config.FristaConfig{
		ExePath:         `C:\Program Files\Frista\frista.exe`,
		UsernameEnc:     "0115R050_novi",
		PasswordEnc:     "Bpjs1234#",
		StartupDelaySec: 7,
		ScanTimeoutSec:  60,
	}

	v := frista.NewWindowsHeadless(cfg)
	fmt.Printf("IsAvailable: %v\n", v.IsAvailable())

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	fmt.Printf("Spawning frista.exe + injecting creds + noPeserta=%s...\n", testNoPeserta)
	start := time.Now()
	res, err := v.Verify(ctx, testNoPeserta)
	elapsed := time.Since(start).Round(time.Millisecond)

	if err != nil {
		fmt.Printf("FAIL (%s): %v\n", elapsed, err)
		os.Exit(1)
	}
	fmt.Printf("OK (%s): Success=%v Token=%s\n", elapsed, res.Success, res.Token)
	// Beri waktu 5 detik untuk observasi window Frista sebelum kill
	time.Sleep(5 * time.Second)
	_ = v.(interface{ Stop() error }).Stop()
	fmt.Println("Frista killed.")
}

func testFingerprint() {
	fmt.Println("=== TEST AFTER.EXE ===")
	cfg := config.FingerprintConfig{
		ExePath:         `C:\Program Files (x86)\Aplikasi Sidik Jari BPJS Kesehatan\After.exe`,
		UsernameEnc:     "0115R050_novi",
		PasswordEnc:     "Bpjs1234#",
		StartupDelaySec: 5,
		ScanTimeoutSec:  30,
	}

	v := fingerprint.NewWindowsHeadless(cfg)
	fmt.Printf("IsAvailable: %v\n", v.IsAvailable())

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Printf("Spawning After.exe + inject login + start scan noPeserta=%s...\n", testNoPeserta)
	start := time.Now()
	res, err := v.Verify(ctx, testNoPeserta)
	elapsed := time.Since(start).Round(time.Millisecond)

	if err != nil {
		fmt.Printf("FAIL (%s): %v\n", elapsed, err)
		os.Exit(1)
	}
	fmt.Printf("OK (%s): Success=%v Token=%s\n", elapsed, res.Success, res.Token)
	_ = v.(interface{ Stop() error }).Stop()
}
