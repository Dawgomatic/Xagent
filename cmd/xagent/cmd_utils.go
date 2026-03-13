package main

import (
	"io"
	"os"
	"path/filepath"

	"github.com/Dawgomatic/Xagent/pkg/config"
)

func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".xagent", "config.json")
}

func loadConfig() (*config.Config, error) {
	return config.LoadConfig(getConfigPath())
}
