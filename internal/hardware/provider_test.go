package hardware

import (
	"runtime"
	"testing"

	"github.com/arunika/apm-go/internal/config"
)

func TestNewProvider_FieldsNotNil(t *testing.T) {
	p := NewProvider(config.Config{
		Dev: config.DevConfig{MockServerPort: 9090},
	}, nil)

	if p == nil {
		t.Fatal("NewProvider returned nil")
	}
	if p.Frista == nil {
		t.Error("Provider.Frista nil")
	}
	if p.Fingerprint == nil {
		t.Error("Provider.Fingerprint nil")
	}
	if p.Printer == nil {
		t.Error("Provider.Printer nil")
	}
}

func TestNewProvider_PlatformLabel(t *testing.T) {
	p := NewProvider(config.Config{}, nil)
	if got := p.Platform(); got != runtime.GOOS {
		t.Errorf("Platform() = %q, want %q", got, runtime.GOOS)
	}
}

func TestNewProvider_IsRealHardware(t *testing.T) {
	p := NewProvider(config.Config{}, nil)
	want := runtime.GOOS == "windows"
	if got := p.IsRealHardware(); got != want {
		t.Errorf("IsRealHardware() on %s = %v, want %v",
			runtime.GOOS, got, want)
	}
}

func TestNewProvider_AllInterfacesAvailable(t *testing.T) {
	p := NewProvider(config.Config{
		Dev: config.DevConfig{MockServerPort: 9090},
	}, nil)

	// Mocks default to available; Windows stubs juga delegasi ke mocks
	if !p.Frista.IsAvailable() {
		t.Error("Frista should be available right after init")
	}
	if !p.Fingerprint.IsAvailable() {
		t.Error("Fingerprint should be available right after init")
	}
	if !p.Printer.IsAvailable() {
		t.Error("Printer should be available right after init")
	}
}
