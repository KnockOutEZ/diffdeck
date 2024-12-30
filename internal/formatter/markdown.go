// File: internal/formatter/markdown.go

package formatter

import (
    "fmt"
    "strings"
    "time"
    "github.com/KnockOutEZ/diffdeck/internal/git"
)

type MarkdownFormatter struct {
    opts Options
}

func (f *MarkdownFormatter) Format(changes []git.FileChange) (string, error) {
    var buf strings.Builder

    // Write header
    buf.WriteString("# Diffdeck Output\n\n")
    buf.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339)))

    // Write summary
    buf.WriteString("## Summary\n\n")
    buf.WriteString(fmt.Sprintf("- Total changes: %d\n", len(changes)))
    buf.WriteString(fmt.Sprintf("- Diff mode: %s\n\n", f.opts.DiffMode))

    // Write changes
    buf.WriteString("## Changes\n\n")
    for _, change := range changes {
        buf.WriteString(fmt.Sprintf("### %s\n\n", change.Path))
        buf.WriteString(fmt.Sprintf("- Status: `%s`\n", change.Status))
        buf.WriteString(fmt.Sprintf("- Language: `%s`\n", change.Language))
        if change.Status == git.Renamed {
            buf.WriteString(fmt.Sprintf("- Old path: `%s`\n", change.OldPath))
        }
        buf.WriteString("\n")

        switch f.opts.DiffMode {
        case "unified":
            diff := generateUnifiedDiff(change.OldContent, change.Content, f.opts.ShowLineNumbers)
            buf.WriteString("```diff\n")
            buf.WriteString(diff)
            buf.WriteString("```\n\n")
        case "side-by-side":
            diff := generateSideBySideDiff(change.OldContent, change.Content, f.opts.ShowLineNumbers)
            buf.WriteString("```\n")
            buf.WriteString(diff)
            buf.WriteString("```\n\n")
        default:
            buf.WriteString("```")
            if change.Language != "Unknown" {
                buf.WriteString(strings.ToLower(change.Language))
            }
            buf.WriteString("\n")
            buf.WriteString(change.Content)
            buf.WriteString("```\n\n")
        }
    }

    return buf.String(), nil
}