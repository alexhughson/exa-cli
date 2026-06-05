package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	EnvAPIKey     = "EXA_API_KEY"
	EnvConfigPath = "EXA_CLI_CONFIG"
)

var ErrHomeNotSet = errors.New("HOME is not set")

type LookupEnv func(string) (string, bool)

type File struct {
	APIKey string `json:"api_key"`
}

type KeySource struct {
	Key    string
	Source string
}

func LoadAPIKey(lookup LookupEnv) (KeySource, error) {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	if key, ok := lookup(EnvAPIKey); ok && strings.TrimSpace(key) != "" {
		return KeySource{Key: strings.TrimSpace(key), Source: EnvAPIKey}, nil
	}

	path, err := Path(lookup)
	if err != nil {
		if errors.Is(err, ErrHomeNotSet) {
			return KeySource{}, nil
		}
		return KeySource{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return KeySource{}, nil
		}
		return KeySource{}, fmt.Errorf("read config: %w", err)
	}

	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return KeySource{}, fmt.Errorf("parse config: %w", err)
	}
	if strings.TrimSpace(file.APIKey) == "" {
		return KeySource{}, nil
	}
	return KeySource{Key: strings.TrimSpace(file.APIKey), Source: path}, nil
}

func SaveAPIKey(apiKey string, lookup LookupEnv) (string, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", errors.New("api key is empty")
	}

	path, err := Path(lookup)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}

	data, err := json.MarshalIndent(File{APIKey: apiKey}, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return "", fmt.Errorf("set config permissions: %w", err)
	}
	return path, nil
}

func Logout(lookup LookupEnv) (string, error) {
	path, err := Path(lookup)
	if err != nil {
		return "", err
	}
	err = os.Remove(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return path, fmt.Errorf("remove config: %w", err)
	}
	return path, nil
}

func Path(lookup LookupEnv) (string, error) {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	if path, ok := lookup(EnvConfigPath); ok && strings.TrimSpace(path) != "" {
		return strings.TrimSpace(path), nil
	}
	home, ok := lookup("HOME")
	if !ok || strings.TrimSpace(home) == "" {
		return "", ErrHomeNotSet
	}
	return filepath.Join(home, ".exa-cli", "config.json"), nil
}

func Redact(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
