package upgrade

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// SWE100821: RL checkpoint metadata returned by the RL server.
type RLCheckpoint struct {
	Path      string  `json:"path"`
	Step      int     `json:"step"`
	AvgReward float64 `json:"avg_reward"`
}

const (
	githubAPIBase = "https://api.github.com/repos/Dawgomatic/Xagent"
	httpTimeout   = 30 * time.Second
)

type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type Release struct {
	TagName    string         `json:"tag_name"`
	Name       string         `json:"name"`
	Draft      bool           `json:"draft"`
	Prerelease bool           `json:"prerelease"`
	Assets     []ReleaseAsset `json:"assets"`
	Body       string         `json:"body"`
	PublishedAt string        `json:"published_at"`
}

type CheckResult struct {
	CurrentVersion string
	LatestVersion  string
	UpdateAvail    bool
	Release        *Release
}

func CheckLatest(currentVersion string) (*CheckResult, error) {
	client := &http.Client{Timeout: httpTimeout}
	url := githubAPIBase + "/releases/latest"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "xagent-upgrade/"+currentVersion)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return &CheckResult{
			CurrentVersion: currentVersion,
			LatestVersion:  "",
			UpdateAvail:    false,
		}, nil
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("parsing release: %w", err)
	}

	latest := strings.TrimPrefix(rel.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")

	return &CheckResult{
		CurrentVersion: current,
		LatestVersion:  latest,
		UpdateAvail:    latest != current && current != "dev" && latest != "",
		Release:        &rel,
	}, nil
}

func binaryAssetName() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	name := fmt.Sprintf("xagent-%s-%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

func findAsset(rel *Release) *ReleaseAsset {
	target := binaryAssetName()
	for _, a := range rel.Assets {
		if a.Name == target {
			return &a
		}
	}
	return nil
}

func findChecksumAsset(rel *Release) *ReleaseAsset {
	for _, a := range rel.Assets {
		if strings.Contains(a.Name, "sha256") {
			return &a
		}
	}
	return nil
}

func DownloadAndReplace(rel *Release, currentVersion string) error {
	asset := findAsset(rel)
	if asset == nil {
		return fmt.Errorf("no binary found for %s/%s in release %s", runtime.GOOS, runtime.GOARCH, rel.TagName)
	}

	currentBin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}
	currentBin, err = filepath.EvalSymlinks(currentBin)
	if err != nil {
		return fmt.Errorf("resolving symlinks: %w", err)
	}

	fmt.Printf("Downloading %s (%d bytes)...\n", asset.Name, asset.Size)

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(asset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("downloading binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	tmpFile := currentBin + ".new"
	f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(f, hasher), resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("writing binary: %w", err)
	}
	downloadSum := hex.EncodeToString(hasher.Sum(nil))

	fmt.Printf("Downloaded %d bytes (sha256: %s)\n", written, downloadSum[:16]+"...")

	checksumAsset := findChecksumAsset(rel)
	if checksumAsset != nil {
		if err := verifyChecksum(client, checksumAsset, asset.Name, downloadSum); err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		fmt.Println("Checksum verified")
	}

	backupFile := currentBin + ".bak"
	if err := os.Rename(currentBin, backupFile); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("backing up current binary: %w", err)
	}

	if err := os.Rename(tmpFile, currentBin); err != nil {
		os.Rename(backupFile, currentBin)
		return fmt.Errorf("replacing binary: %w", err)
	}

	os.Remove(backupFile)

	fmt.Printf("Upgraded: %s -> %s\n", currentVersion, rel.TagName)
	return nil
}

func verifyChecksum(client *http.Client, checksumAsset *ReleaseAsset, binaryName, actualSum string) error {
	resp, err := client.Get(checksumAsset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("fetching checksums: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading checksums: %w", err)
	}

	for _, line := range strings.Split(string(body), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 {
			name := filepath.Base(parts[1])
			if name == binaryName {
				if parts[0] != actualSum {
					return fmt.Errorf("expected %s, got %s", parts[0], actualSum)
				}
				return nil
			}
		}
	}

	return fmt.Errorf("no checksum entry for %s", binaryName)
}

