package main

import (
	"log"
	"os"
	"smart-tambak/core"
)

func main() {
	log.Println("============================================================")
	log.Println("  SMART TAMBAK — Udang Vaname")
	log.Println("  Tambakmulyo, Kebumen, Jawa Tengah")
	log.Println("============================================================")

	// ============================================================
	// 1. Load konfigurasi dari config.yaml
	// ============================================================
	cfg, err := core.LoadConfig("config.yaml")
	if err != nil {
		log.Fatal("[MAIN] Gagal load config:", err)
	}

	if cfg.App.Simulation {
		log.Println("[MAIN] Mode: SIMULASI (laptop)")
	} else {
		log.Println("[MAIN] Mode: PRODUCTION (HG680P)")
	}

	// ============================================================
	// 2. Buat folder data jika belum ada
	// ============================================================
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatal("[MAIN] Gagal buat folder data:", err)
	}

	// ============================================================
	// 3. Inisialisasi database SQLite
	// ============================================================
	if err := core.InitDatabase(cfg.Database.Path); err != nil {
		log.Fatal("[MAIN] Gagal inisialisasi database:", err)
	}

	// ============================================================
	// 4. Matikan semua relay saat startup (safety)
	// ============================================================
	core.SetAllRelayOff(cfg)

	// ============================================================
	// 5. Jalankan 3 goroutine secara paralel
	// ============================================================

	// Goroutine 1 — Baca sensor setiap 5 detik
	go core.StartSensorReader(cfg)
	log.Println("[MAIN] Goroutine 1 (Sensor Reader) dimulai")

	// Goroutine 2 — Logika trigger kontrol aktuator
	go core.StartController(cfg)
	log.Println("[MAIN] Goroutine 2 (Controller) dimulai")

	// Goroutine 3 — Web server dashboard (blocking — harus terakhir)
	log.Println("[MAIN] Goroutine 3 (Web Server) dimulai")
	core.StartServer(cfg)
}
