// SWE100821: Phone tool — gives the agent full access to a USB-attached phone
// via ADB (Android) or libimobiledevice (iOS). Auto-detects phone type.
// Reused: pkg/tools/base.go L6-11 (Tool interface), pkg/tools/result.go L8-33 (ToolResult).

package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Dawgomatic/Xagent/pkg/config"
	"github.com/Dawgomatic/Xagent/pkg/phone"
)

// PhoneTool exposes phone operations to the agent via the Tool interface.
type PhoneTool struct {
	workspace string
	cfg       config.PhoneConfig
	detector  *phone.Detector
}

// NewPhoneTool creates a phone tool with the given config.
func NewPhoneTool(workspace string, cfg config.PhoneConfig) *PhoneTool {
	return &PhoneTool{
		workspace: workspace,
		cfg:       cfg,
		detector:  phone.NewDetector(cfg.ADBPath, cfg.Serial, cfg.AutoDetect),
	}
}

func (t *PhoneTool) Name() string        { return "phone" }
func (t *PhoneTool) Description() string {
	return "Interact with a USB-attached phone (Android via ADB, iOS via libimobiledevice). " +
		"Actions: status, screenshot, shell, app_list, app_launch, app_running, tap, swipe, text, push, pull, install, raw"
}

func (t *PhoneTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "The action to perform",
				"enum":        []string{"status", "screenshot", "shell", "app_list", "app_launch", "app_running", "tap", "swipe", "text", "push", "pull", "install", "raw"},
			},
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Shell command (for 'shell' action) or raw ADB args (for 'raw' action)",
			},
			"package": map[string]interface{}{
				"type":        "string",
				"description": "App package name (for app_launch, app_running, install)",
			},
			"x": map[string]interface{}{
				"type":        "number",
				"description": "X coordinate (for tap, swipe start)",
			},
			"y": map[string]interface{}{
				"type":        "number",
				"description": "Y coordinate (for tap, swipe start)",
			},
			"x2": map[string]interface{}{
				"type":        "number",
				"description": "End X coordinate (for swipe)",
			},
			"y2": map[string]interface{}{
				"type":        "number",
				"description": "End Y coordinate (for swipe)",
			},
			"duration": map[string]interface{}{
				"type":        "number",
				"description": "Swipe duration in milliseconds (default 300)",
			},
			"text": map[string]interface{}{
				"type":        "string",
				"description": "Text to type (for 'text' action)",
			},
			"local_path": map[string]interface{}{
				"type":        "string",
				"description": "Host file path (for push source / pull destination / install APK)",
			},
			"remote_path": map[string]interface{}{
				"type":        "string",
				"description": "Device file path (for push destination / pull source)",
			},
			"filename": map[string]interface{}{
				"type":        "string",
				"description": "Screenshot filename (for 'screenshot', saved in workspace)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *PhoneTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	action, _ := args["action"].(string)
	if action == "" {
		return ErrorResult("'action' parameter is required")
	}

	info := t.detector.Detect(ctx)
	if info == nil || !info.Connected {
		return ErrorResult("No phone detected via USB. Ensure the device is connected and ADB/USB debugging is enabled.")
	}

	switch info.Type {
	case phone.PhoneAndroid:
		return t.executeAndroid(ctx, info, action, args)
	case phone.PhoneIOS:
		return t.executeIOS(ctx, info, action, args)
	default:
		return ErrorResult(fmt.Sprintf("Unsupported phone type: %s", info.Type))
	}
}