func UpgradeModel(model string) error {
	// SWE100821: Check for RL checkpoint first, fall back to standard Ollama pull.
	if model == "" || model == "xagent-rl" {
		rlURL := os.Getenv("RL_SERVER_URL")
		if rlURL != "" {
			workspace := os.Getenv("XAGENT_WORKSPACE")
			if workspace == "" {
				workspace = filepath.Join(os.Getenv("HOME"), ".xagent")
			}
			newModel, err := UpgradeFromCheckpoint(rlURL, workspace)
			if err != nil {
				log.Printf("RL checkpoint upgrade failed, falling back: %v", err)
			} else if newModel != "" {
				fmt.Printf("RL model deployed: %s\n", newModel)
				return nil
			}
		}
	}

	if model == "" {
		modelFile := filepath.Join(os.Getenv("HOME"), ".ollama_model")
		data, err := os.ReadFile(modelFile)
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "MODEL=") {
					model = strings.TrimPrefix(line, "MODEL=")
					break
				}
			}
		}
	}

	if model == "" {
		model = "qwen2.5:1.5b"
	}

	fmt.Printf("Pulling latest model: %s\n", model)
	cmd := exec.Command("ollama", "pull", model)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ollama pull %s: %w", model, err)
	}
	fmt.Println("Model updated")
	return nil
}

// UpgradeFromCheckpoint loads an RL-trained checkpoint and creates an Ollama model from it.
// SWE100821: Closes the RL loop — training data → checkpoint → deployable model.
func UpgradeFromCheckpoint(rlServerURL string, workspace string) (string, error) {
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Get(rlServerURL + "/checkpoint/latest")
	if err != nil {
		return "", fmt.Errorf("fetching RL checkpoint: %w", err)
	}
	defer resp.Body.Close()

	// SWE100821: No checkpoint available is not an error — just nothing to upgrade.
	if resp.StatusCode == 404 {
		return "", nil
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("RL server returned HTTP %d", resp.StatusCode)
	}

	var ckpt RLCheckpoint
	if err := json.NewDecoder(resp.Body).Decode(&ckpt); err != nil {
		return "", fmt.Errorf("parsing checkpoint metadata: %w", err)
	}
	if ckpt.Path == "" {
		return "", nil
	}

	// SWE100821: Build Ollama Modelfile from RL checkpoint.
	modelfileContent := fmt.Sprintf("FROM %s\nSYSTEM \"Xagent RL-trained model (step %d, reward %.3f)\"", ckpt.Path, ckpt.Step, ckpt.AvgReward)

	modelfileDir := filepath.Join(workspace, "rl")
	if err := os.MkdirAll(modelfileDir, 0755); err != nil {
		return "", fmt.Errorf("creating rl dir: %w", err)
	}
	modelfilePath := filepath.Join(modelfileDir, "Modelfile")
	if err := os.WriteFile(modelfilePath, []byte(modelfileContent), 0644); err != nil {
		return "", fmt.Errorf("writing Modelfile: %w", err)
	}

	// SWE100821: Shell out to ollama create.
	modelName := "xagent-rl"
	cmd := exec.Command("ollama", "create", modelName, "-f", modelfilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ollama create %s: %w", modelName, err)
	}

	log.Printf("RL upgrade complete: model=%s step=%d avg_reward=%.3f", modelName, ckpt.Step, ckpt.AvgReward)
	return modelName, nil
}

func UpgradeSkills(installDir string) error {
	skillsDir := filepath.Join(installDir, "openclaw-skills")
	if _, err := os.Stat(filepath.Join(skillsDir, ".git")); os.IsNotExist(err) {
		fmt.Println("OpenClaw skills not cloned, skipping skills update")
		return nil
	}

	fmt.Println("Updating OpenClaw skills archive...")
	cmd := exec.Command("git", "-C", skillsDir, "pull", "--ff-only")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git pull skills: %w", err)
	}
	fmt.Println("Skills archive updated")
	return nil
}
