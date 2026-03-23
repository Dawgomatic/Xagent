// SWE100821: Camera sensor source — captures frames from USB webcams, CSI cameras,
// and phone cameras. Optionally runs captured frames through Ollama vision model
// for scene description, injecting ambient visual awareness into the agent's context.
// Captures are agent-driven (on-demand tool + event triggers) with a slow ambient fallback.

package sensors

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// CameraInfo describes a discovered camera device.
type CameraInfo struct {
	Path   string // /dev/video0, "phone:front", "phone:screen", etc.
	Name   string // human label
	Type   string // "v4l2", "csi", "phone-screen", "phone-front", "phone-rear"
	Facing int    // Android camera facing: 0=rear, 1=front (-1 for non-phone)
	Width  int
	Height int
}

// CameraSource captures frames from cameras and optionally describes them with vision.
type CameraSource struct {
	cameras          []CameraInfo
	workspace        string
	ollamaURL        string
	visionModel      string
	adbPath          string
	adbSerial        string
	interval         time.Duration
	mu               sync.RWMutex
	lastScene        string    // cached scene description
	pollIndex        int       // SWE100821: rotate through cameras across polls
	lastEventCapture time.Time // SWE100821: cooldown gate for event-triggered captures
	httpClient       *http.Client
}

// CameraSourceConfig configures the camera perception source.
type CameraSourceConfig struct {
	Workspace   string
	OllamaURL   string // default: http://localhost:11434
	VisionModel string // default: moondream
	ADBPath     string
	ADBSerial   string
	Interval    time.Duration // default: 2 minutes
}

