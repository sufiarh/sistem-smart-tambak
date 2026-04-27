package core

import (
	"log"
	"math/rand"
	"time"

	"github.com/goburrow/modbus"
	"encoding/binary"
	"math"
)

// SensorReading adalah data mentah hasil baca sensor satu siklus
type SensorReading struct {
	PH          float64
	DO          float64
	Temperature float64
	Salinity    float64
	Turbidity   float64
	WaterLevel  float64
	Light       float64
	ReadAt      time.Time
}

// currentReading menyimpan data sensor terbaru (dibaca goroutine lain)
var currentReading SensorReading
var readingReady bool

// GetCurrentReading mengembalikan data sensor terbaru
func GetCurrentReading() (SensorReading, bool) {
	return currentReading, readingReady
}

// StartSensorReader adalah Goroutine 1 — baca sensor terus-menerus
func StartSensorReader(cfg *Config) {
	log.Println("[SENSOR] Goroutine sensor reader dimulai")

	interval := time.Duration(cfg.Sensors.ReadInterval) * time.Second

	for {
		var reading SensorReading
		var err error

		if cfg.App.Simulation {
			// Mode simulasi — data random realistis
			reading = readSimulated(cfg)
		} else {
			// Mode production — baca RS485 nyata
			reading, err = readRS485(cfg)
			if err != nil {
				log.Println("[SENSOR] Error baca RS485:", err)
				time.Sleep(interval)
				continue
			}
		}

		reading.ReadAt = time.Now()

		// Simpan ke memory (dibaca controller)
		currentReading = reading
		readingReady = true

		// Simpan ke SQLite
		err = SaveSensorData(SensorData{
			PH:          reading.PH,
			DO:          reading.DO,
			Temperature: reading.Temperature,
			Salinity:    reading.Salinity,
			Turbidity:   reading.Turbidity,
			WaterLevel:  reading.WaterLevel,
			Light:       reading.Light,
		})
		if err != nil {
			log.Println("[SENSOR] Gagal simpan ke database:", err)
		}

		log.Printf("[SENSOR] pH:%.2f DO:%.2f Suhu:%.2f Sal:%.2f Turb:%.2f Level:%.2f Cahaya:%.2f",
			   reading.PH, reading.DO, reading.Temperature,
	     reading.Salinity, reading.Turbidity,
	     reading.WaterLevel, reading.Light,
		)

		time.Sleep(interval)
	}
}

// readRS485 membaca semua sensor via Modbus RTU (production mode)
func readRS485(cfg *Config) (SensorReading, error) {
	var r SensorReading

	handler := modbus.NewRTUClientHandler(cfg.Hardware.RS485Port)
	handler.BaudRate = cfg.Hardware.RS485Baudrate
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 1
	handler.Timeout = time.Duration(cfg.Hardware.RS485Timeout) * time.Second

	if err := handler.Connect(); err != nil {
		return r, err
	}
	defer handler.Close()

	client := modbus.NewClient(handler)

	// Baca tiap sensor berdasarkan alamat Modbus
	r.PH, _ = readModbusFloat(client, byte(cfg.Sensors.PH.Address))
	r.DO, _ = readModbusFloat(client, byte(cfg.Sensors.DO.Address))
	r.Temperature, _ = readModbusFloat(client, byte(cfg.Sensors.Temperature.Address))
	r.Salinity, _ = readModbusFloat(client, byte(cfg.Sensors.Salinity.Address))
	r.Turbidity, _ = readModbusFloat(client, byte(cfg.Sensors.Turbidity.Address))
	r.WaterLevel, _ = readModbusFloat(client, byte(cfg.Sensors.WaterLevel.Address))
	r.Light, _ = readModbusFloat(client, byte(cfg.Sensors.Light.Address))

	return r, nil
}

// readModbusFloat membaca 2 register (32-bit float) dari sensor RS485
func readModbusFloat(client modbus.Client, address byte) (float64, error) {
	// Kebanyakan sensor RS485 menyimpan float32 di 2 register (4 byte)
	results, err := client.ReadHoldingRegisters(0, 2)
	if err != nil {
		return 0, err
	}
	// Konversi 4 byte → float32 → float64
	bits := binary.BigEndian.Uint32(results)
	float := math.Float32frombits(bits)
	return float64(float), nil
}

// readSimulated menghasilkan data sensor random yang realistis (laptop mode)
func readSimulated(cfg *Config) SensorReading {
	sim := cfg.Simulation

	// Simulasi perubahan DO yang dramatis setiap malam
	// agar logika trigger aerator bisa diuji
	hour := time.Now().Hour()
	doBase := randBetween(sim.DORange[0], sim.DORange[1])
	if hour >= 0 && hour <= 5 {
		// Dini hari — DO sengaja dibuat lebih rendah
		doBase = randBetween(sim.DORange[0], 4.5)
	}

	return SensorReading{
		PH:          randBetween(sim.PHRange[0], sim.PHRange[1]),
		DO:          doBase,
		Temperature: randBetween(sim.TemperatureRange[0], sim.TemperatureRange[1]),
		Salinity:    randBetween(sim.SalinityRange[0], sim.SalinityRange[1]),
		Turbidity:   randBetween(sim.TurbidityRange[0], sim.TurbidityRange[1]),
		WaterLevel:  randBetween(sim.WaterLevelRange[0], sim.WaterLevelRange[1]),
		Light:       randBetween(sim.LightRange[0], sim.LightRange[1]),
	}
}

// randBetween menghasilkan float64 random antara min dan max
func randBetween(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}
