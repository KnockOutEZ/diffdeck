// File: internal/formatter/xml.go

package formatter

import (
    "encoding/xml"
    "fmt"
    "time"
    "github.com/KnockOutEZ/diffdeck/internal/git"
)

type XMLFormatter struct {
    opts Options
}

type XMLOutput struct {
    XMLName   xml.Name    `xml:"diffdeck"`
    Generated string      `xml:"generated,attr"`
    Summary   XMLSummary  `xml:"summary"`
    Changes   []XMLChange `xml:"changes>change"`
}

type XMLSummary struct {
    TotalFiles int    `xml:"totalFiles"`
    DiffMode   string `xml:"diffMode"`
}

type XMLChange struct {
    Path       string `xml:"path,attr"`
    OldPath    string `xml:"oldPath,omitempty"`
    Status     string `xml:"status"`
    Language   string `xml:"language"`
    OldContent string `xml:"oldContent,omitempty"`
    NewContent string `xml:"newContent,omitempty"`
    Diff       string `xml:"diff,omitempty"`
}

func (f *XMLFormatter) Format(changes []git.FileChange) (string, error) {
    output := XMLOutput{
        Generated: time.Now().Format(time.RFC3339),
        Summary: XMLSummary{
            TotalFiles: len(changes),
            DiffMode:   f.opts.DiffMode,
        },
    }

    for _, change := range changes {
        xmlChange := XMLChange{
            Path:     change.Path,
            OldPath:  change.OldPath,
            Status:   string(change.Status),
            Language: change.Language,
        }

        switch f.opts.DiffMode {
        case "unified":
            xmlChange.Diff = generateUnifiedDiff(change.OldContent, change.Content, f.opts.ShowLineNumbers)
        case "side-by-side":
            xmlChange.Diff = generateSideBySideDiff(change.OldContent, change.Content, f.opts.ShowLineNumbers)
        default:
            xmlChange.OldContent = change.OldContent
            xmlChange.NewContent = change.Content
        }

        output.Changes = append(output.Changes, xmlChange)
    }

    data, err := xml.MarshalIndent(output, "", "  ")
    if err != nil {
        return "", fmt.Errorf("failed to marshal XML: %w", err)
    }

    return xml.Header + string(data), nil
}
