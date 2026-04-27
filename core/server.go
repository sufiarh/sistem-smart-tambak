package core

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

// Password dashboard
const dashboardPassword = "petambak"

// ============================================================
// AUTH MIDDLEWARE
// ============================================================

// authHandler adalah global middleware untuk semua request
func authHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Halaman login & logout bebas diakses tanpa auth
		if r.URL.Path == "/login" || r.URL.Path == "/login/submit" || r.URL.Path == "/logout" {
			next.ServeHTTP(w, r)
			return
		}

		// CSS & JS boleh diakses tanpa auth (untuk styling halaman login)
		if r.URL.Path == "/style.css" || r.URL.Path == "/app.js" {
			next.ServeHTTP(w, r)
			return
		}

		// Cek cookie session
		cookie, err := r.Cookie("session")
		if err != nil || cookie.Value != dashboardPassword {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ============================================================
// LOGIN HANDLERS
// ============================================================

// handlerLogin menampilkan halaman login
func handlerLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<!DOCTYPE html>
	<html lang="id">
	<head>
	<meta charset="UTF-8"/>
	<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
	<title>Smart Tambak — Login</title>
	<link rel="stylesheet" href="/style.css"/>
	<style>
	.login-wrapper {
	min-height: 100vh;
	display: flex;
	align-items: center;
	justify-content: center;
	background: var(--bg);
}
.login-card {
background: white;
border-radius: 16px;
padding: 40px 32px;
box-shadow: 0 4px 24px rgba(0,0,0,0.12);
	width: 100%;
	max-width: 360px;
	text-align: center;
}
.login-logo  { font-size: 3rem; margin-bottom: 8px; }
.login-title { font-size: 1.3rem; font-weight: 700; color: var(--primary); margin-bottom: 4px; }
.login-sub   { font-size: 0.82rem; color: var(--text-light); margin-bottom: 28px; }
.login-input {
width: 100%;
padding: 12px 16px;
border: 2px solid var(--border);
	border-radius: 8px;
	font-size: 1rem;
	margin-bottom: 16px;
	outline: none;
	transition: border 0.2s;
}
.login-input:focus { border-color: var(--primary); }
.login-btn {
width: 100%;
padding: 12px;
background: var(--primary);
	color: white;
	border: none;
	border-radius: 8px;
	font-size: 1rem;
	font-weight: 600;
	cursor: pointer;
	transition: opacity 0.2s;
}
.login-btn:hover { opacity: 0.88; }
.login-error {
background: #fadbd8;
color: #922b21;
border-radius: 8px;
padding: 10px;
font-size: 0.85rem;
margin-bottom: 16px;
}
</style>
</head>
<body>
<div class="login-wrapper">
<div class="login-card">
<div class="login-logo">🦐</div>
<div class="login-title">Smart Tambak</div>
<div class="login-sub">R&D Sufi Anugrah dan Abu Yazid Bustomi</div>
` + loginError(r) + `
<form method="POST" action="/login/submit">
<input class="login-input" type="password"
name="password" placeholder="Masukkan password..."
autofocus required/>
<button class="login-btn" type="submit">🔐 Masuk</button>
</form>
</div>
</div>
</body>
</html>`))
}

// loginError menampilkan pesan error jika login gagal
func loginError(r *http.Request) string {
	if r.URL.Query().Get("error") == "1" {
		return `<div class="login-error">❌ Password salah, coba lagi.</div>`
	}
	return ""
}

// handlerLoginSubmit memproses form login
func handlerLoginSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	password := r.FormValue("password")

	if password != dashboardPassword {
		http.Redirect(w, r, "/login?error=1", http.StatusFound)
		return
	}

	// Set cookie session selama 7 hari
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    dashboardPassword,
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

// handlerLogout menghapus session dan redirect ke login
func handlerLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/login", http.StatusFound)
}

// ============================================================
// START SERVER
// ============================================================

// StartServer adalah Goroutine 3 — web server dashboard
func StartServer(cfg *Config) {
	mux := http.NewServeMux()

	// ============================================================
	// STATIC FILES
	// ============================================================
	mux.HandleFunc("/", handlerIndex)
	mux.HandleFunc("/style.css", handlerCSS)
	mux.HandleFunc("/app.js", handlerJS)

	// ============================================================
	// LOGIN & LOGOUT
	// ============================================================
	mux.HandleFunc("/login", handlerLogin)
	mux.HandleFunc("/login/submit", handlerLoginSubmit)
	mux.HandleFunc("/logout", handlerLogout)

	// ============================================================
	// REST API — Data Sensor
	// ============================================================

	// GET /api/sensor — data sensor terbaru
	mux.HandleFunc("/api/sensor", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		data, err := GetLatestSensor()
		if err != nil {
			jsonError(w, "Gagal ambil data sensor", 500)
			return
		}
		jsonOK(w, map[string]interface{}{
			"timestamp":   data.Timestamp.Format("2006-01-02 15:04:05"),
		       "ph":          data.PH,
	 "do":          data.DO,
	 "temperature": data.Temperature,
	 "salinity":    data.Salinity,
	 "turbidity":   data.Turbidity,
	 "water_level": data.WaterLevel,
	 "light":       data.Light,
		})
	})

	// GET /api/history?hours=24 — data historis sensor
	mux.HandleFunc("/api/history", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		hoursStr := r.URL.Query().Get("hours")
		hours := 24
		if hoursStr != "" {
			if h, err := strconv.Atoi(hoursStr); err == nil {
				hours = h
			}
		}

		history, err := GetSensorHistory(hours)
		if err != nil {
			jsonError(w, "Gagal ambil histori", 500)
			return
		}

		var result []map[string]interface{}
		for _, d := range history {
			result = append(result, map[string]interface{}{
				"timestamp":   d.Timestamp.Format("15:04:05"),
					"ph":          d.PH,
					"do":          d.DO,
					"temperature": d.Temperature,
					"salinity":    d.Salinity,
					"turbidity":   d.Turbidity,
					"water_level": d.WaterLevel,
					"light":       d.Light,
			})
		}
		jsonOK(w, result)
	})

	// ============================================================
	// REST API — Status Relay
	// ============================================================

	// GET /api/relay — status semua relay
	mux.HandleFunc("/api/relay", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		jsonOK(w, map[string]interface{}{
			"aerator1": RelayStatus.Aerator1,
	 "aerator2": RelayStatus.Aerator2,
	 "pump_in":  RelayStatus.PumpIn,
	 "pump_out": RelayStatus.PumpOut,
	 "feeder":   RelayStatus.Feeder,
	 "lamp":     RelayStatus.Lamp,
		})
	})

	// ============================================================
	// REST API — Kontrol Manual Relay
	// ============================================================

	// POST /api/control — kontrol manual relay
	mux.HandleFunc("/api/control", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method != http.MethodPost {
			jsonError(w, "Method tidak diizinkan", 405)
			return
		}

		var body struct {
			Actuator string `json:"actuator"`
			State    bool   `json:"state"`
			Mode     string `json:"mode"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonError(w, "Body tidak valid", 400)
			return
		}

		// Set mode AUTO/MANUAL
		if body.Mode != "" {
			if err := SetControlMode(body.Actuator, body.Mode); err != nil {
				jsonError(w, "Gagal set mode", 500)
				return
			}
		}

		// Kontrol relay jika mode MANUAL
		if body.Mode == "MANUAL" {
			channel := actuatorToChannel(cfg, body.Actuator)
			if channel == 0 {
				jsonError(w, "Aktuator tidak dikenal", 400)
				return
			}
			if err := SetRelay(cfg, channel, body.State); err != nil {
				jsonError(w, "Gagal kontrol relay", 500)
				return
			}
		}

		jsonOK(w, map[string]string{
			"status":   "ok",
	 "actuator": body.Actuator,
	 "mode":     body.Mode,
		})
	})

	// ============================================================
	// REST API — Mode Kontrol
	// ============================================================

	// GET /api/mode — ambil mode semua aktuator
	mux.HandleFunc("/api/mode", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		actuators := []string{
			"aerator1", "aerator2",
			"pump_in", "pump_out",
			"feeder", "lamp",
		}
		modes := make(map[string]string)
		for _, a := range actuators {
			modes[a] = GetControlMode(a)
		}
		jsonOK(w, modes)
	})

	// ============================================================
	// REST API — Status Sistem
	// ============================================================

	// GET /api/status — status keseluruhan sistem
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		reading, ready := GetCurrentReading()
		if !ready {
			jsonOK(w, map[string]string{"status": "loading"})
			return
		}

		status := "normal"
		if reading.DO < 4.0 ||
			reading.PH < 7.5 ||
			reading.PH > 8.5 ||
			reading.WaterLevel < 1.0 {
				status = "warning"
			}
			if reading.DO < 2.0 {
				status = "danger"
			}

			jsonOK(w, map[string]interface{}{
				"status":  status,
	  "message": statusMessage(status),
			})
	})

	// Jalankan server dengan auth middleware global
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("[SERVER] Dashboard berjalan di http://%s", addr)
	if err := http.ListenAndServe(addr, authHandler(mux)); err != nil {
		log.Fatal("[SERVER] Gagal jalankan server:", err)
	}
}

