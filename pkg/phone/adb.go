// SWE100821: ADB command wrapper for Android phone access.
// Provides Run() for arbitrary commands and convenience methods for common operations.
// Reuses deny-pattern approach from pkg/tools/shell.go for destructive command blocking.

package phone

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

// ADB wraps the Android Debug Bridge CLI.
type ADB struct {
	path         string
	serial       string
	timeout      time.Duration
	denyPatterns []*regexp.Regexp
}

// NewADB creates an ADB wrapper for a specific device.
//
// adbPath: path to adb binary.
// serial: device serial (from Detector).
// extraDeny: additional shell patterns to block (from config).
func NewADB(adbPath, serial string, extraDeny []string) *ADB {
	if adbPath == "" {
		adbPath = "adb"
	}

	// SWE100821: Default deny patterns — prevent bricking/data-loss
	deny := []*regexp.Regexp{
		regexp.MustCompile(`\brm\s+-rf\s+/`),
		regexp.MustCompile(`\breboot\s+bootloader\b`),
		regexp.MustCompile(`\bfastboot\b`),
		regexp.MustCompile(`\bflash\b`),
		regexp.MustCompile(`\bformat\b.*\b/data\b`),
		regexp.MustCompile(`\bwipe\b`),
		regexp.MustCompile(`\bfactory.?reset\b`),
		regexp.MustCompile(`\bdd\s+if=`),
	}
	for _, pat := range extraDeny {
		if re, err := regexp.Compile(pat); err == nil {
			deny = append(deny, re)
		}
	}

	return &ADB{
		path:         adbPath,
		serial:       serial,
		timeout:      defaultTimeout,
		denyPatterns: deny,
	}
}

// Run executes an arbitrary adb command: adb -s <serial> <args...>.
// Returns stdout, stderr, and any error.
func (a *ADB) Run(ctx context.Context, args ...string) (string, string, error) {
	cmdArgs := make([]string, 0, len(args)+2)
	if a.serial != "" {
		cmdArgs = append(cmdArgs, "-s", a.serial)
	}
	cmdArgs = append(cmdArgs, args...)

	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, a.path, cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// Shell runs `adb shell <cmd>` after checking deny patterns.
func (a *ADB) Shell(ctx context.Context, command string) (string, error) {
	if err := a.guardCommand(command); err != nil {
		return "", err
	}
	stdout, stderr, err := a.Run(ctx, "shell", command)
	if err != nil {
		if stderr != "" {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr))
		}
		return "", err
	}
	return stdout, nil
}

// guardCommand checks a shell command against deny patterns.
func (a *ADB) guardCommand(command string) error {
	lower := strings.ToLower(command)
	for _, pat := range a.denyPatterns {
		if pat.MatchString(lower) {
			return fmt.Errorf("command blocked by safety filter: matches %q", pat.String())
		}
	}
	return nil
}

// BatteryInfo holds parsed battery state.
type BatteryInfo struct {
	Level    int    `json:"level"`
	Status   string `json:"status"`
	Plugged  string `json:"plugged"`
	Temp     int    `json:"temperature_c"`
	Health   string `json:"health"`
}

// GetBattery returns parsed battery info from `dumpsys battery`.
func (a *ADB) GetBattery(ctx context.Context) (*BatteryInfo, error) {
	out, err := a.Shell(ctx, "dumpsys battery")
	if err != nil {
		return nil, fmt.Errorf("dumpsys battery: %w", err)
	}
	info := &BatteryInfo{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch strings.ToLower(k) {
		case "level":
			info.Level, _ = strconv.Atoi(v)
		case "status":
			info.Status = batteryStatusName(v)
		case "plugged":
			info.Plugged = pluggedName(v)
		case "temperature":
			t, _ := strconv.Atoi(v)
			info.Temp = t / 10
		case "health":
			info.Health = v
		}
	}
	return info, nil
}

func batteryStatusName(code string) string {
	switch code {
	case "2":
		return "charging"
	case "3":
		return "discharging"
	case "4":
		return "not_charging"
	case "5":
		return "full"
	default:
		return "unknown"
	}
}

func pluggedName(code string) string {
	switch code {
	case "1":
		return "ac"
	case "2":
		return "usb"
	case "4":
		return "wireless"
	default:
		return "none"
	}
}

