package formatter

import (
    "bytes"
    "fmt"
    "strings"
    "github.com/KnockOutEZ/diffdeck/internal/git"
)

type Options struct {
    Style           string
    ShowLineNumbers bool
    TopFilesLength  int
    DiffMode        string
}

type Formatter interface {
    Format(changes []git.FileChange) (string, error)
}

func NewFormatter(opts Options) Formatter {
    switch opts.Style {
    case "markdown":
        return &MarkdownFormatter{opts: opts}
    case "xml":
        return &XMLFormatter{opts: opts}
    default:
        return &PlainFormatter{opts: opts}
    }
}

type PlainFormatter struct {
    opts Options
}

func (f *PlainFormatter) Format(changes []git.FileChange) (string, error) {
    var buf bytes.Buffer

    buf.WriteString("Diffdeck Output\n")
    buf.WriteString("==============\n\n")

    buf.WriteString(fmt.Sprintf("Total changes: %d\n", len(changes)))
    buf.WriteString(fmt.Sprintf("Diff mode: %s\n\n", f.opts.DiffMode))

    for _, change := range changes {
        buf.WriteString(fmt.Sprintf("File: %s\n", change.Path))
        buf.WriteString(fmt.Sprintf("Status: %s\n", change.Status))
        if change.Status == git.Renamed {
            buf.WriteString(fmt.Sprintf("Old path: %s\n", change.OldPath))
        }
        buf.WriteString("----------------------------------------\n")

        switch f.opts.DiffMode {
        case "unified":
            diff := generateUnifiedDiff(change.OldContent, change.Content, f.opts.ShowLineNumbers)
            buf.WriteString(diff)
        case "side-by-side":
            diff := generateSideBySideDiff(change.OldContent, change.Content, f.opts.ShowLineNumbers)
            buf.WriteString(diff)
        default:
            buf.WriteString(change.Content)
        }

        buf.WriteString("\n\n")
    }

    return buf.String(), nil
}

func generateUnifiedDiff(oldContent, newContent string, showLineNumbers bool) string {
    if oldContent == "" {
        return newContent
    }

    var buf bytes.Buffer
    oldLines := strings.Split(oldContent, "\n")
    newLines := strings.Split(newContent, "\n")

    for i := 0; i < len(oldLines) || i < len(newLines); i++ {
        if i < len(oldLines) && i < len(newLines) {
            if oldLines[i] != newLines[i] {
                if showLineNumbers {
                    buf.WriteString(fmt.Sprintf("-%d: %s\n", i+1, oldLines[i]))
                    buf.WriteString(fmt.Sprintf("+%d: %s\n", i+1, newLines[i]))
                } else {
                    buf.WriteString(fmt.Sprintf("-%s\n", oldLines[i]))
                    buf.WriteString(fmt.Sprintf("+%s\n", newLines[i]))
                }
            } else {
                if showLineNumbers {
                    buf.WriteString(fmt.Sprintf(" %d: %s\n", i+1, oldLines[i]))
                } else {
                    buf.WriteString(fmt.Sprintf(" %s\n", oldLines[i]))
                }
            }
        } else if i < len(oldLines) {
            if showLineNumbers {
                buf.WriteString(fmt.Sprintf("-%d: %s\n", i+1, oldLines[i]))
            } else {
                buf.WriteString(fmt.Sprintf("-%s\n", oldLines[i]))
            }
        } else {
            if showLineNumbers {
                buf.WriteString(fmt.Sprintf("+%d: %s\n", i+1, newLines[i]))
            } else {
                buf.WriteString(fmt.Sprintf("+%s\n", newLines[i]))
            }
        }
    }

    return buf.String()
}

func generateSideBySideDiff(oldContent, newContent string, showLineNumbers bool) string {
    var buf bytes.Buffer
    oldLines := strings.Split(oldContent, "\n")
    newLines := strings.Split(newContent, "\n")

    maxWidth := 80
    separator := " | "

    for i := 0; i < len(oldLines) || i < len(newLines); i++ {
        var leftLine, rightLine string

        if i < len(oldLines) {
            leftLine = oldLines[i]
        }
        if i < len(newLines) {
            rightLine = newLines[i]
        }

        if showLineNumbers {
            leftNum := fmt.Sprintf("%4d", i+1)
            rightNum := fmt.Sprintf("%4d", i+1)
            buf.WriteString(fmt.Sprintf("%s: %-*s %s %s: %s\n",
                leftNum, maxWidth, leftLine, separator, rightNum, rightLine))
        } else {
            buf.WriteString(fmt.Sprintf("%-*s %s %s\n",
                maxWidth, leftLine, separator, rightLine))
        }
    }

    return buf.String()
}
