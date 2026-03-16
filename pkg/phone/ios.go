// SWE100821: iOS phone access via libimobiledevice CLI tools.
// Provides the same interface shape as ADB for basic operations.
// Returns clear errors when libimobiledevice is not installed.

package phone

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// IOS wraps libimobiledevice CLI tools for iOS device access.
type IOS struct {
	udid    string
	timeout time.Duration
}

// NewIOS creates an iOS device wrapper.
func NewIOS(udid string) *IOS {
	return &IOS{
		udid:    udid,
		timeout: defaultTimeout,
	}
}

// run executes a libimobiledevice command with the device UDID.
func (i *IOS) run(ctx context.Context, cmd string, args ...string) (string, string, error) {
	if _, err := exec.LookPath(cmd); err != nil {
		return "", "", fmt.Errorf("%s not found — install libimobiledevice: apt install libimobiledevice-utils", cmd)
	}

	fullArgs := make([]string, 0, len(args)+2)
	if i.udid != "" {
		fullArgs = append(fullArgs, "-u", i.udid)
	}
	fullArgs = append(fullArgs, args...)

	ctx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	c := exec.CommandContext(ctx, cmd, fullArgs...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	return stdout.String(), stderr.String(), err
}

// GetBattery returns battery info from ideviceinfo.
func (i *IOS) GetBattery(ctx context.Context) (*BatteryInfo, error) {
	out, _, err := i.run(ctx, "ideviceinfo", "-q", "com.apple.mobile.battery")
	if err != nil {
		return nil, fmt.Errorf("ideviceinfo battery: %w", err)
	}

	info := &BatteryInfo{}
	for _, line := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch k {
		case "BatteryCurrentCapacity":
			info.Level, _ = strconv.Atoi(v)
		case "BatteryIsCharging":
			if v == "true" {
				info.Status = "charging"
			} else {
				info.Status = "discharging"
			}
		case "ExternalConnected":
			if v == "true" {
				info.Plugged = "usb"
			} else {
				info.Plugged = "none"
			}
		}
	}
	return info, nil
}

// Screenshot captures the iOS screen and saves to outPath.
func (i *IOS) Screenshot(ctx context.Context, outPath string) error {
	_, stderr, err := i.run(ctx, "idevicescreenshot", outPath)
	if err != nil {
		return fmt.Errorf("idevicescreenshot: %w: %s", err, stderr)
	}
	return nil
}

// GetDeviceInfo returns a map of device properties.
func (i *IOS) GetDeviceInfo(ctx context.Context) (map[string]string, error) {
	out, _, err := i.run(ctx, "ideviceinfo")
	if err != nil {
		return nil, fmt.Errorf("ideviceinfo: %w", err)
	}

	info := make(map[string]string)
	for _, line := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		info[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return info, nil
}

// Shell is not supported on iOS — returns an informative error.
func (i *IOS) Shell(ctx context.Context, command string) (string, error) {
	return "", fmt.Errorf("shell access not available on iOS devices — use specific tool actions instead")
}

// ListApps returns installed app bundle IDs via ideviceinstaller.
func (i *IOS) ListApps(ctx context.Context) ([]string, error) {
	out, _, err := i.run(ctx, "ideviceinstaller", "-l")
	if err != nil {
		return nil, fmt.Errorf("ideviceinstaller: %w", err)
	}
	var apps []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Total") || strings.HasPrefix(line, "CFBundle") {
			continue
		}
		parts := strings.SplitN(line, ",", 2)
		if len(parts) > 0 {
			apps = append(apps, strings.TrimSpace(parts[0]))
		}
	}
	return apps, nil
}
