// SWE100821: Camera tool — gives the agent on-demand access to capture images from
// any discovered camera (USB webcam, CSI, phone front/rear/screen). The agent decides
// when to look; captures are not timer-driven. Vision analysis via Ollama is automatic.

package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Dawgomatic/Xagent/pkg/sensors"
)

// CameraTool exposes camera capture to the agent via the Tool interface.
type CameraTool struct {
	cameraSource *sensors.CameraSource
}

// NewCameraTool creates a camera tool backed by the perception camera source.
func NewCameraTool(cs *sensors.CameraSource) *CameraTool {
	return &CameraTool{cameraSource: cs}
}

func (t *CameraTool) Name() string { return "camera" }

func (t *CameraTool) Description() string {
	return "Capture images from available cameras (phone rear/front/screen, USB webcam, CSI). " +
		"Use this to see your surroundings, check on something physical, or when the user asks what you see. " +
		"Actions: capture (take a photo from one camera), look (capture from all cameras), list (show available cameras)."
}

func (t *CameraTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"capture", "look", "list"},
				"description": "capture: photo from one camera; look: photo from all cameras; list: show available cameras",
			},
			"camera": map[string]interface{}{
				"type":        "string",
				"description": "Camera name to capture from (e.g. 'phone-rear', 'phone-front', 'phone-screen', 'video0'). Defaults to rear camera if omitted.",
			},
			"question": map[string]interface{}{
				"type":        "string",
				"description": "Optional question to ask about the image (e.g. 'Is anyone in the room?'). Default: general scene description.",
			},
		},
		"required": []string{"action"},
	}
}

func (t *CameraTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
	if t.cameraSource == nil {
		return ErrorResult("Camera subsystem not available — no cameras discovered at startup.")
	}

	action, _ := args["action"].(string)
	switch action {
	case "list":
		return t.listCameras()
	case "capture":
		camera, _ := args["camera"].(string)
		return t.capture(ctx, camera)
	case "look":
		return t.lookAll(ctx)
	default:
		return ErrorResult(fmt.Sprintf("Unknown action %q. Use: capture, look, or list.", action))
	}
}

// SWE100821: listCameras shows discovered cameras with types.
func (t *CameraTool) listCameras() *ToolResult {
	cams := t.cameraSource.ListCameras()
	if len(cams) == 0 {
		return NewToolResult("No cameras discovered. Check USB connections or phone ADB.")
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Available cameras (%d):\n", len(cams)))
	for _, c := range cams {
		b.WriteString(fmt.Sprintf("  - %s (type: %s", c.Name, c.Type))
		if c.Path != "" {
			b.WriteString(fmt.Sprintf(", path: %s", c.Path))
		}
		b.WriteString(")\n")
	}
	return NewToolResult(b.String())
}

// SWE100821: capture takes a photo from a single camera and returns the scene description.
func (t *CameraTool) capture(ctx context.Context, cameraName string) *ToolResult {
	desc, framePath, err := t.cameraSource.CaptureFrom(ctx, cameraName)
	if framePath != "" {
		defer os.Remove(framePath)
	}
	if err != nil {
		return ErrorResult(fmt.Sprintf("Capture failed: %v", err))
	}

	if desc == "" {
		return NewToolResult("Image captured but vision analysis unavailable (Ollama may be busy). The frame was saved but could not be described.")
	}

	label := cameraName
	if label == "" {
		label = "default"
	}
	return &ToolResult{
		ForLLM:  fmt.Sprintf("[Camera: %s] %s", label, desc),
		ForUser: fmt.Sprintf("Camera (%s): %s", label, desc),
	}
}

// SWE100821: lookAll captures from every discovered camera and returns combined descriptions.
func (t *CameraTool) lookAll(ctx context.Context) *ToolResult {
	cams := t.cameraSource.ListCameras()
	if len(cams) == 0 {
		return ErrorResult("No cameras available.")
	}

	var b strings.Builder
	b.WriteString("Visual scan from all cameras:\n")
	successCount := 0

	for _, cam := range cams {
		desc, framePath, err := t.cameraSource.CaptureFrom(ctx, cam.Name)
		if framePath != "" {
			os.Remove(framePath)
		}
		if err != nil {
			b.WriteString(fmt.Sprintf("  %s: FAILED (%v)\n", cam.Name, err))
			continue
		}
		if desc == "" {
			b.WriteString(fmt.Sprintf("  %s: captured but vision unavailable\n", cam.Name))
			continue
		}
		b.WriteString(fmt.Sprintf("  %s: %s\n", cam.Name, desc))
		successCount++
	}

	if successCount == 0 {
		return ErrorResult("All camera captures failed or vision analysis unavailable.")
	}

	result := b.String()
	return &ToolResult{
		ForLLM:  result,
		ForUser: result,
	}
}