// GetScreenState returns whether the screen is on.
func (a *ADB) GetScreenState(ctx context.Context) (bool, error) {
	out, err := a.Shell(ctx, "dumpsys display | grep mScreenState")
	if err != nil {
		out2, err2 := a.Shell(ctx, "dumpsys power | grep mWakefulness")
		if err2 != nil {
			return false, fmt.Errorf("screen state: %w", err)
		}
		out = out2
	}
	lower := strings.ToLower(out)
	return strings.Contains(lower, "on") || strings.Contains(lower, "awake"), nil
}

// Screenshot captures the screen and saves to outPath on the host.
func (a *ADB) Screenshot(ctx context.Context, outPath string) error {
	stdout, stderr, err := a.Run(ctx, "exec-out", "screencap", "-p")
	if err != nil {
		return fmt.Errorf("screencap: %w: %s", err, stderr)
	}
	return os.WriteFile(outPath, []byte(stdout), 0644)
}

// IsAppRunning checks if a package has a running process.
func (a *ADB) IsAppRunning(ctx context.Context, pkg string) (bool, error) {
	out, err := a.Shell(ctx, fmt.Sprintf("pidof %s", pkg))
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(out) != "", nil
}

// LaunchApp starts an app by package name using monkey.
func (a *ADB) LaunchApp(ctx context.Context, pkg string) error {
	_, err := a.Shell(ctx, fmt.Sprintf("monkey -p %s -c android.intent.category.LAUNCHER 1", pkg))
	return err
}

// ListPackages returns installed package names.
func (a *ADB) ListPackages(ctx context.Context) ([]string, error) {
	out, err := a.Shell(ctx, "pm list packages")
	if err != nil {
		return nil, err
	}
	var pkgs []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if p, ok := strings.CutPrefix(line, "package:"); ok {
			pkgs = append(pkgs, p)
		}
	}
	return pkgs, nil
}

// SendTap sends a tap event at screen coordinates.
func (a *ADB) SendTap(ctx context.Context, x, y int) error {
	_, err := a.Shell(ctx, fmt.Sprintf("input tap %d %d", x, y))
	return err
}

// SendSwipe sends a swipe gesture between two points over durationMs.
func (a *ADB) SendSwipe(ctx context.Context, x1, y1, x2, y2, durationMs int) error {
	_, err := a.Shell(ctx, fmt.Sprintf("input swipe %d %d %d %d %d", x1, y1, x2, y2, durationMs))
	return err
}

// SendText types text into the currently focused field.
// SWE100821: Escape shell-special chars to prevent ADB shell injection.
// The ADB "input text" command uses %s for spaces and requires escaping of
// quotes, backslashes, parentheses, and other shell metacharacters.
func (a *ADB) SendText(ctx context.Context, text string) error {
	r := strings.NewReplacer(
		" ", "%s",
		"'", "\\'",
		"\"", "\\\"",
		"\\", "\\\\",
		"(", "\\(",
		")", "\\)",
		"&", "\\&",
		";", "\\;",
		"|", "\\|",
		"$", "\\$",
		"`", "\\`",
	)
	escaped := r.Replace(text)
	_, err := a.Shell(ctx, fmt.Sprintf("input text %s", escaped))
	return err
}

// Push copies a file from host to device.
func (a *ADB) Push(ctx context.Context, localPath, remotePath string) error {
	_, stderr, err := a.Run(ctx, "push", localPath, remotePath)
	if err != nil {
		return fmt.Errorf("push: %w: %s", err, stderr)
	}
	return nil
}

// Pull copies a file from device to host.
func (a *ADB) Pull(ctx context.Context, remotePath, localPath string) error {
	_, stderr, err := a.Run(ctx, "pull", remotePath, localPath)
	if err != nil {
		return fmt.Errorf("pull: %w: %s", err, stderr)
	}
	return nil
}

// Install installs an APK on the device.
func (a *ADB) Install(ctx context.Context, apkPath string) error {
	_, stderr, err := a.Run(ctx, "install", "-r", apkPath)
	if err != nil {
		return fmt.Errorf("install: %w: %s", err, stderr)
	}
	return nil
}
