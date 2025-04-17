package config

import (
	"github.com/CyberRoute/graphspecter/pkg/types"
	"time"
)

func ApplyFileConfigToCLIConfig(fileCfg *types.FileConfig, cliCfg *types.CLIConfig) {
	if cliCfg.BaseURL == "" {
		cliCfg.BaseURL = fileCfg.BaseURL
	}
	if cliCfg.Timeout == 1*time.Second {
		cliCfg.Timeout = fileCfg.Timeout
	}
	if cliCfg.LogLevel == "" {
		cliCfg.LogLevel = fileCfg.LogLevel
	}
	if cliCfg.LogFile == "" {
		cliCfg.LogFile = fileCfg.LogFile
	}
	if cliCfg.OutputFile != fileCfg.OutputFile {
		cliCfg.OutputFile = fileCfg.OutputFile
	}
	if cliCfg.MaxDepth == 10 {
		cliCfg.MaxDepth = fileCfg.MaxDepth
	}
	if !cliCfg.NoColor {
		cliCfg.NoColor = fileCfg.NoColor
	}
	if cliCfg.SchemaFile == "" {
		cliCfg.SchemaFile = fileCfg.SchemaFile
	}
	if cliCfg.Headers == nil && len(fileCfg.Headers) > 0 {
		cliCfg.Headers = make(map[string]string, len(fileCfg.Headers))
		for k, v := range fileCfg.Headers {
			cliCfg.Headers[k] = v
		}
	}
	if !cliCfg.Detect && fileCfg.Detect {
		cliCfg.Detect = true
	}
}
