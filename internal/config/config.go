package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type OutputConfig struct {
    FilePath            string `json:"filePath"`
    Style               string `json:"style"`
    HeaderText          string `json:"headerText"`
    InstructionFilePath string `json:"instructionFilePath"`
    FileSummary         bool   `json:"fileSummary"`
    DirectoryStructure  bool   `json:"directoryStructure"`
    RemoveComments      bool   `json:"removeComments"`
    RemoveEmptyLines    bool   `json:"removeEmptyLines"`
    ShowLineNumbers     bool   `json:"showLineNumbers"`
    CopyToClipboard     bool   `json:"copyToClipboard"`
    TopFilesLength      int    `json:"topFilesLength"`
    IncludeEmptyDirs    bool   `json:"includeEmptyDirectories"`
}

type IgnoreConfig struct {
    UseGitignore       bool     `json:"useGitignore"`
    UseDefaultPatterns bool     `json:"useDefaultPatterns"`
    CustomPatterns     []string `json:"customPatterns"`
}

type SecurityConfig struct {
    EnableSecurityCheck bool `json:"enableSecurityCheck"`
}

type Config struct {
    Output   OutputConfig   `json:"output"`
    Include  []string      `json:"include"`
    Ignore   IgnoreConfig  `json:"ignore"`
    Security SecurityConfig `json:"security"`
}

// Load loads the configuration from a file. If no file is specified,
// it looks for diffdeck.config.json in the current directory.
func Load(path string) (*Config, error) {
    if path == "" {
        path = "diffdeck.config.json"
    }

    cfg := DefaultConfig

    // Check if config file exists
    if _, err := os.Stat(path); err == nil {
        data, err := os.ReadFile(path)
        if err != nil {
            return nil, err
        }

        if err := json.Unmarshal(data, &cfg); err != nil {
            return nil, err
        }
    } else if !os.IsNotExist(err) {
        return nil, err
    }

    if err := cfg.validate(); err != nil {
        return nil, err
    }

    return &cfg, nil
}

// Save saves the configuration to a file
func (c *Config) Save(path string) error {
    if path == "" {
        path = "diffdeck.config.json"
    }

    data, err := json.MarshalIndent(c, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(path, data, 0644)
}

// validate checks if the configuration is valid
func (c *Config) validate() error {
    // Validate output style
    switch c.Output.Style {
    case "plain", "xml", "markdown":
        // valid styles
    default:
        return errors.New("invalid output style: must be 'plain', 'xml', or 'markdown'")
    }

    // Validate TopFilesLength
    if c.Output.TopFilesLength < 0 {
        return errors.New("topFilesLength must be non-negative")
    }

    // Validate file paths
    if c.Output.InstructionFilePath != "" {
        if _, err := os.Stat(c.Output.InstructionFilePath); err != nil {
            return errors.New("instruction file not found")
        }
    }

    // Ensure output directory exists
    outputDir := filepath.Dir(c.Output.FilePath)
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        return err
    }

    return nil
}

// GetIgnorePatterns returns all ignore patterns based on the configuration
func (c *Config) GetIgnorePatterns() ([]string, error) {
    var patterns []string

    // Add default patterns if enabled
    if c.Ignore.UseDefaultPatterns {
        patterns = append(patterns, DefaultIgnorePatterns...)
    }

    // Add custom patterns
    patterns = append(patterns, c.Ignore.CustomPatterns...)

    // Add patterns from .gitignore if enabled
    if c.Ignore.UseGitignore {
        gitignorePatterns, err := loadGitignorePatterns()
        if err != nil {
            return nil, err
        }
        patterns = append(patterns, gitignorePatterns...)
    }

    return patterns, nil
}

// loadGitignorePatterns loads patterns from .gitignore file
func loadGitignorePatterns() ([]string, error) {
    data, err := os.ReadFile(".gitignore")
    if err != nil {
        if os.IsNotExist(err) {
            return nil, nil
        }
        return nil, err
    }

    var patterns []string
    lines := strings.Split(string(data), "\n")
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line != "" && !strings.HasPrefix(line, "#") {
            patterns = append(patterns, line)
        }
    }

    return patterns, nil
}