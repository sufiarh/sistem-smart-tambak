package core

import (
	"log"
	"time"
)

// StartController adalah Goroutine 2 — logika trigger semua aktuator
func StartController(cfg *Config) {
	log.Println("[CONTROLLER] Goroutine controller dimulai")

	interval := time.Duration(cfg.Sensors.ReadInterval) * time.Second

	// Tracking status terakhir untuk hysteresis
	var lastAerator bool
	var lastPumpOut bool

	// Tracking feeder untuk hindari double trigger
	lastFeederTime := make(map[string]time.Time)

	for {
		// Tunggu data sensor tersedia
		reading, ready := GetCurrentReading()
		if !ready {
			time.Sleep(1 * time.Second)
			continue
		}

		// ============================================================
		// AERATOR #1 & #2 — Trigger DO
		// ============================================================
		aeratorMode1 := GetControlMode("aerator1")
		aeratorMode2 := GetControlMode("aerator2")

		if aeratorMode1 == "AUTO" && aeratorMode2 == "AUTO" {
			if reading.DO < cfg.Threshold.DOAeratorOn && !lastAerator {
				log.Printf("[CONTROLLER] DO rendah (%.2f) → Aerator ON", reading.DO)
				SetRelay(cfg, cfg.Relay.Aerator1, true)
				SetRelay(cfg, cfg.Relay.Aerator2, true)
				lastAerator = true

			} else if reading.DO > cfg.Threshold.DOAeratorOff && lastAerator {
				log.Printf("[CONTROLLER] DO normal (%.2f) → Aerator OFF", reading.DO)
				SetRelay(cfg, cfg.Relay.Aerator1, false)
				SetRelay(cfg, cfg.Relay.Aerator2, false)
				lastAerator = false
			}
		}

		// ============================================================
		// POMPA INLET — Trigger pH + Salinitas + Water Level
		// ============================================================
		if GetControlMode("pump_in") == "AUTO" {
			pumpInOn := false
			reason := ""

			if reading.PH < cfg.Threshold.PHMin {
				pumpInOn = true
				reason = "pH terlalu asam"
			} else if reading.PH > cfg.Threshold.PHMax {
				pumpInOn = true
				reason = "pH terlalu basa"
			} else if reading.Salinity > cfg.Threshold.SalinityMax {
				pumpInOn = true
				reason = "salinitas terlalu tinggi"
			} else if reading.WaterLevel < cfg.Threshold.WaterLevelMin {
				pumpInOn = true
				reason = "water level rendah"
			}

			currentPumpIn := RelayStatus.PumpIn
			if pumpInOn && !currentPumpIn {
				log.Printf("[CONTROLLER] Pompa Inlet ON — %s", reason)
				SetRelay(cfg, cfg.Relay.PumpIn, true)
			} else if !pumpInOn && currentPumpIn {
				log.Println("[CONTROLLER] Pompa Inlet OFF — kondisi normal")
				SetRelay(cfg, cfg.Relay.PumpIn, false)
			}
		}

		// ============================================================
		// POMPA OUTLET — Trigger Turbiditas
		// ============================================================
		if GetControlMode("pump_out") == "AUTO" {
			if reading.Turbidity > cfg.Threshold.TurbidityOn && !lastPumpOut {
				log.Printf("[CONTROLLER] Turbiditas tinggi (%.2f NTU) → Pompa Outlet ON", reading.Turbidity)
				SetRelay(cfg, cfg.Relay.PumpOut, true)
				lastPumpOut = true

			} else if reading.Turbidity < cfg.Threshold.TurbidityOff && lastPumpOut {
				log.Printf("[CONTROLLER] Turbiditas normal (%.2f NTU) → Pompa Outlet OFF", reading.Turbidity)
				SetRelay(cfg, cfg.Relay.PumpOut, false)
				lastPumpOut = false
			}
		}

		// ============================================================
		// AUTO FEEDER — Jadwal + Cek DO
		// ============================================================
		if GetControlMode("feeder") == "AUTO" {
			checkFeeder(cfg, reading, lastFeederTime)
		}

		// ============================================================
		// LAMPU — Trigger Sensor Cahaya
		// ============================================================
		if GetControlMode("lamp") == "AUTO" {
			currentLamp := RelayStatus.Lamp
			if reading.Light < cfg.Threshold.LightThreshold && !currentLamp {
				log.Printf("[CONTROLLER] Cahaya rendah (%.2f lux) → Lampu ON", reading.Light)
				SetRelay(cfg, cfg.Relay.Lamp, true)

			} else if reading.Light >= cfg.Threshold.LightThreshold && currentLamp {
				log.Printf("[CONTROLLER] Cahaya cukup (%.2f lux) → Lampu OFF", reading.Light)
				SetRelay(cfg, cfg.Relay.Lamp, false)
			}
		}

		// Simpan status relay ke database
		SaveRelayState(RelayStatus)

		time.Sleep(interval)
	}
}

// checkFeeder menangani logika jadwal + trigger DO feeder
func checkFeeder(cfg *Config, reading SensorReading, lastFed map[string]time.Time) {
	now := time.Now()
	currentTime := now.Format("15:04")

	for _, schedule := range cfg.Feeder.Schedules {
		// Cek apakah waktu sekarang sesuai jadwal (toleransi 1 menit)
		if currentTime != schedule {
			continue
		}

		// Cek apakah sudah diberi pakan di jadwal ini
		if last, ok := lastFed[schedule]; ok {
			if now.Sub(last) < 50*time.Minute {
				continue // Sudah diberi pakan di jadwal ini
			}
		}

		// Cek DO sebelum beri pakan
		if reading.DO < cfg.Threshold.DOFeederMin {
			log.Printf("[CONTROLLER] Jadwal pakan %s — DO rendah (%.2f) → TUNDA %d menit",
				   schedule, reading.DO, cfg.Feeder.RetryDelayMinutes)
			return
		}

		// DO cukup — aktifkan feeder
		log.Printf("[CONTROLLER] Jadwal pakan %s — DO normal (%.2f) → Feeder ON",
			   schedule, reading.DO)

		SetRelay(cfg, cfg.Relay.Feeder, true)
		time.Sleep(time.Duration(cfg.Feeder.DurationSeconds) * time.Second)
		SetRelay(cfg, cfg.Relay.Feeder, false)

		log.Printf("[CONTROLLER] Feeder selesai (%d detik)", cfg.Feeder.DurationSeconds)
		lastFed[schedule] = now
	}
}