func (t *PhoneTool) executeAndroid(ctx context.Context, info *phone.PhoneInfo, action string, args map[string]interface{}) *ToolResult {
	adb := phone.NewADB(t.cfg.ADBPath, info.Serial, t.cfg.DenyShell)

	switch action {
	case "status":
		return t.androidStatus(ctx, adb, info)
	case "screenshot":
		return t.androidScreenshot(ctx, adb, args)
	case "shell":
		return t.androidShell(ctx, adb, args)
	case "app_list":
		return t.androidAppList(ctx, adb)
	case "app_launch":
		return t.androidAppLaunch(ctx, adb, args)
	case "app_running":
		return t.androidAppRunning(ctx, adb, args)
	case "tap":
		return t.androidTap(ctx, adb, args)
	case "swipe":
		return t.androidSwipe(ctx, adb, args)
	case "text":
		return t.androidText(ctx, adb, args)
	case "push":
		return t.androidPush(ctx, adb, args)
	case "pull":
		return t.androidPull(ctx, adb, args)
	case "install":
		return t.androidInstall(ctx, adb, args)
	case "raw":
		return t.androidRaw(ctx, adb, args)
	default:
		return ErrorResult(fmt.Sprintf("Unknown action: %s", action))
	}
}

func (t *PhoneTool) androidStatus(ctx context.Context, adb *phone.ADB, info *phone.PhoneInfo) *ToolResult {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Phone: %s (serial: %s)\n", info.Model, info.Serial))

	bat, err := adb.GetBattery(ctx)
	if err == nil {
		sb.WriteString(fmt.Sprintf("Battery: %d%% (%s, plugged: %s, temp: %d°C, health: %s)\n",
			bat.Level, bat.Status, bat.Plugged, bat.Temp, bat.Health))
	} else {
		sb.WriteString(fmt.Sprintf("Battery: error — %s\n", err))
	}

	screenOn, err := adb.GetScreenState(ctx)
	if err == nil {
		state := "OFF"
		if screenOn {
			state = "ON"
		}
		sb.WriteString(fmt.Sprintf("Screen: %s\n", state))
	}

	waRunning, _ := adb.IsAppRunning(ctx, "com.whatsapp")
	if waRunning {
		sb.WriteString("WhatsApp: running\n")
	} else {
		sb.WriteString("WhatsApp: not running\n")
	}

	return SilentResult(sb.String())
}

func (t *PhoneTool) androidScreenshot(ctx context.Context, adb *phone.ADB, args map[string]interface{}) *ToolResult {
	filename, _ := args["filename"].(string)
	if filename == "" {
		filename = "phone_screenshot.png"
	}
	// SWE100821: Use filepath.Base to prevent path traversal — "../../../etc/passwd" becomes "passwd"
	outPath := filepath.Join(t.workspace, filepath.Base(filename))

	if err := adb.Screenshot(ctx, outPath); err != nil {
		return ErrorResult(fmt.Sprintf("Screenshot failed: %s", err))
	}
	return SilentResult(fmt.Sprintf("Screenshot saved to %s", outPath))
}

func (t *PhoneTool) androidShell(ctx context.Context, adb *phone.ADB, args map[string]interface{}) *ToolResult {
	command, _ := args["command"].(string)
	if command == "" {
		return ErrorResult("'command' parameter required for shell action")
	}
	out, err := adb.Shell(ctx, command)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Shell error: %s", err))
	}
	return SilentResult(out)
}

func (t *PhoneTool) androidAppList(ctx context.Context, adb *phone.ADB) *ToolResult {
	pkgs, err := adb.ListPackages(ctx)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to list packages: %s", err))
	}
	return SilentResult(fmt.Sprintf("Installed packages (%d):\n%s", len(pkgs), strings.Join(pkgs, "\n")))
}

func (t *PhoneTool) androidAppLaunch(ctx context.Context, adb *phone.ADB, args map[string]interface{}) *ToolResult {
	pkg, _ := args["package"].(string)
	if pkg == "" {
		return ErrorResult("'package' parameter required for app_launch")
	}
	if err := adb.LaunchApp(ctx, pkg); err != nil {
		return ErrorResult(fmt.Sprintf("Failed to launch %s: %s", pkg, err))
	}
	return SilentResult(fmt.Sprintf("Launched %s", pkg))
}

func (t *PhoneTool) androidAppRunning(ctx context.Context, adb *phone.ADB, args map[string]interface{}) *ToolResult {
	pkg, _ := args["package"].(string)
	if pkg == "" {
		return ErrorResult("'package' parameter required for app_running")
	}
	running, err := adb.IsAppRunning(ctx, pkg)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to check %s: %s", pkg, err))
	}
	if running {
		return SilentResult(fmt.Sprintf("%s is running", pkg))
	}
	return SilentResult(fmt.Sprintf("%s is NOT running", pkg))
}

