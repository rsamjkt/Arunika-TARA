.PHONY: help dev build-mac build-windows test test-coverage lint clean setup-cross-compile mock-card-read mock-fp-fail

# Default — list semua target
help:
	@echo "APM (T.A.R.A) — Make targets"
	@echo ""
	@echo "  make dev               Hot reload Vue + Go (wails dev)"
	@echo "  make build-mac         Build aplikasi untuk macOS (native)"
	@echo "  make build-windows     Cross-compile ke Windows dari Mac (butuh mingw-w64)"
	@echo "  make test              Jalankan unit tests semua package"
	@echo "  make test-coverage     Tests dengan coverage report (target ≥80%)"
	@echo "  make lint              golangci-lint"
	@echo "  make clean             Hapus build artifacts"
	@echo ""
	@echo "  make mock-card-read NIK=... NAMA=...   Simulasi tap KTP via Frista mock"
	@echo "  make mock-fp-fail                       Force fingerprint gagal sekali"
	@echo ""
	@echo "  make setup-cross-compile               Install mingw-w64 toolchain (butuh brew)"

dev:
	wails dev

build-mac:
	wails build -platform darwin/universal

build-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
	CC=x86_64-w64-mingw32-gcc \
	wails build -platform windows/amd64

setup-cross-compile:
	@which brew >/dev/null || (echo "Homebrew belum terinstall — install dari https://brew.sh"; exit 1)
	brew install mingw-w64

test:
	go test ./... -v

test-coverage:
	go test ./... -v -coverprofile=coverage.txt -covermode=atomic
	go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report: coverage.html"

lint:
	@which golangci-lint >/dev/null || (echo "Install: brew install golangci-lint atau go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1)
	golangci-lint run ./...

clean:
	rm -rf build/bin dist frontend/dist coverage.txt coverage.html

# Mock helpers untuk dev di Mac
mock-card-read:
	@curl -sX POST http://localhost:9090/mock/card-read \
		-H "Content-Type: application/json" \
		-d '{"nik":"$(NIK)","nama":"$(NAMA)","tgl_lahir":"1980-05-15","alamat":"Jl. Merdeka No. 1, Jakarta","no_kartu":"0001234567890012"}'
	@echo ""

mock-fp-fail:
	@curl -sX POST http://localhost:9090/mock/fp-fail
	@echo ""