// NewCameraSource creates a camera sensor source.
func NewCameraSource(cfg CameraSourceConfig) *CameraSource {
	if cfg.OllamaURL == "" {
		cfg.OllamaURL = "http://localhost:11434"
	}
	if cfg.VisionModel == "" {
		cfg.VisionModel = "moondream"
	}
	if cfg.ADBPath == "" {
		cfg.ADBPath = "adb"
	}
	// SWE100821: 60 min ambient fallback — primary captures are now agent-driven
	// (on-demand tool calls) and event-triggered (motion/light/screen changes).
	// This slow timer is just a safety net so the agent never goes fully blind.
	if cfg.Interval <= 0 {
		cfg.Interval = 60 * time.Minute
	}
	if cfg.Workspace == "" {
		cfg.Workspace = "/tmp"
	}
	return &CameraSource{
		workspace:   cfg.Workspace,
		ollamaURL:   cfg.OllamaURL,
		visionModel: cfg.VisionModel,
		adbPath:     cfg.ADBPath,
		adbSerial:   cfg.ADBSerial,
		interval:    cfg.Interval,
		httpClient:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (cs *CameraSource) Name() string { return "camera" }

func (cs *CameraSource) Interval() time.Duration { return cs.interval }

func (cs *CameraSource) Available() bool {
	// Available if we have any cameras OR a phone with ADB
	cs.discoverCameras()
	return len(cs.cameras) > 0
}

func (cs *CameraSource) Poll(ctx context.Context) ([]SensorReading, error) {
	if len(cs.cameras) == 0 {
		cs.discoverCameras()
	}
	if len(cs.cameras) == 0 {
		return nil, fmt.Errorf("no cameras available")
	}

	now := time.Now()
	var readings []SensorReading

	// SWE100821: Rotate through cameras across polls — each poll captures from the next
	// camera in the list. This spreads GPU load and gives the agent views from all cameras.
	cs.mu.Lock()
	idx := cs.pollIndex % len(cs.cameras)
	cs.pollIndex++
	cs.mu.Unlock()

	cam := cs.cameras[idx]
	framePath := filepath.Join(cs.workspace, fmt.Sprintf(".perception_frame_%s.jpg", cam.Name))
	// SWE100821: Reuse captureCamera dispatcher
	captureErr := cs.captureCamera(ctx, cam, framePath)

	// SWE100821: Propagate capture errors so monitor tracks failures correctly.
	// Previously returned nil error, which made the monitor think the poll succeeded.
	if captureErr != nil {
		return nil, fmt.Errorf("camera %s capture: %w", cam.Name, captureErr)
	}

	// Report which camera captured
	readings = append(readings, SensorReading{
		Name: "camera/" + cam.Name, Value: 1, Unit: "active", Timestamp: now,
	})

	// Run vision analysis if Ollama is available
	desc, err := cs.analyzeFrame(ctx, framePath)
	if err != nil {
		logger.WarnCF("perception", "Vision analysis failed", map[string]interface{}{
			"camera": cam.Name, "error": err.Error(),
		})
	} else if desc != "" {
		cs.mu.Lock()
		cs.lastScene = desc
		cs.mu.Unlock()

		readings = append(readings, SensorReading{
			Name: "camera/scene/" + cam.Name, Value: 0, Unit: desc, Timestamp: now,
		})
		logger.InfoCF("perception", "Scene described", map[string]interface{}{
			"camera": cam.Name, "len": len(desc),
		})
	}

	os.Remove(framePath)

	return readings, nil
}

func (cs *CameraSource) discoverCameras() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.cameras = nil

	// 1. V4L2 USB/CSI cameras
	matches, _ := filepath.Glob("/dev/video*")
	for _, path := range matches {
		name := filepath.Base(path)
		// Check if it's a capture device (not metadata)
		if cs.isCaptureDev(path) {
			cs.cameras = append(cs.cameras, CameraInfo{
				Path: path, Name: name, Type: "v4l2",
			})
			logger.InfoCF("perception", "Discovered camera", map[string]interface{}{
				"path": path, "type": "v4l2",
			})
		}
	}

	// 2. Tegra CSI cameras (NVIDIA-specific paths)
	csiPaths := []string{
		"/dev/video0", // Often CSI on Jetson
	}
	for _, p := range csiPaths {
		// Skip if already found as v4l2
		found := false
		for _, c := range cs.cameras {
			if c.Path == p {
				found = true
				break
			}
		}
		if found {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			cs.cameras = append(cs.cameras, CameraInfo{
				Path: p, Name: "csi-" + filepath.Base(p), Type: "csi",
			})
		}
	}

	// 3. Phone cameras (if ADB available) — discover front + rear via camera IDs
	adbPath, err := exec.LookPath(cs.adbPath)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, adbPath, "get-state").CombinedOutput()
		if err == nil && strings.TrimSpace(string(out)) == "device" {
			// Phone screen capture is always available
			cs.cameras = append(cs.cameras, CameraInfo{
				Name: "phone-screen", Type: "phone-screen", Facing: -1,
			})

			// SWE100821: Query available camera IDs to discover front + rear
			cameraIDs := cs.discoverPhoneCameraIDs(ctx, adbPath)
			for _, cam := range cameraIDs {
				cs.cameras = append(cs.cameras, cam)
			}

			if len(cameraIDs) == 0 {
				// Fallback: assume standard rear (0) + front (1) if we can't query
				cs.cameras = append(cs.cameras, CameraInfo{
					Name: "phone-rear", Type: "phone-rear", Facing: 0,
				})
				cs.cameras = append(cs.cameras, CameraInfo{
					Name: "phone-front", Type: "phone-front", Facing: 1,
				})
			}

			var names []string
			for _, c := range cs.cameras {
				names = append(names, c.Name)
			}
			logger.InfoCF("perception", "Discovered phone cameras via ADB", map[string]interface{}{
				"cameras": names,
			})
		}
	}
}

// SWE100821: discoverPhoneCameraIDs queries the Android camera service for available camera IDs
// and their facing direction (front/rear). Uses dumpsys media.camera on Android 9+.
func (cs *CameraSource) discoverPhoneCameraIDs(ctx context.Context, adbPath string) []CameraInfo {
	args := []string{}
	if cs.adbSerial != "" {
		args = append(args, "-s", cs.adbSerial)
	}
	args = append(args, "shell", "dumpsys media.camera | grep -E 'Camera ID|Facing'")

	cmd := exec.CommandContext(ctx, adbPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	var cameras []CameraInfo
	lines := strings.Split(string(out), "\n")

	// Parse pairs of "Camera ID: X" and "Facing: BACK/FRONT"
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Look for camera ID lines
		if !strings.Contains(line, "Camera") || !strings.Contains(line, "ID") {
			continue
		}

		// Extract the ID number
		var camID string
		for _, part := range strings.Fields(line) {
			part = strings.TrimRight(part, ":,")
			if len(part) > 0 && part[0] >= '0' && part[0] <= '9' {
				camID = part
				break
			}
		}
		if camID == "" {
			continue
		}

		// Look ahead for facing direction
		facing := -1
		facingLabel := "unknown"
		for j := i + 1; j < len(lines) && j < i+5; j++ {
			fl := strings.ToLower(strings.TrimSpace(lines[j]))
			if strings.Contains(fl, "facing") {
				if strings.Contains(fl, "back") || strings.Contains(fl, "rear") {
					facing = 0
					facingLabel = "rear"
				} else if strings.Contains(fl, "front") {
					facing = 1
					facingLabel = "front"
				}
				break
			}
		}

		camType := "phone-rear"
		camName := "phone-rear-" + camID
		if facing == 1 {
			camType = "phone-front"
			camName = "phone-front-" + camID
		} else if facing == -1 {
			camName = "phone-cam-" + camID
		}

		cameras = append(cameras, CameraInfo{
			Path: camID, Name: camName, Type: camType, Facing: facing,
		})

		logger.InfoCF("perception", "Found phone camera", map[string]interface{}{
			"id": camID, "facing": facingLabel,
		})
	}

	return cameras
}

