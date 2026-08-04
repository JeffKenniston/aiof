package config

import (
	"os"
	"path/filepath"
)

func GetHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return filepath.Join(home, ".aiof")
}

func GetConfigDir() string {
	return filepath.Join(GetHomeDir(), "config")
}

func GetCacheDir() string {
	return filepath.Join(GetHomeDir(), "cache")
}

func GetLogsDir() string {
	return filepath.Join(GetHomeDir(), "logs")
}

func GetBrainDir() string {
	return filepath.Join(GetHomeDir(), "brain")
}

func GetSandboxDir() string {
	return filepath.Join(GetHomeDir(), "sandbox")
}

func GetTempDir() string {
	return filepath.Join(GetHomeDir(), "temp")
}

func GetModelsDir() string {
	return filepath.Join(GetHomeDir(), "models")
}

func EnsureDirs() error {
	dirs := []string{
		GetConfigDir(),
		GetCacheDir(),
		GetLogsDir(),
		GetBrainDir(),
		GetSandboxDir(),
		GetTempDir(),
		GetModelsDir(),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}