func (t *PhoneTool) androidTap(ctx context.Context, adb *phone.ADB, args map[string]interface{}) *ToolResult {
	x, y, err := extractCoords(args, "x", "y")
	if err != nil {
		return ErrorResult(err.Error())
	}
	if err := adb.SendTap(ctx, x, y); err != nil {
		return ErrorResult(fmt.Sprintf("Tap failed: %s", err))
	}
	return SilentResult(fmt.Sprintf("Tapped at (%d, %d)", x, y))
}

func (t *PhoneTool) androidSwipe(ctx context.Context, adb *phone.ADB, args map[string]interface{}) *ToolResult {
	x1, y1, err := extractCoords(args, "x", "y")
	if err != nil {
		return ErrorResult(fmt.Sprintf("Start coords: %s", err))
	}
	x2, y2, err := extractCoords(args, "x2", "y2")
	if err != nil {
		return ErrorResult(fmt.Sprintf("End coords: %s", err))
	}
	dur := intArg(args, "duration", 300)
	if err := adb.SendSwipe(ctx, x1, y1, x2, y2, dur); err != nil {
		return ErrorResult(fmt.Sprintf("Swipe failed: %s", err))
	}
	return SilentResult(fmt.Sprintf("Swiped from (%d,%d) to (%d,%d) over %dms", x1, y1, x2, y2, dur))
}

func (t *PhoneTool) androidText(ctx context.Context, adb *phone.ADB, args map[string]interface{}) *ToolResult {
	text, _ := args["text"].(string)
	if text == "" {
		return ErrorResult("'text' parameter required for text action")
	}
	if err := adb.SendText(ctx, text); err != nil {
		return ErrorResult(fmt.Sprintf("Text input failed: %s", err))
	}
	return SilentResult(fmt.Sprintf("Typed: %s", text))
}

func (t *PhoneTool) androidPush(ctx context.Context, adb *phone.ADB, args map[string]interface{}) *ToolResult {
	local, _ := args["local_path"].(string)
	remote, _ := args["remote_path"].(string)
	if local == "" || remote == "" {
		return ErrorResult("'local_path' and 'remote_path' required for push")
	}
	if err := adb.Push(ctx, local, remote); err != nil {
		return ErrorResult(fmt.Sprintf("Push failed: %s", err))
	}
	return SilentResult(fmt.Sprintf("Pushed %s → %s", local, remote))
}

func (t *PhoneTool) androidPull(ctx context.Context, adb *phone.ADB, args map[string]interface{}) *ToolResult {
	remote, _ := args["remote_path"].(string)
	local, _ := args["local_path"].(string)
	if remote == "" || local == "" {
		return ErrorResult("'remote_path' and 'local_path' required for pull")
	}
	if err := adb.Pull(ctx, remote, local); err != nil {
		return ErrorResult(fmt.Sprintf("Pull failed: %s", err))
	}
	return SilentResult(fmt.Sprintf("Pulled %s → %s", remote, local))
}

func (t *PhoneTool) androidInstall(ctx context.Context, adb *phone.ADB, args map[string]interface{}) *ToolResult {
	apk, _ := args["local_path"].(string)
	if apk == "" {
		if pkg, _ := args["package"].(string); pkg != "" {
			apk = pkg
		}
	}
	if apk == "" {
		return ErrorResult("'local_path' (APK path) required for install")
	}
	if err := adb.Install(ctx, apk); err != nil {
		return ErrorResult(fmt.Sprintf("Install failed: %s", err))
	}
	return SilentResult(fmt.Sprintf("Installed %s", apk))
}

