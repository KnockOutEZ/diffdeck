package formatter

import (
    "fmt"
    "strings"
    "github.com/KnockOutEZ/diffdeck/internal/scanner"
)

type PlainFormatter struct{}

func (f *PlainFormatter) Format(files []scanner.File, cfg FormatConfig) (string, error) {
    var sb strings.Builder

    // Add header
    sb.WriteString("This file is a merged representation of the entire codebase, combining all repository files into a single document.\n\n")
    
    if cfg.HeaderText != "" {
        sb.WriteString(cfg.HeaderText)
        sb.WriteString("\n\n")
    }

    // Add file summary if enabled
    if cfg.ShowFileSummary {
        sb.WriteString("================================================================\n")
        sb.WriteString("File Summary\n")
        sb.WriteString("================================================================\n")
        summary := generateFileSummary(files, cfg.TopFilesLength)
        sb.WriteString(summary)
        sb.WriteString("\n\n")
    }

    // Add directory structure if enabled
    if cfg.ShowDirStructure {
        sb.WriteString("================================================================\n")
        sb.WriteString("Directory Structure\n")
        sb.WriteString("================================================================\n")
        structure := generateDirectoryStructure(files)
        sb.WriteString(structure)
        sb.WriteString("\n\n")
    }

    // Add files
    sb.WriteString("================================================================\n")
    sb.WriteString("Files\n")
    sb.WriteString("================================================================\n\n")

    for _, file := range files {
        if !file.IsDir {
            sb.WriteString("================\n")
            sb.WriteString(fmt.Sprintf("File: %s\n", file.Path))
            sb.WriteString("================\n")
            sb.WriteString(file.Content)
            sb.WriteString("\n\n")
        }
    }

    // Add instructions if provided
    if cfg.InstructionText != "" {
        sb.WriteString("================================================================\n")
        sb.WriteString("Instructions\n")
        sb.WriteString("================================================================\n")
        sb.WriteString(cfg.InstructionText)
        sb.WriteString("\n")
    }

    return sb.String(), nil
}