// isCaptureDev checks if a v4l2 device is a video capture device (not metadata-only).
func (cs *CameraSource) isCaptureDev(path string) bool {
	// Try v4l2-ctl, fall back to assuming capture if it exists
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "v4l2-ctl", "-d", path, "--all").CombinedOutput()
	if err != nil {
		// v4l2-ctl not installed — assume it's a capture device if it exists
		_, statErr := os.Stat(path)
		return statErr == nil
	}
	return strings.Contains(string(out), "Video Capture")
}

// captureV4L2 grabs a single frame from a V4L2 camera device using ffmpeg.
func (cs *CameraSource) captureV4L2(ctx context.Context, devPath, outPath string) error {
	captureCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// SWE100821: Try ffmpeg first, then fswebcam as fallback
	cmd := exec.CommandContext(captureCtx, "ffmpeg",
		"-f", "v4l2", "-i", devPath,
		"-frames:v", "1", "-y",
		"-loglevel", "error",
		outPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		// Fallback: try fswebcam
		cmd2 := exec.CommandContext(captureCtx, "fswebcam",
			"-d", devPath,
			"--no-banner",
			"-r", "640x480",
			outPath,
		)
		if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
			return fmt.Errorf("ffmpeg: %s; fswebcam: %s", string(out), string(out2))
		}
	}

	// Verify the file was created and has content
	info, err := os.Stat(outPath)
	if err != nil || info.Size() < 100 {
		return fmt.Errorf("captured frame too small or missing")
	}
	return nil
}

// capturePhoneScreen takes a screenshot of the phone screen via ADB.
// Reused: pkg/phone/adb.go Screenshot() pattern
func (cs *CameraSource) capturePhoneScreen(ctx context.Context, outPath string) error {
	captureCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	args := []string{}
	if cs.adbSerial != "" {
		args = append(args, "-s", cs.adbSerial)
	}
	args = append(args, "exec-out", "screencap", "-p")

	cmd := exec.CommandContext(captureCtx, cs.adbPath, args...)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("phone screenshot: %w", err)
	}
	if len(out) < 100 {
		return fmt.Errorf("phone screenshot too small (%d bytes)", len(out))
	}
	return os.WriteFile(outPath, out, 0644)
}

