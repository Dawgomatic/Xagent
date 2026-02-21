// SWE100821: Text-to-speech via Piper for voice conversation loop.
// Completes the voice loop: voice input (Whisper) → agent → TTS output (Piper).
// Piper runs locally for privacy — no cloud dependency.
// Falls back to espeak if Piper is unavailable.

package voice

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// TTSEngine provides text-to-speech synthesis.
type TTSEngine struct {
	engine     string // "piper", "espeak", "none"
	piperPath  string
	modelPath  string
	outputDir  string
	sampleRate int
	mu         sync.Mutex
}

// NewTTSEngine creates a TTS engine, auto-detecting available backends.
func NewTTSEngine(workspace string) *TTSEngine {
	outputDir := filepath.Join(workspace, "voice_output")
	os.MkdirAll(outputDir, 0755)

	tts := &TTSEngine{
		engine:     "none",
		outputDir:  outputDir,
		sampleRate: 22050,
	}

	// SWE100821: Auto-detect TTS backend
	if path, err := exec.LookPath("piper"); err == nil {
		tts.engine = "piper"
		tts.piperPath = path
		logger.InfoCF("voice", "TTS engine: piper", nil)
	} else if _, err := exec.LookPath("espeak-ng"); err == nil {
		tts.engine = "espeak"
		logger.InfoCF("voice", "TTS engine: espeak-ng (fallback)", nil)
	} else if _, err := exec.LookPath("espeak"); err == nil {
		tts.engine = "espeak"
		logger.InfoCF("voice", "TTS engine: espeak (fallback)", nil)
	} else {
		logger.InfoCF("voice", "No TTS engine found (voice output disabled)", nil)
	}

	return tts
}

// IsAvailable returns whether TTS is available.
func (t *TTSEngine) IsAvailable() bool {
	return t.engine != "none"
}

// SetModel sets the Piper voice model path.
func (t *TTSEngine) SetModel(modelPath string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.modelPath = modelPath
}

// Synthesize converts text to speech and returns the path to the WAV file.
func (t *TTSEngine) Synthesize(ctx context.Context, text string) (audioPath string, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.engine == "none" {
		return "", fmt.Errorf("no TTS engine available")
	}

	// Generate unique output filename
	outputFile := filepath.Join(t.outputDir, fmt.Sprintf("tts_%d.wav", time.Now().UnixNano()))

	switch t.engine {
	case "piper":
		err = t.synthesizePiper(ctx, text, outputFile)
	case "espeak":
		err = t.synthesizeEspeak(ctx, text, outputFile)
	default:
		return "", fmt.Errorf("unknown TTS engine: %s", t.engine)
	}

	if err != nil {
		return "", err
	}

	return outputFile, nil
}

// SynthesizeToBytes converts text to speech and returns raw audio bytes.
func (t *TTSEngine) SynthesizeToBytes(ctx context.Context, text string) ([]byte, error) {
	path, err := t.Synthesize(ctx, text)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)

	return os.ReadFile(path)
}

// CleanupOldFiles removes TTS output files older than maxAge.
func (t *TTSEngine) CleanupOldFiles(maxAge time.Duration) int {
	entries, err := os.ReadDir(t.outputDir)
	if err != nil {
		return 0
	}

	cutoff := time.Now().Add(-maxAge)
	cleaned := 0
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(t.outputDir, e.Name()))
			cleaned++
		}
	}
	return cleaned
}

func (t *TTSEngine) synthesizePiper(ctx context.Context, text, outputFile string) error {
	args := []string{"--output_file", outputFile}
	if t.modelPath != "" {
		args = append(args, "--model", t.modelPath)
	}

	cmd := exec.CommandContext(ctx, t.piperPath, args...)
	cmd.Stdin = bytes.NewReader([]byte(text))

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("piper TTS failed: %v (stderr: %s)", err, stderr.String())
	}

	return nil
}

func (t *TTSEngine) synthesizeEspeak(ctx context.Context, text, outputFile string) error {
	espeakBin := "espeak-ng"
	if _, err := exec.LookPath(espeakBin); err != nil {
		espeakBin = "espeak"
	}

	cmd := exec.CommandContext(ctx, espeakBin, "-w", outputFile, text)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("espeak TTS failed: %v (stderr: %s)", err, stderr.String())
	}

	return nil
}

// GetEngine returns the name of the active TTS engine.
func (t *TTSEngine) GetEngine() string {
	return t.engine
}
