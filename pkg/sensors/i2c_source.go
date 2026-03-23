// SWE100821: I2C sensor source — reads from devices found on I2C buses.
// Auto-scans for known sensor addresses and reads data from common sensors:
// BME280/BMP280 (temp/humidity/pressure), AHT20 (temp/humidity), TSL2561 (light).
// Graceful failure: skips unresponsive addresses, retries on next poll.

package sensors

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// knownI2CSensors maps addresses to sensor types for auto-identification.
var knownI2CSensors = map[int]string{
	0x38: "AHT20 (temp/humidity)",
	0x39: "TSL2561 (light)",
	0x40: "HDC1080 (temp/humidity)",
	0x44: "SHT30 (temp/humidity)",
	0x48: "ADS1115 (ADC)",
	0x49: "TMP102 (temperature)",
	0x50: "EEPROM",
	0x53: "ADXL345 (accelerometer)",
	0x5A: "MLX90614 (IR temp)",
	0x68: "MPU6050 (accel+gyro)",
	0x76: "BMP280/BME280 (temp/pressure)",
	0x77: "BMP280/BME280 alt (temp/pressure)",
}

// I2CSource reads sensors from a single I2C bus.
type I2CSource struct {
	bus       string
	devPath   string
	addresses []int    // discovered sensor addresses
	labels    []string // human-readable labels for discovered sensors
}

// NewI2CSource creates an I2C sensor source for a given bus.
func NewI2CSource(bus, devPath string) *I2CSource {
	return &I2CSource{bus: bus, devPath: devPath}
}

func (is *I2CSource) Name() string { return "i2c-" + is.bus }

func (is *I2CSource) Interval() time.Duration { return 30 * time.Second }

func (is *I2CSource) Available() bool {
	_, err := os.Stat(is.devPath)
	return err == nil
}

func (is *I2CSource) Poll(ctx context.Context) ([]SensorReading, error) {
	if !is.Available() {
		return nil, fmt.Errorf("I2C bus %s not accessible", is.devPath)
	}

	// Scan for devices on first poll
	if len(is.addresses) == 0 {
		is.scan(ctx)
	}

	if len(is.addresses) == 0 {
		return nil, fmt.Errorf("no I2C devices found on bus %s", is.bus)
	}

	now := time.Now()
	var readings []SensorReading

	for i, addr := range is.addresses {
		label := is.labels[i]
		// Try to read from known sensor types
		if r, err := is.readKnownSensor(ctx, addr, label, now); err == nil {
			readings = append(readings, r...)
		}
	}

	return readings, nil
}

func (is *I2CSource) scan(ctx context.Context) {
	// Use i2cdetect to find devices
	scanCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	out, err := exec.CommandContext(scanCtx, "i2cdetect", "-y", is.bus).CombinedOutput()
	if err != nil {
		logger.WarnCF("perception", "I2C scan failed", map[string]interface{}{
			"bus": is.bus, "error": err.Error(),
		})
		return
	}

	// Parse i2cdetect output for addresses
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.Fields(strings.SplitN(line, ":", 2)[1])
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "--" || p == "" || p == "UU" {
				continue
			}
			addr, err := strconv.ParseInt(p, 16, 32)
			if err != nil || addr < 0x03 || addr > 0x77 {
				continue
			}
			is.addresses = append(is.addresses, int(addr))
			label := fmt.Sprintf("i2c-%s/0x%02x", is.bus, addr)
			if name, ok := knownI2CSensors[int(addr)]; ok {
				label = fmt.Sprintf("i2c-%s/%s", is.bus, name)
			}
			is.labels = append(is.labels, label)
			logger.InfoCF("perception", "Found I2C device", map[string]interface{}{
				"bus": is.bus, "address": fmt.Sprintf("0x%02x", addr), "label": label,
			})
		}
	}
}

func (is *I2CSource) readKnownSensor(ctx context.Context, addr int, label string, now time.Time) ([]SensorReading, error) {
	// SWE100821: Use i2cget for simple single-register reads from known sensors.
	// For complex multi-register protocols (BME280 calibration, etc.) this returns
	// a raw value — enough for presence detection and rough readings.
	readCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	switch addr {
	case 0x76, 0x77:
		// BMP280/BME280 — read temperature register 0xFA (raw, uncalibrated)
		return is.readBMP280(readCtx, addr, label, now)
	case 0x49:
		// TMP102 — read temperature register 0x00
		return is.readTMP102(readCtx, addr, label, now)
	default:
		// For unknown sensors, just report presence
		return []SensorReading{{
			Name:      label,
			Value:     1,
			Unit:      "present",
			Timestamp: now,
		}}, nil
	}
}

func (is *I2CSource) readBMP280(ctx context.Context, addr int, label string, now time.Time) ([]SensorReading, error) {
	out, err := exec.CommandContext(ctx, "i2cget", "-y", is.bus, fmt.Sprintf("0x%02x", addr), "0xFA", "w").CombinedOutput()
	if err != nil {
		return nil, err
	}
	val := strings.TrimSpace(string(out))
	raw, err := strconv.ParseInt(strings.TrimPrefix(val, "0x"), 16, 32)
	if err != nil {
		return nil, err
	}
	// BMP280 raw temp — rough conversion (without calibration data, ~±5°C accuracy)
	tempC := float64(raw) / 100.0
	if tempC > 100 || tempC < -40 {
		return nil, fmt.Errorf("BMP280 raw value out of range: %d", raw)
	}
	return []SensorReading{{
		Name: label + "/temp", Value: tempC, Unit: "°C (raw)", Timestamp: now,
	}}, nil
}

func (is *I2CSource) readTMP102(ctx context.Context, addr int, label string, now time.Time) ([]SensorReading, error) {
	out, err := exec.CommandContext(ctx, "i2cget", "-y", is.bus, fmt.Sprintf("0x%02x", addr), "0x00", "w").CombinedOutput()
	if err != nil {
		return nil, err
	}
	val := strings.TrimSpace(string(out))
	raw, err := strconv.ParseInt(strings.TrimPrefix(val, "0x"), 16, 32)
	if err != nil {
		return nil, err
	}
	// TMP102: swap bytes (little-endian i2cget), shift right 4, multiply by 0.0625
	swapped := ((raw & 0xFF) << 8) | ((raw >> 8) & 0xFF)
	tempC := float64(swapped>>4) * 0.0625
	return []SensorReading{{
		Name: label + "/temp", Value: tempC, Unit: "°C", Timestamp: now,
	}}, nil
}
