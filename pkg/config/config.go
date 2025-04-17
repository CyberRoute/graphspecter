package config

import (
	"encoding/json"
	"fmt"
	"github.com/CyberRoute/graphspecter/pkg/types"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"time"
)

func LoadConfigFile(path string) (*types.FileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg types.FileConfig
	ext := filepath.Ext(path)

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}

	if cfg.TimeoutRaw != "" {
		parsedTimeout, err := time.ParseDuration(cfg.TimeoutRaw)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout duration: %w", err)
		}
		cfg.Timeout = parsedTimeout
	}

	return &cfg, nil
}
