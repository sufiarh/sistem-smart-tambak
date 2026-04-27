package core

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v3"
)

// ============================================================
// CONFIG — Struct untuk baca config.yaml
// ============================================================

type Config struct {
	App struct {
		Name       string `yaml:"name"`
		Version    string `yaml:"version"`
		Simulation bool   `yaml:"simulation"`
	} `yaml:"app"`

	Hardware struct {
		RS485Port      string `yaml:"rs485_port"`
		RS485Baudrate  int    `yaml:"rs485_baudrate"`
		RS485Timeout   int    `yaml:"rs485_timeout"`
		RelayVendorID  int    `yaml:"relay_vendor_id"`
		RelayProductID int    `yaml:"relay_product_id"`
	} `yaml:"hardware"`

	Sensors struct {
		ReadInterval int `yaml:"read_interval"`
		PH           struct {
			Address int    `yaml:"address"`
			Name    string `yaml:"name"`
			Unit    string `yaml:"unit"`
		} `yaml:"ph"`
		DO struct {
			Address int    `yaml:"address"`
			Name    string `yaml:"name"`
			Unit    string `yaml:"unit"`
		} `yaml:"do"`
		Temperature struct {
			Address int    `yaml:"address"`
			Name    string `yaml:"name"`
			Unit    string `yaml:"unit"`
		} `yaml:"temperature"`
		Salinity struct {
			Address int    `yaml:"address"`
			Name    string `yaml:"name"`
			Unit    string `yaml:"unit"`
		} `yaml:"salinity"`
		Turbidity struct {
			Address int    `yaml:"address"`
			Name    string `yaml:"name"`
			Unit    string `yaml:"unit"`
		} `yaml:"turbidity"`
		WaterLevel struct {
			Address int    `yaml:"address"`
			Name    string `yaml:"name"`
			Unit    string `yaml:"unit"`
		} `yaml:"water_level"`
		Light struct {
			Address int    `yaml:"address"`
			Name    string `yaml:"name"`
			Unit    string `yaml:"unit"`
		} `yaml:"light"`
	} `yaml:"sensors"`

	Threshold struct {
		DOAeratorOn    float64 `yaml:"do_aerator_on"`
		DOAeratorOff   float64 `yaml:"do_aerator_off"`
		PHMin          float64 `yaml:"ph_min"`
		PHMax          float64 `yaml:"ph_max"`
		SalinityMax    float64 `yaml:"salinity_max"`
		WaterLevelMin  float64 `yaml:"water_level_min"`
		TurbidityOn    float64 `yaml:"turbidity_on"`
		TurbidityOff   float64 `yaml:"turbidity_off"`
		DOFeederMin    float64 `yaml:"do_feeder_min"`
		LightThreshold float64 `yaml:"light_threshold"`
	} `yaml:"threshold"`

	Feeder struct {
		DurationSeconds    int      `yaml:"duration_seconds"`
		RetryDelayMinutes  int      `yaml:"retry_delay_minutes"`
		Schedules          []string `yaml:"schedules"`
	} `yaml:"feeder"`

	Relay struct {
		Aerator1 int `yaml:"aerator_1"`
		Aerator2 int `yaml:"aerator_2"`
		PumpIn   int `yaml:"pump_in"`
		PumpOut  int `yaml:"pump_out"`
		Feeder   int `yaml:"feeder"`
		Lamp     int `yaml:"lamp"`
	} `yaml:"relay"`

	Server struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`

	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`

	Simulation struct {
		UpdateInterval   int       `yaml:"update_interval"`
		PHRange          []float64 `yaml:"ph_range"`
		DORange          []float64 `yaml:"do_range"`
		TemperatureRange []float64 `yaml:"temperature_range"`
		SalinityRange    []float64 `yaml:"salinity_range"`
		TurbidityRange   []float64 `yaml:"turbidity_range"`
		WaterLevelRange  []float64 `yaml:"water_level_range"`
		LightRange       []float64 `yaml:"light_range"`
	} `yaml:"simulation"`
}

