package core

import (
	"fmt"
	"log"

	"github.com/karalabe/hid"
)

// RelayStatus menyimpan status ON/OFF semua relay saat ini
var RelayStatus = RelayState{}

// SetRelay menyalakan atau mematikan relay berdasarkan channel
func SetRelay(cfg *Config, channel int, on bool) error {
	if cfg.App.Simulation {
		return setRelaySimulated(channel, on)
	}
	return setRelayHID(cfg, channel, on)
}

// setRelaySimulated — mode laptop, print ke terminal saja
func setRelaySimulated(channel int, on bool) error {
	status := "OFF"
	if on {
		status = "ON"
	}

	name := channelName(channel)
	log.Printf("[RELAY] %-20s CH%d → %s", name, channel, status)

	// Update status di memory
	updateRelayStatus(channel, on)
	return nil
}

// setRelayHID — mode production, kirim perintah ke USB Relay via HID
func setRelayHID(cfg *Config, channel int, on bool) error {
	// Buka koneksi ke USB Relay Module
	devices := hid.Enumerate(
		uint16(cfg.Hardware.RelayVendorID),
				 uint16(cfg.Hardware.RelayProductID),
	)

	if len(devices) == 0 {
		return fmt.Errorf("USB relay tidak ditemukan (VID:%X PID:%X)",
				  cfg.Hardware.RelayVendorID,
		    cfg.Hardware.RelayProductID,
		)
	}

	device, err := devices[0].Open()
	if err != nil {
		return fmt.Errorf("gagal buka USB relay: %v", err)
	}
	defer device.Close()

	// Perintah HID untuk relay module 8 channel
	// Format: [0x00, command, channel, 0x00, ...]
	var command byte = 0xFE // OFF
	if on {
		command = 0xFF // ON
	}

	buf := make([]byte, 9)
	buf[0] = 0x00
	buf[1] = command
	buf[2] = byte(channel)

	_, err = device.Write(buf)
	if err != nil {
		return fmt.Errorf("gagal kirim perintah relay: %v", err)
	}

	updateRelayStatus(channel, on)
	name := channelName(channel)
	log.Printf("[RELAY] %-20s CH%d → %v", name, channel, on)

	return nil
}

// updateRelayStatus update status relay di memory berdasarkan channel
func updateRelayStatus(channel int, on bool) {
	switch channel {
		case 1:
			RelayStatus.Aerator1 = on
		case 2:
			RelayStatus.Aerator2 = on
		case 3:
			RelayStatus.PumpIn = on
		case 4:
			RelayStatus.PumpOut = on
		case 5:
			RelayStatus.Feeder = on
		case 6:
			RelayStatus.Lamp = on
	}
}

// channelName mengembalikan nama aktuator berdasarkan channel
func channelName(channel int) string {
	names := map[int]string{
		1: "Aerator #1",
		2: "Aerator #2",
		3: "Pompa Inlet",
		4: "Pompa Outlet",
		5: "Auto Feeder",
		6: "Lampu",
	}
	if name, ok := names[channel]; ok {
		return name
	}
	return fmt.Sprintf("Channel %d", channel)
}

// SetAllRelayOff mematikan semua relay (dipakai saat startup)
func SetAllRelayOff(cfg *Config) {
	log.Println("[RELAY] Matikan semua relay...")
	for ch := 1; ch <= 6; ch++ {
		if err := SetRelay(cfg, ch, false); err != nil {
			log.Printf("[RELAY] Gagal matikan CH%d: %v", ch, err)
		}
	}
}