// ============================================================
// STATIC FILE HANDLERS
// ============================================================

func handlerIndex(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, "index.html tidak ditemukan", 404)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(data)
}

func handlerCSS(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("web/style.css")
	if err != nil {
		http.Error(w, "style.css tidak ditemukan", 404)
		return
	}
	w.Header().Set("Content-Type", "text/css")
	w.Write(data)
}

func handlerJS(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("web/app.js")
	if err != nil {
		http.Error(w, "app.js tidak ditemukan", 404)
		return
	}
	w.Header().Set("Content-Type", "application/javascript")
	w.Write(data)
}

// ============================================================
// HELPER FUNCTIONS
// ============================================================

func jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func actuatorToChannel(cfg *Config, actuator string) int {
	switch actuator {
		case "aerator1":
			return cfg.Relay.Aerator1
		case "aerator2":
			return cfg.Relay.Aerator2
		case "pump_in":
			return cfg.Relay.PumpIn
		case "pump_out":
			return cfg.Relay.PumpOut
		case "feeder":
			return cfg.Relay.Feeder
		case "lamp":
			return cfg.Relay.Lamp
	}
	return 0
}

func statusMessage(status string) string {
	switch status {
		case "normal":
			return "Semua parameter dalam kondisi normal"
		case "warning":
			return "Ada parameter yang perlu diperhatikan"
		case "danger":
			return "KONDISI KRITIS — segera periksa tambak!"
	}
	return "Memuat data..."
}
