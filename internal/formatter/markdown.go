package formatter

import (
    "fmt"
    "strings"
    "github.com/KnockOutEZ/diffdeck/internal/scanner"
)

type MarkdownFormatter struct{}

func (f *MarkdownFormatter) Format(files []scanner.File, cfg FormatConfig) (string, error) {
    var sb strings.Builder

    // Add header
    sb.WriteString("# Codebase Overview\n\n")
    sb.WriteString("This file is a merged representation of the entire codebase, combining all repository files into a single document.\n\n")
    
    if cfg.HeaderText != "" {
        sb.WriteString(cfg.HeaderText)
        sb.WriteString("\n\n")
    }

    // Add file summary if enabled
    if cfg.ShowFileSummary {
        sb.WriteString("## File Summary\n\n")
        summary := generateFileSummary(files, cfg.TopFilesLength)
        sb.WriteString(summary)
        sb.WriteString("\n\n")
    }

    // Add directory structure if enabled
    if cfg.ShowDirStructure {
        sb.WriteString("## Directory Structure\n\n")
        sb.WriteString("```\n")
        structure := generateDirectoryStructure(files)
        sb.WriteString(structure)
        sb.WriteString("```\n\n")
    }

    // Add files
    sb.WriteString("## Repository Files\n\n")
    for _, file := range files {
        if !file.IsDir {
            sb.WriteString(fmt.Sprintf("### File: %s\n\n", file.Path))
            sb.WriteString("```")
            
            // Add language hint for syntax highlighting if available
            if ext := getLanguageFromPath(file.Path); ext != "" {
                sb.WriteString(ext)
            }
            
            sb.WriteString("\n")
            sb.WriteString(file.Content)
            sb.WriteString("\n```\n\n")
        }
    }

    // Add instructions if provided
    if cfg.InstructionText != "" {
        sb.WriteString("## Instructions\n\n")
        sb.WriteString(cfg.InstructionText)
        sb.WriteString("\n")
    }

    return sb.String(), nil
}