// SWE100821: capturePhoneHWCamera captures from front (1) or rear (0) camera.
// Tries multiple methods in order of reliability:
//   1. cmd media.camera capture — Android 12+ shell-level capture, no UI
//   2. am start with CAMERA_FACING intent — opens camera app, simulates shutter
//   3. Fallback to phone screen screenshot
func (cs *CameraSource) capturePhoneHWCamera(ctx context.Context, facing int, outPath string) error {
	captureCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	remotePath := "/sdcard/DCIM/perception_capture.jpg"
	facingLabel := "rear"
	if facing == 1 {
		facingLabel = "front"
	}

	shell := func(cmd string) (string, error) {
		args := []string{}
		if cs.adbSerial != "" {
			args = append(args, "-s", cs.adbSerial)
		}
		args = append(args, "shell", cmd)
		c := exec.CommandContext(captureCtx, cs.adbPath, args...)
		out, err := c.CombinedOutput()
		return string(out), err
	}

	pull := func(remote, local string) error {
		args := []string{}
		if cs.adbSerial != "" {
			args = append(args, "-s", cs.adbSerial)
		}
		args = append(args, "pull", remote, local)
		c := exec.CommandContext(captureCtx, cs.adbPath, args...)
		_, err := c.CombinedOutput()
		return err
	}

	// Clean up any previous capture
	shell("rm -f " + remotePath)

	// Method 1: Android Camera2 test utility (works without UI on many devices)
	// Uses the hidden cmd interface to capture directly from camera hardware
	if _, err := shell(fmt.Sprintf(
		"cmd media.camera capture --camera-id %d --output %s 2>/dev/null",
		facing, remotePath,
	)); err == nil {
		if err := pull(remotePath, outPath); err == nil {
			if info, _ := os.Stat(outPath); info != nil && info.Size() > 1000 {
				shell("rm -f " + remotePath)
				logger.InfoCF("perception", "Camera capture via cmd media.camera", map[string]interface{}{
					"facing": facingLabel, "size": info.Size(),
				})
				return nil
			}
		}
	}

	// Method 2: Use Android intent to open camera, simulate shutter key, pull latest photo
	camFacingExtra := fmt.Sprintf("--ei android.intent.extras.CAMERA_FACING %d", facing)
	// Try multiple camera app component names (varies by manufacturer)
	cameraApps := []string{
		"com.android.camera2/com.android.camera.CameraLauncher",
		"com.android.camera/.CameraActivity",
		"com.sec.android.app.camera/.Camera", // Samsung
		"com.google.android.GoogleCamera/com.android.camera.CameraLauncher",
	}

	for _, app := range cameraApps {
		captureCmd := fmt.Sprintf(
			"am start -a android.media.action.IMAGE_CAPTURE %s -n %s 2>/dev/null && "+
				"sleep 3 && "+
				"input keyevent 27 && "+ // KEYCODE_CAMERA (shutter)
				"sleep 2 && "+
				"ls -t /sdcard/DCIM/Camera/*.jpg /sdcard/DCIM/*.jpg 2>/dev/null | head -1",
			camFacingExtra, app,
		)

		latestFile, err := shell(captureCmd)
		if err != nil {
			continue
		}
		latestFile = strings.TrimSpace(latestFile)
		if latestFile == "" {
			shell("input keyevent 4") // Close camera app
			continue
		}

		// Copy latest photo to our staging path, then pull
		shell(fmt.Sprintf("cp '%s' %s", latestFile, remotePath))
		shell("input keyevent 4") // Close camera app

		if err := pull(remotePath, outPath); err == nil {
			if info, _ := os.Stat(outPath); info != nil && info.Size() > 1000 {
				shell("rm -f " + remotePath)
				logger.InfoCF("perception", "Camera capture via intent", map[string]interface{}{
					"facing": facingLabel, "app": app, "size": info.Size(),
				})
				return nil
			}
		}
	}

	// Method 3: Fallback to phone screen
	logger.WarnCF("perception", "HW camera capture failed, falling back to screen", map[string]interface{}{
		"facing": facingLabel,
	})
	return cs.capturePhoneScreen(ctx, outPath)
}

