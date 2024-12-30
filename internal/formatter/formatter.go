package formatter

import (
    "github.com/KnockOutEZ/diffdeck/internal/scanner"
)

// Formatter interface defines the contract for different output formats
type Formatter interface {
    Format(files []scanner.File, cfg FormatConfig) (string, error)
}

// FormatConfig holds formatting configuration
type FormatConfig struct {
    HeaderText          string
    InstructionText     string
    ShowFileSummary     bool
    ShowDirStructure    bool
    ShowLineNumbers     bool
    TopFilesLength      int
}

// NewFormatter creates a new formatter based on the style
func NewFormatter(style string) Formatter {
    switch style {
    case "markdown":
        return &MarkdownFormatter{}
    case "xml":
        return &XMLFormatter{}
    default:
        return &PlainFormatter{}
    }
}