func (t *PhoneTool) androidRaw(ctx context.Context, adb *phone.ADB, args map[string]interface{}) *ToolResult {
	command, _ := args["command"].(string)
	if command == "" {
		return ErrorResult("'command' parameter required for raw action (space-separated adb args)")
	}
	parts := strings.Fields(command)
	stdout, stderr, err := adb.Run(ctx, parts...)
	if err != nil {
		msg := fmt.Sprintf("ADB error: %s", err)
		if stderr != "" {
			msg += "\nstderr: " + strings.TrimSpace(stderr)
		}
		return ErrorResult(msg)
	}
	result := stdout
	if stderr != "" {
		result += "\nstderr: " + stderr
	}
	return SilentResult(result)
}

// --- iOS actions ---

func (t *PhoneTool) executeIOS(ctx context.Context, info *phone.PhoneInfo, action string, args map[string]interface{}) *ToolResult {
	ios := phone.NewIOS(info.Serial)

	switch action {
	case "status":
		return t.iosStatus(ctx, ios, info)
	case "screenshot":
		return t.iosScreenshot(ctx, ios, args)
	case "shell":
		return ErrorResult("Shell access is not available on iOS devices. Use specific actions instead.")
	case "app_list":
		return t.iosAppList(ctx, ios)
	case "raw":
		return ErrorResult("Raw ADB commands are not available on iOS devices.")
	default:
		return ErrorResult(fmt.Sprintf("Action '%s' is not supported on iOS devices. Supported: status, screenshot, app_list", action))
	}
}

func (t *PhoneTool) iosStatus(ctx context.Context, ios *phone.IOS, info *phone.PhoneInfo) *ToolResult {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("iOS Phone: %s (udid: %s)\n", info.Model, info.Serial))

	bat, err := ios.GetBattery(ctx)
	if err == nil {
		sb.WriteString(fmt.Sprintf("Battery: %d%% (%s, plugged: %s)\n", bat.Level, bat.Status, bat.Plugged))
	} else {
		sb.WriteString(fmt.Sprintf("Battery: error — %s\n", err))
	}

	devInfo, err := ios.GetDeviceInfo(ctx)
	if err == nil {
		if v, ok := devInfo["ProductVersion"]; ok {
			sb.WriteString(fmt.Sprintf("iOS Version: %s\n", v))
		}
		if v, ok := devInfo["DeviceName"]; ok {
			sb.WriteString(fmt.Sprintf("Device Name: %s\n", v))
		}
	}

	return SilentResult(sb.String())
}

func (t *PhoneTool) iosScreenshot(ctx context.Context, ios *phone.IOS, args map[string]interface{}) *ToolResult {
	filename, _ := args["filename"].(string)
	if filename == "" {
		filename = "phone_screenshot.png"
	}
	outPath := filepath.Join(t.workspace, filename)

	if err := ios.Screenshot(ctx, outPath); err != nil {
		return ErrorResult(fmt.Sprintf("Screenshot failed: %s", err))
	}
	return SilentResult(fmt.Sprintf("Screenshot saved to %s", outPath))
}

func (t *PhoneTool) iosAppList(ctx context.Context, ios *phone.IOS) *ToolResult {
	apps, err := ios.ListApps(ctx)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to list apps: %s", err))
	}
	return SilentResult(fmt.Sprintf("Installed apps (%d):\n%s", len(apps), strings.Join(apps, "\n")))
}

// --- helpers ---

func extractCoords(args map[string]interface{}, xKey, yKey string) (int, int, error) {
	xVal, ok := args[xKey]
	if !ok {
		return 0, 0, fmt.Errorf("'%s' parameter required", xKey)
	}
	yVal, ok := args[yKey]
	if !ok {
		return 0, 0, fmt.Errorf("'%s' parameter required", yKey)
	}
	x, err := toInt(xVal)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid %s: %w", xKey, err)
	}
	y, err := toInt(yVal)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid %s: %w", yKey, err)
	}
	return x, y, nil
}

func toInt(v interface{}) (int, error) {
	switch val := v.(type) {
	case float64:
		return int(val), nil
	case int:
		return val, nil
	case string:
		return strconv.Atoi(val)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", v)
	}
}

func intArg(args map[string]interface{}, key string, defaultVal int) int {
	v, ok := args[key]
	if !ok {
		return defaultVal
	}
	i, err := toInt(v)
	if err != nil {
		return defaultVal
	}
	return i
}