// LoadConfig membaca config.yaml dan mengembalikan struct Config
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// DB adalah koneksi global SQLite
var DB *sql.DB

// SensorData adalah struktur data satu baris pembacaan sensor
type SensorData struct {
	ID          int64
	Timestamp   time.Time
	PH          float64
	DO          float64
	Temperature float64
	Salinity    float64
	Turbidity   float64
	WaterLevel  float64
	Light       float64
}

// RelayState adalah struktur status relay/aktuator
type RelayState struct {
	ID          int64
	Timestamp   time.Time
	Aerator1    bool
	Aerator2    bool
	PumpIn      bool
	PumpOut     bool
	Feeder      bool
	Lamp        bool
}

// InitDatabase membuka koneksi SQLite dan membuat tabel jika belum ada
func InitDatabase(path string) error {
	var err error

	// Buka atau buat file database
	DB, err = sql.Open("sqlite3", path)
	if err != nil {
		return err
	}

	// Cek koneksi
	if err = DB.Ping(); err != nil {
		return err
	}

	// Buat tabel jika belum ada
	if err = migrate(); err != nil {
		return err
	}

	log.Println("[DATABASE] SQLite terhubung:", path)
	return nil
}

// migrate membuat semua tabel yang dibutuhkan
func migrate() error {
	queries := []string{
		// Tabel data sensor
		`CREATE TABLE IF NOT EXISTS sensor_data (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp   DATETIME DEFAULT CURRENT_TIMESTAMP,
			ph          REAL NOT NULL,
			do_level    REAL NOT NULL,
			temperature REAL NOT NULL,
			salinity    REAL NOT NULL,
			turbidity   REAL NOT NULL,
			water_level REAL NOT NULL,
			light       REAL NOT NULL
		)`,

		// Tabel status relay
		`CREATE TABLE IF NOT EXISTS relay_state (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp   DATETIME DEFAULT CURRENT_TIMESTAMP,
			aerator1    INTEGER NOT NULL DEFAULT 0,
			aerator2    INTEGER NOT NULL DEFAULT 0,
			pump_in     INTEGER NOT NULL DEFAULT 0,
			pump_out    INTEGER NOT NULL DEFAULT 0,
			feeder      INTEGER NOT NULL DEFAULT 0,
			lamp        INTEGER NOT NULL DEFAULT 0
		)`,

		// Tabel mode kontrol (AUTO atau MANUAL per aktuator)
		`CREATE TABLE IF NOT EXISTS control_mode (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			actuator    TEXT UNIQUE NOT NULL,
			mode        TEXT NOT NULL DEFAULT 'AUTO',
			updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Isi default control_mode jika kosong
		`INSERT OR IGNORE INTO control_mode (actuator, mode) VALUES
			('aerator1', 'AUTO'),
			('aerator2', 'AUTO'),
			('pump_in',  'AUTO'),
			('pump_out', 'AUTO'),
			('feeder',   'AUTO'),
			('lamp',     'AUTO')`,

		// Index untuk query historis lebih cepat
		`CREATE INDEX IF NOT EXISTS idx_sensor_timestamp
			ON sensor_data(timestamp DESC)`,
	}

	for _, q := range queries {
		if _, err := DB.Exec(q); err != nil {
			return err
		}
	}

	log.Println("[DATABASE] Migrasi tabel selesai")
	return nil
}

// SaveSensorData menyimpan satu baris data sensor ke SQLite
func SaveSensorData(d SensorData) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := DB.Exec(`
	INSERT INTO sensor_data
	(timestamp, ph, do_level, temperature, salinity, turbidity, water_level, light)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			  now,
		   d.PH, d.DO, d.Temperature, d.Salinity,
		   d.Turbidity, d.WaterLevel, d.Light,
	)
	return err
}

