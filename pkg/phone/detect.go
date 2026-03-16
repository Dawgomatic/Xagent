// SWE100821: Phone device detection — auto-detect Android (ADB) and iOS (libimobiledevice)
// phones connected via USB. Returns PhoneInfo with type, serial, model, and connection state.

package phone

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// PhoneType identifies the OS of a connected phone.
type PhoneType string

const (
	PhoneAndroid PhoneType = "android"
	PhoneIOS     PhoneType = "ios"
)

// PhoneInfo describes a detected phone device.
type PhoneInfo struct {
	Type      PhoneType `json:"type"`
	Serial    string    `json:"serial"`
	Model     string    `json:"model"`
	Connected bool      `json:"connected"`
}

// Detector finds phones connected via USB.
type Detector struct {
	adbPath    string
	serial     string // pin to a specific serial, empty = auto-detect first
	autoDetect bool
	mu         sync.Mutex
}

// NewDetector creates a phone detector.
//
// adbPath: path to adb binary ("adb" uses $PATH).
// serial: if non-empty, only detect this serial.
// autoDetect: when true, pick the first detected device if serial is empty.
func NewDetector(adbPath, serial string, autoDetect bool) *Detector {
	if adbPath == "" {
		adbPath = "adb"
	}
	return &Detector{
		adbPath:    adbPath,
		serial:     serial,
		autoDetect: autoDetect,
	}
}

// Detect returns info about the connected phone, checking Android then iOS.
// Returns nil if no phone is found.
func (d *Detector) Detect(ctx context.Context) *PhoneInfo {
	d.mu.Lock()
	defer d.mu.Unlock()

	// SWE100821: Try Android first (more common for this use-case)
	if info := d.detectAndroid(ctx); info != nil {
		return info
	}
	return d.detectIOS(ctx)
}

func (d *Detector) detectAndroid(ctx context.Context) *PhoneInfo {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, d.adbPath, "devices", "-l").Output()
	if err != nil {
		return nil
	}

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of") || strings.HasPrefix(line, "*") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		serial := parts[0]
		state := parts[1]
		if state != "device" {
			continue
		}

		// SWE100821: If pinned to a serial, skip non-matching
		if d.serial != "" && serial != d.serial {
			continue
		}
		if !d.autoDetect && d.serial == "" {
			continue
		}

		model := ""
		for _, p := range parts[2:] {
			if strings.HasPrefix(p, "model:") {
				model = strings.TrimPrefix(p, "model:")
				break
			}
		}

		return &PhoneInfo{
			Type:      PhoneAndroid,
			Serial:    serial,
			Model:     model,
			Connected: true,
		}
	}
	return nil
}

func (d *Detector) detectIOS(ctx context.Context) *PhoneInfo {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "idevice_id", "-l").Output()
	if err != nil {
		return nil
	}

	for _, line := range strings.Split(string(out), "\n") {
		serial := strings.TrimSpace(line)
		if serial == "" {
			continue
		}
		if d.serial != "" && serial != d.serial {
			continue
		}
		if !d.autoDetect && d.serial == "" {
			continue
		}

		model := d.iosModel(ctx, serial)
		return &PhoneInfo{
			Type:      PhoneIOS,
			Serial:    serial,
			Model:     model,
			Connected: true,
		}
	}
	return nil
}

func (d *Detector) iosModel(ctx context.Context, udid string) string {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "ideviceinfo", "-u", udid, "-k", "ProductType").Output()
	if err != nil {
		return "unknown"
	}
	m := strings.TrimSpace(string(out))
	if m == "" {
		return "unknown"
	}
	return m
}

// FormatStatus returns a one-line summary suitable for logging or prompts.
func (p *PhoneInfo) FormatStatus() string {
	if p == nil || !p.Connected {
		return "No phone connected"
	}
	return fmt.Sprintf("%s phone connected: serial=%s model=%s", p.Type, p.Serial, p.Model)
}
