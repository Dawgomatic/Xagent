// SWE100821: Sensor source auto-discovery — probes the system for anything that can
// provide sensor data: USB-attached phones (ADB), I2C buses, GPIO, host thermals.
// Returns a list of SensorSources ready to poll. Re-discovery runs periodically to
// detect hot-plugged devices.

package sensors

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// DiscoveryResult holds the outcome of a sensor discovery sweep.
type DiscoveryResult struct {
	Sources    []SensorSource
	Discovered time.Time
	Summary    string
}

// SWE100821: workspace path, set by DiscoverWithWorkspace. Defaults to /tmp.
var discoveryWorkspace string

// Discover probes the system and returns all available sensor sources.
// Safe to call repeatedly — will re-probe on each call.
func Discover() *DiscoveryResult {
	workspace := discoveryWorkspace
	if workspace == "" {
		workspace = "/tmp"
	}
	result := &DiscoveryResult{Discovered: time.Now()}
	var summaryParts []string

	// 1. Host sensors (always available on Linux)
	host := NewXavierSource()
	if host.Available() {
		result.Sources = append(result.Sources, host)
		summaryParts = append(summaryParts, "host-thermals")
		logger.InfoC("perception", "Discovered host sensor source (thermals, RAM, disk, load)")
	}

	// 2. Phone via ADB
	phone := discoverPhone()
	if phone != nil {
		result.Sources = append(result.Sources, phone)
		summaryParts = append(summaryParts, "phone-adb")
		logger.InfoC("perception", "Discovered phone sensor source via ADB")
	}

	// 3. I2C buses
	i2cSources := discoverI2C()
	for _, src := range i2cSources {
		result.Sources = append(result.Sources, src)
		summaryParts = append(summaryParts, "i2c:"+src.Name())
	}

	// 4. GPIO (check for gpiochip devices)
	if gpioAvail := discoverGPIO(); gpioAvail {
		summaryParts = append(summaryParts, "gpio-available")
		logger.InfoC("perception", "GPIO chips detected (not yet polling — needs sensor config)")
	}

	// 5. Cameras (USB webcam, CSI, phone camera/screen)
	cameraSrc := discoverCameras(workspace)
	if cameraSrc != nil {
		result.Sources = append(result.Sources, cameraSrc)
		summaryParts = append(summaryParts, "cameras")
	}

	// 6. USB sensors (serial devices that might be sensor boards)
	usbSensors := discoverUSBSensors()
	if len(usbSensors) > 0 {
		summaryParts = append(summaryParts, strings.Join(usbSensors, ","))
	}

	if len(summaryParts) == 0 {
		result.Summary = "no sensor sources discovered"
	} else {
		result.Summary = strings.Join(summaryParts, " | ")
	}

	logger.InfoCF("perception", "Discovery complete", map[string]interface{}{
		"sources": len(result.Sources),
		"summary": result.Summary,
	})

	return result
}

func discoverPhone() *PhoneSource {
	adbPath, err := exec.LookPath("adb")
	if err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, adbPath, "devices").CombinedOutput()
	if err != nil {
		return nil
	}

	// Parse "adb devices" output for authorized devices
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List") || strings.HasPrefix(line, "*") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == "device" {
			src := NewPhoneSource(adbPath, parts[0])
			if src.Available() {
				return src
			}
		}
	}
	return nil
}

func discoverI2C() []SensorSource {
	matches, err := filepath.Glob("/dev/i2c-*")
	if err != nil || len(matches) == 0 {
		return nil
	}

	var sources []SensorSource
	for _, path := range matches {
		bus := strings.TrimPrefix(filepath.Base(path), "i2c-")
		src := NewI2CSource(bus, path)
		if src.Available() {
			sources = append(sources, src)
			logger.InfoCF("perception", "Discovered I2C bus", map[string]interface{}{"bus": bus, "path": path})
		}
	}
	return sources
}

func discoverGPIO() bool {
	matches, _ := filepath.Glob("/dev/gpiochip*")
	return len(matches) > 0
}

// SetDiscoveryWorkspace sets the workspace path used for camera frame storage.
func SetDiscoveryWorkspace(ws string) {
	discoveryWorkspace = ws
}

func discoverCameras(workspace string) *CameraSource {
	// Check for any video devices or phone ADB
	hasVideo := false
	matches, _ := filepath.Glob("/dev/video*")
	if len(matches) > 0 {
		hasVideo = true
	}

	hasPhone := false
	adbPath, err := exec.LookPath("adb")
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, adbPath, "get-state").CombinedOutput()
		if err == nil && strings.TrimSpace(string(out)) == "device" {
			hasPhone = true
		}
	}

	if !hasVideo && !hasPhone {
		return nil
	}

	src := NewCameraSource(CameraSourceConfig{
		Workspace: workspace,
		ADBPath:   adbPath,
	})
	if src.Available() {
		logger.InfoCF("perception", "Discovered camera sources", map[string]interface{}{
			"usb_cameras": len(matches),
			"phone":       hasPhone,
		})
		return src
	}
	return nil
}

func discoverUSBSensors() []string {
	var found []string

	// Check for common USB sensor serial devices
	patterns := []string{
		"/dev/ttyUSB*",
		"/dev/ttyACM*",
	}
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			info, err := os.Stat(m)
			if err != nil {
				continue
			}
			found = append(found, "usb-serial:"+info.Name())
			logger.InfoCF("perception", "Found USB serial device", map[string]interface{}{"path": m})
		}
	}
	return found
}