// SaveRelayState menyimpan status relay saat ini ke SQLite
func SaveRelayState(r RelayState) error {
	_, err := DB.Exec(`
		INSERT INTO relay_state
			(aerator1, aerator2, pump_in, pump_out, feeder, lamp)
		VALUES (?, ?, ?, ?, ?, ?)`,
		boolToInt(r.Aerator1), boolToInt(r.Aerator2),
		boolToInt(r.PumpIn), boolToInt(r.PumpOut),
		boolToInt(r.Feeder), boolToInt(r.Lamp),
	)
	return err
}

// GetLatestSensor mengambil data sensor terbaru dari SQLite
func GetLatestSensor() (SensorData, error) {
	var d SensorData
	var ts string

	err := DB.QueryRow(`
		SELECT id, timestamp, ph, do_level, temperature,
		       salinity, turbidity, water_level, light
		FROM sensor_data
		ORDER BY timestamp DESC
		LIMIT 1`,
	).Scan(
		&d.ID, &ts, &d.PH, &d.DO, &d.Temperature,
		&d.Salinity, &d.Turbidity, &d.WaterLevel, &d.Light,
	)
	if err != nil {
		return d, err
	}

	d.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
	return d, nil
}

// GetSensorHistory mengambil data historis N jam terakhir
func GetSensorHistory(hours int) ([]SensorData, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour).
	Format("2006-01-02 15:04:05")

	// Aggregate per jam — rata-rata nilai sensor tiap jam
	rows, err := DB.Query(`
	SELECT
	strftime('%Y-%m-%d %H:00:00', timestamp) as hour,
			      AVG(ph),
			      AVG(do_level),
			      AVG(temperature),
			      AVG(salinity),
			      AVG(turbidity),
			      AVG(water_level),
			      AVG(light)
	FROM sensor_data
	WHERE timestamp >= ?
	GROUP BY strftime('%Y-%m-%d %H', timestamp)
	ORDER BY hour ASC`,
	since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SensorData
	for rows.Next() {
		var d SensorData
		var ts string
		if err := rows.Scan(
			&ts,
		      &d.PH, &d.DO, &d.Temperature,
		      &d.Salinity, &d.Turbidity,
		      &d.WaterLevel, &d.Light,
		); err != nil {
			continue
		}

		// Parse timestamp
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05",
		}
		var parseErr error
		for _, f := range formats {
			d.Timestamp, parseErr = time.ParseInLocation(f, ts, time.Local)
			if parseErr == nil {
				break
			}
		}
		if parseErr != nil {
			log.Println("[DATABASE] Gagal parse timestamp:", ts)
		}

		result = append(result, d)
	}
	return result, nil
}

// GetLatestRelayState mengambil status relay terbaru
func GetLatestRelayState() (RelayState, error) {
	var r RelayState
	var ts string
	var a1, a2, pi, po, f, l int

	err := DB.QueryRow(`
		SELECT id, timestamp, aerator1, aerator2,
		       pump_in, pump_out, feeder, lamp
		FROM relay_state
		ORDER BY timestamp DESC
		LIMIT 1`,
	).Scan(&r.ID, &ts, &a1, &a2, &pi, &po, &f, &l)

	if err != nil {
		return r, err
	}

	r.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
	r.Aerator1 = intToBool(a1)
	r.Aerator2 = intToBool(a2)
	r.PumpIn = intToBool(pi)
	r.PumpOut = intToBool(po)
	r.Feeder = intToBool(f)
	r.Lamp = intToBool(l)

	return r, nil
}

// GetControlMode mengambil mode (AUTO/MANUAL) suatu aktuator
func GetControlMode(actuator string) string {
	var mode string
	err := DB.QueryRow(`
		SELECT mode FROM control_mode WHERE actuator = ?`, actuator,
	).Scan(&mode)
	if err != nil {
		return "AUTO"
	}
	return mode
}

// SetControlMode mengubah mode (AUTO/MANUAL) suatu aktuator
func SetControlMode(actuator, mode string) error {
	_, err := DB.Exec(`
		UPDATE control_mode
		SET mode = ?, updated_at = CURRENT_TIMESTAMP
		WHERE actuator = ?`, mode, actuator,
	)
	return err
}

// helper
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i == 1
}
