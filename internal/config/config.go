package config

import (
    "encoding/json"
    "os"
    "path/filepath"
)

type Config struct {
    Output struct {
        FilePath        string `json:"filePath"`
        Style          string `json:"style"`
        ShowLineNumbers bool   `json:"showLineNumbers"`
        CopyToClipboard bool   `json:"copyToClipboard"`
        TopFilesLength  int    `json:"topFilesLength"`
    } `json:"output"`

    Include []string `json:"include"`
    Ignore  struct {
        Patterns []string `json:"patterns"`
    } `json:"ignore"`
    
    Security struct {
        DisableSecurityCheck bool  `json:"disableSecurityCheck"`
        MaxFileSize         int64 `json:"maxFileSize"`
    } `json:"security"`

    Git struct {
        DefaultRemote string `json:"defaultRemote"`
        CacheDir     string `json:"cacheDir"`
        Timeout      string `json:"timeout"`
    } `json:"git"`
}

func DefaultConfig() *Config {
    cfg := &Config{}
    
    // Set default output options
    cfg.Output.FilePath = "diffdeck-output.txt"
    cfg.Output.Style = "plain"
    cfg.Output.ShowLineNumbers = false
    cfg.Output.CopyToClipboard = false
    cfg.Output.TopFilesLength = 5

    // Set default patterns
    cfg.Include = []string{"**/*"}  // Match everything by default
    cfg.Ignore.Patterns = []string{
        ".git/**",
        ".github/**",
        "node_modules/**",
        "vendor/**",
        "dist/**",
        "build/**",
        "*.exe",
        "*.dll",
        "*.so",
        "*.dylib",
        "*.test",
        "*.out",
        "*.log",
        "*.tmp",
        "*.temp",
        ".DS_Store",
        "Thumbs.db",
        "**/.git/**",
        "**/node_modules/**",
        "**/vendor/**",
        "**/.idea/**",
        "**/.vscode/**",
    }

    // Set default security options
    cfg.Security.DisableSecurityCheck = false
    cfg.Security.MaxFileSize = 10 * 1024 * 1024 // 10MB

    // Set default git options
    cfg.Git.CacheDir = filepath.Join(os.TempDir(), "diffdeck-cache")
    cfg.Git.Timeout = "5m"

    return cfg
}

func Load(path string) (*Config, error) {
    cfg := DefaultConfig()

    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return cfg, nil
        }
        return nil, err
    }

    if err := json.Unmarshal(data, cfg); err != nil {
        return nil, err
    }

    return cfg, nil
}

func (c *Config) Save(path string) error {
    data, err := json.MarshalIndent(c, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(path, data, 0644)
}