// analyzeFrame sends the captured frame to Ollama vision for scene description.
func (cs *CameraSource) analyzeFrame(ctx context.Context, framePath string) (string, error) {
	data, err := os.ReadFile(framePath)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	// Compact prompt to minimize inference time on embedded
	reqBody := map[string]interface{}{
		"model":  cs.visionModel,
		"prompt": "Describe what you see in this image in 1-2 sentences. Focus on: people present, notable objects, environment (indoor/outdoor), lighting, any text visible.",
		"images": []string{encoded},
		"stream": false,
		"options": map[string]interface{}{
			"num_predict": 100, // Keep description short
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	analysisCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(analysisCtx, "POST",
		cs.ollamaURL+"/api/generate",
		strings.NewReader(string(jsonData)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := cs.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama vision unavailable: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("vision API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	// Truncate long descriptions
	desc := strings.TrimSpace(result.Response)
	if len(desc) > 300 {
		desc = desc[:300] + "..."
	}
	return desc, nil
}

// LastScene returns the most recent scene description (for direct access).
func (cs *CameraSource) LastScene() string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.lastScene
}

// SWE100821: ListCameras returns all discovered cameras for the agent tool.
func (cs *CameraSource) ListCameras() []CameraInfo {
	if len(cs.cameras) == 0 {
		cs.discoverCameras()
	}
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	out := make([]CameraInfo, len(cs.cameras))
	copy(out, cs.cameras)
	return out
}

// SWE100821: CaptureFrom captures a frame from a named camera (or first match) and
// runs vision analysis. Returns (description, framePath, error). The caller should
// clean up framePath if non-empty.
func (cs *CameraSource) CaptureFrom(ctx context.Context, cameraName string) (string, string, error) {
	if len(cs.cameras) == 0 {
		cs.discoverCameras()
	}
	cs.mu.RLock()
	cams := cs.cameras
	cs.mu.RUnlock()

	if len(cams) == 0 {
		return "", "", fmt.Errorf("no cameras available")
	}

	cam, err := cs.findCamera(cameraName, cams)
	if err != nil {
		return "", "", err
	}

	framePath := filepath.Join(cs.workspace, fmt.Sprintf(".capture_%s_%d.jpg", cam.Name, time.Now().UnixMilli()))
	if captureErr := cs.captureCamera(ctx, cam, framePath); captureErr != nil {
		return "", "", fmt.Errorf("capture %s: %w", cam.Name, captureErr)
	}

	desc, analysisErr := cs.analyzeFrame(ctx, framePath)
	if analysisErr != nil {
		logger.WarnCF("perception", "Vision analysis failed on demand capture", map[string]interface{}{
			"camera": cam.Name, "error": analysisErr.Error(),
		})
		return "", framePath, nil
	}

	cs.mu.Lock()
	cs.lastScene = desc
	cs.mu.Unlock()

	return desc, framePath, nil
}

// SWE100821: CaptureDefault picks the best available camera (rear > screen > first)
// and captures from it.
func (cs *CameraSource) CaptureDefault(ctx context.Context) (string, string, error) {
	return cs.CaptureFrom(ctx, "")
}

// SWE100821: EventCapture is called by the monitor when sensor events warrant a capture.
// Respects a 5-minute cooldown to prevent rapid-fire captures. Returns readings to
// inject back into the monitor's buffer.
func (cs *CameraSource) EventCapture(ctx context.Context) ([]SensorReading, bool) {
	cs.mu.Lock()
	if time.Since(cs.lastEventCapture) < 5*time.Minute {
		cs.mu.Unlock()
		return nil, false
	}
	cs.lastEventCapture = time.Now()
	cs.mu.Unlock()

	desc, framePath, err := cs.CaptureDefault(ctx)
	if framePath != "" {
		os.Remove(framePath)
	}
	if err != nil {
		logger.WarnCF("perception", "Event-triggered capture failed", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, false
	}

	now := time.Now()
	var readings []SensorReading
	readings = append(readings, SensorReading{
		Name: "camera/event-capture", Value: 1, Unit: "active", Timestamp: now,
	})
	if desc != "" {
		readings = append(readings, SensorReading{
			Name: "camera/scene/event", Value: 0, Unit: desc, Timestamp: now,
		})
	}
	logger.InfoCF("perception", "Event-triggered capture succeeded", map[string]interface{}{
		"desc_len": len(desc),
	})
	return readings, true
}

// SWE100821: findCamera resolves a camera by name, or picks the best default.
func (cs *CameraSource) findCamera(name string, cams []CameraInfo) (CameraInfo, error) {
	if name != "" {
		for _, c := range cams {
			if c.Name == name || c.Type == name {
				return c, nil
			}
		}
		return CameraInfo{}, fmt.Errorf("camera %q not found; available: %s", name, cs.cameraNames(cams))
	}
	// Default priority: rear > screen > first available
	for _, pref := range []string{"phone-rear", "phone-screen"} {
		for _, c := range cams {
			if c.Type == pref {
				return c, nil
			}
		}
	}
	return cams[0], nil
}

// SWE100821: captureCamera dispatches to the right capture method based on camera type.
func (cs *CameraSource) captureCamera(ctx context.Context, cam CameraInfo, framePath string) error {
	switch cam.Type {
	case "v4l2", "csi":
		return cs.captureV4L2(ctx, cam.Path, framePath)
	case "phone-screen":
		return cs.capturePhoneScreen(ctx, framePath)
	case "phone-rear":
		return cs.capturePhoneHWCamera(ctx, 0, framePath)
	case "phone-front":
		return cs.capturePhoneHWCamera(ctx, 1, framePath)
	default:
		return fmt.Errorf("unknown camera type: %s", cam.Type)
	}
}

func (cs *CameraSource) cameraNames(cams []CameraInfo) string {
	var names []string
	for _, c := range cams {
		names = append(names, c.Name)
	}
	return strings.Join(names, ", ")
}
