// SWE100821: USB device enumeration tool.
// Lists connected USB devices via lsusb or sysfs fallback.
// Enables the agent to see what's physically connected to the host.
package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// USBTool provides USB device enumeration for the agent.
type USBTool struct{}

func NewUSBTool() *USBTool {
	return &USBTool{}
}

func (t *USBTool) Name() string {
	return "usb"
}

func (t *USBTool) Description() string {
	return "List USB devices connected to this machine. Actions: list (enumerate all connected USB devices with vendor/product info), detail (show detailed info for a specific bus/device)."
}

func (t *USBTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"list", "detail"},
				"description": "Action: list (show all USB devices) or detail (show info for a specific device)",
			},
			"device": map[string]interface{}{
				"type":        "string",
				"description": "Bus:Device ID (e.g. '001:003') for detail action. Optional.",
			},
		},
		"required": []string{"action"},
	}
}

func (t *USBTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	action, _ := args["action"].(string)
	if action == "" {
		action = "list"
	}

	switch action {
	case "list":
		return t.listDevices()
	case "detail":
		device, _ := args["device"].(string)
		return t.detailDevice(device)
	default:
		return ErrorResult(fmt.Sprintf("Unknown action: %s. Use 'list' or 'detail'.", action))
	}
}

// listDevices tries lsusb first, falls back to sysfs.
func (t *USBTool) listDevices() *ToolResult {
	if out, err := exec.Command("lsusb").Output(); err == nil {
		lines := strings.TrimSpace(string(out))
		if lines == "" {
			return NewToolResult("No USB devices found.")
		}
		return NewToolResult(lines)
	}
	return t.listFromSysfs()
}

// listFromSysfs reads /sys/bus/usb/devices/ as fallback when lsusb is unavailable.
func (t *USBTool) listFromSysfs() *ToolResult {
	sysPath := "/sys/bus/usb/devices"
	entries, err := os.ReadDir(sysPath)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Cannot enumerate USB devices: lsusb not found and %s unreadable: %v", sysPath, err))
	}

	var sb strings.Builder
	for _, e := range entries {
		devPath := filepath.Join(sysPath, e.Name())
		vendor := readSysfsFile(filepath.Join(devPath, "idVendor"))
		product := readSysfsFile(filepath.Join(devPath, "idProduct"))
		manufacturer := readSysfsFile(filepath.Join(devPath, "manufacturer"))
		productName := readSysfsFile(filepath.Join(devPath, "product"))

		if vendor == "" && product == "" {
			continue
		}

		line := fmt.Sprintf("%s: %s:%s", e.Name(), vendor, product)
		if manufacturer != "" || productName != "" {
			line += fmt.Sprintf(" %s %s", manufacturer, productName)
		}
		sb.WriteString(line + "\n")
	}

	result := strings.TrimSpace(sb.String())
	if result == "" {
		return NewToolResult("No USB devices found via sysfs.")
	}
	return NewToolResult(result)
}

func (t *USBTool) detailDevice(device string) *ToolResult {
	if device == "" {
		return ErrorResult("device parameter required for detail action (e.g. '001:003')")
	}

	out, err := exec.Command("lsusb", "-v", "-s", device).Output()
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to get details for device %s: %v", device, err))
	}
	result := strings.TrimSpace(string(out))
	if len(result) > 3000 {
		result = result[:3000] + "\n...(truncated)"
	}
	return NewToolResult(result)
}

func readSysfsFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
