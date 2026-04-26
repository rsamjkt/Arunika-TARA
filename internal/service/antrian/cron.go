package antrian

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
)

// DailyResetSchedule adalah cron expression default untuk reset
// harian: 00:01 setiap hari (memberi 1 menit buffer setelah midnight
// untuk memastikan waktu sudah cleanly di hari baru).
const DailyResetSchedule = "1 0 * * *"

// StartDailyReset memulai cron job yang memanggil svc.ResetAll() pada
// jadwal harian (default DailyResetSchedule, timezone Asia/Jakarta).
//
// Mengembalikan *cron.Cron supaya caller bisa Stop() saat shutdown
// (graceful exit). Caller bertanggung jawab atas lifecycle.
//
//	c, err := antrian.StartDailyReset(svc, "")
//	defer c.Stop()
//
// Schedule bisa di-override (mis. test pakai "@every 1s"). Empty
// string → pakai DailyResetSchedule.
func StartDailyReset(svc *AntrianService, schedule string) (*cron.Cron, error) {
	if svc == nil {
		return nil, fmt.Errorf("antrian: svc nil")
	}
	if schedule == "" {
		schedule = DailyResetSchedule
	}

	wib := wibLoc()
	c := cron.New(cron.WithLocation(wib))

	logger := svc.logger
	if logger == nil {
		logger = slog.Default()
	}

	_, err := c.AddFunc(schedule, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		started := time.Now()
		if err := svc.ResetAll(ctx); err != nil {
			logger.Error("antrian-cron: daily reset gagal",
				"err", err, "schedule", schedule)
			return
		}
		logger.Info("antrian-cron: daily reset sukses",
			"schedule", schedule, "duration", time.Since(started))
	})
	if err != nil {
		return nil, fmt.Errorf("antrian-cron: AddFunc(%q): %w", schedule, err)
	}

	c.Start()
	logger.Info("antrian-cron: started",
		"schedule", schedule, "tz", "Asia/Jakarta")
	return c, nil
}

// wibLoc — Asia/Jakarta dengan fallback FixedZone +07:00 jika tzdata
// tidak tersedia (Windows tanpa Go tzdata embed).
func wibLoc() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*3600)
	}
	return loc
}
