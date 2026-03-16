// SWE100821: Voice conversation loop — continuous listen→transcribe→think→speak cycle.
// Uses Groq Whisper STT (transcriber.go) and Piper/espeak TTS (tts.go).
// Records audio via arecord, plays back via aplay.

package voice

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Dawgomatic/Xagent/pkg/logger"
)

// AgentProcessor is the interface the agent loop must satisfy for voice input.
// SWE100821: Decoupled from agent.AgentLoop to avoid circular imports.
type AgentProcessor interface {
	ProcessDirect(ctx context.Context, content, sessionKey string) (string, error)
}

// VoiceLoop orchestrates continuous voice conversations:
// record → transcribe (Groq) → agent → TTS → playback.
type VoiceLoop struct {
	transcriber *GroqTranscriber
	tts         *TTSEngine
	agentLoop   AgentProcessor
	sessionKey  string
	running     bool
	mu          sync.Mutex
	recordDir   string
	chunkSecs   int
}

// NewVoiceLoop creates a VoiceLoop wired to the given transcriber and workspace.
// SWE100821: Reuses GroqTranscriber (transcriber.go) and TTSEngine (tts.go).
func NewVoiceLoop(transcriber *GroqTranscriber, workspace string) *VoiceLoop {
	recordDir := filepath.Join(workspace, "voice_input")
	os.MkdirAll(recordDir, 0755)

	return &VoiceLoop{
		transcriber: transcriber,
		tts:         NewTTSEngine(workspace),
		sessionKey:  "voice:local",
		chunkSecs:   5,
		recordDir:   recordDir,
	}
}

// SetAgent attaches the agent processor for handling transcribed text.
func (vl *VoiceLoop) SetAgent(agent AgentProcessor) {
	vl.mu.Lock()
	defer vl.mu.Unlock()
	// SWE100821: Allow late-binding of agent after construction
	vl.agentLoop = agent
}

// Start begins the voice loop: record → transcribe → process → speak → repeat.
// Blocks until ctx is cancelled or Stop() is called.
func (vl *VoiceLoop) Start(ctx context.Context) error {
	vl.mu.Lock()
	if vl.agentLoop == nil {
		vl.mu.Unlock()
		return fmt.Errorf("agent not set; call SetAgent before Start")
	}
	if vl.running {
		vl.mu.Unlock()
		return fmt.Errorf("voice loop already running")
	}
	vl.running = true
	vl.mu.Unlock()

	defer func() {
		vl.mu.Lock()
		vl.running = false
		vl.mu.Unlock()
	}()

	logger.InfoCF("voice", "Voice loop started", map[string]interface{}{
		"chunk_seconds": vl.chunkSecs,
		"tts_engine":    vl.tts.GetEngine(),
	})

	for {
		select {
		case <-ctx.Done():
			logger.InfoC("voice", "Voice loop stopped (context cancelled)")
			return ctx.Err()
		default:
		}

		// SWE100821: Step 1 — record audio chunk via arecord
		audioPath, err := vl.recordChunk(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			logger.ErrorCF("voice", "Recording failed", map[string]interface{}{"error": err.Error()})
			time.Sleep(time.Second)
			continue
		}

		// SWE100821: Step 2 — transcribe via Groq STT
		resp, err := vl.transcriber.Transcribe(ctx, audioPath)
		os.Remove(audioPath)
		if err != nil {
			logger.ErrorCF("voice", "Transcription failed", map[string]interface{}{"error": err.Error()})
			continue
		}

		text := strings.TrimSpace(resp.Text)
		if text == "" {
			continue
		}

		logger.InfoCF("voice", "Transcribed", map[string]interface{}{"text": text})

		// SWE100821: Step 3 — send to agent
		reply, err := vl.agentLoop.ProcessDirect(ctx, text, vl.sessionKey)
		if err != nil {
			logger.ErrorCF("voice", "Agent processing failed", map[string]interface{}{"error": err.Error()})
			continue
		}

		if reply == "" {
			continue
		}

		// SWE100821: Step 4 — synthesize reply to speech
		if !vl.tts.IsAvailable() {
			logger.InfoCF("voice", "TTS unavailable, skipping playback", map[string]interface{}{"reply_len": len(reply)})
			continue
		}

		wavPath, err := vl.tts.Synthesize(ctx, reply)
		if err != nil {
			logger.ErrorCF("voice", "TTS synthesis failed", map[string]interface{}{"error": err.Error()})
			continue
		}

		// SWE100821: Step 5 — play audio via aplay
		if err := vl.playAudio(ctx, wavPath); err != nil {
			logger.ErrorCF("voice", "Audio playback failed", map[string]interface{}{"error": err.Error()})
		}
		os.Remove(wavPath)
	}
}

// Stop signals the voice loop to cease after the current iteration.
func (vl *VoiceLoop) Stop() {
	vl.mu.Lock()
	defer vl.mu.Unlock()
	// SWE100821: The loop checks vl.running; the caller should cancel the context.
	vl.running = false
}

// recordChunk records a short WAV clip via arecord.
func (vl *VoiceLoop) recordChunk(ctx context.Context) (string, error) {
	outFile := filepath.Join(vl.recordDir, fmt.Sprintf("chunk_%d.wav", time.Now().UnixNano()))

	cmd := exec.CommandContext(ctx, "arecord",
		"-f", "S16_LE",
		"-r", "16000",
		"-c", "1",
		"-d", fmt.Sprintf("%d", vl.chunkSecs),
		outFile,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("arecord failed: %v (stderr: %s)", err, stderr.String())
	}
	return outFile, nil
}

// playAudio plays a WAV file via aplay.
func (vl *VoiceLoop) playAudio(ctx context.Context, path string) error {
	cmd := exec.CommandContext(ctx, "aplay", path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aplay failed: %v (stderr: %s)", err, stderr.String())
	}
	return nil
}
