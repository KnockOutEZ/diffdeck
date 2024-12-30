package formatter

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/KnockOutEZ/diffdeck/internal/scanner"
)

type XMLFormatter struct{}

type xmlOutput struct {
    XMLName     xml.Name `xml:"repository"`
    Summary     string   `xml:"file_summary,omitempty"`
    Structure   string   `xml:"directory_structure,omitempty"`
    Files       []xmlFile `xml:"files>file"`
    Instruction string   `xml:"instruction,omitempty"`
}

type xmlFile struct {
    Path    string `xml:"path,attr"`
    Content string `xml:",cdata"`
}

func (f *XMLFormatter) Format(files []scanner.File, cfg FormatConfig) (string, error) {
    output := xmlOutput{}
    
    // Add file summary if enabled
    if cfg.ShowFileSummary {
        output.Summary = generateFileSummary(files, cfg.TopFilesLength)
    }

    // Add directory structure if enabled
    if cfg.ShowDirStructure {
        output.Structure = generateDirectoryStructure(files)
    }

    // Add files
    for _, file := range files {
        if !file.IsDir {
            output.Files = append(output.Files, xmlFile{
                Path:    file.Path,
                Content: file.Content,
            })
        }
    }

    // Add instructions if provided
    if cfg.InstructionText != "" {
        output.Instruction = cfg.InstructionText
    }

    // Marshal to XML
    data, err := xml.MarshalIndent(output, "", "  ")
    if err != nil {
        return "", err
    }

    // Add XML header and return
    return xml.Header + string(data), nil
}

// Utility functions

func generateFileSummary(files []scanner.File, topN int) string {
    var sb strings.Builder
    var fileCount, totalSize int64

    // Count files and calculate total size
    for _, file := range files {
        if !file.IsDir {
            fileCount++
            totalSize += file.Size
        }
    }

    sb.WriteString(fmt.Sprintf("Total Files: %d\n", fileCount))
    sb.WriteString(fmt.Sprintf("Total Size: %s\n", formatSize(totalSize)))
    
    if topN > 0 {
        sb.WriteString(fmt.Sprintf("\nTop %d largest files:\n", topN))
        topFiles := getLargestFiles(files, topN)
        for i, file := range topFiles {
            sb.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, file.Path, formatSize(file.Size)))
        }
    }

    return sb.String()
}

func generateDirectoryStructure(files []scanner.File) string {
    var sb strings.Builder
    printTree(&sb, files, "", "")
    return sb.String()
}

func printTree(sb *strings.Builder, files []scanner.File, prefix, childPrefix string) {
    for i, file := range files {
        isLast := i == len(files)-1
        
        // Print current file/directory
        if isLast {
            sb.WriteString(prefix + "└── ")
            childPrefix += "    "
        } else {
            sb.WriteString(prefix + "├── ")
            childPrefix += "│   "
        }
        
        sb.WriteString(file.Path + "\n")
        
        // Recursively print children
        if file.IsDir && len(file.Children) > 0 {
            printTree(sb, file.Children, childPrefix, childPrefix)
        }
    }
}

func formatSize(size int64) string {
    const unit = 1024
    if size < unit {
        return fmt.Sprintf("%d B", size)
    }
    div, exp := int64(unit), 0
    for n := size / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func getLargestFiles(files []scanner.File, n int) []scanner.File {
    var nonDirs []scanner.File
    for _, file := range files {
        if !file.IsDir {
            nonDirs = append(nonDirs, file)
        }
    }

    // Sort by size in descending order
    sort.Slice(nonDirs, func(i, j int) bool {
        return nonDirs[i].Size > nonDirs[j].Size
    })

    if len(nonDirs) > n {
        return nonDirs[:n]
    }
    return nonDirs
}

func getLanguageFromPath(path string) string {
    ext := strings.ToLower(filepath.Ext(path))
    switch ext {
    case ".go":
        return "go"
    case ".js":
        return "javascript"
    case ".py":
        return "python"
    case ".java":
        return "java"
    case ".cpp", ".hpp", ".cc", ".hh":
        return "cpp"
    case ".ts":
        return "typescript"
    case ".rb":
        return "ruby"
    case ".php":
        return "php"
    case ".rs":
        return "rust"
    case ".swift":
        return "swift"
    case ".kt":
        return "kotlin"
    case ".scala":
        return "scala"
    case ".html":
        return "html"
    case ".css":
        return "css"
    case ".md":
        return "markdown"
    case ".json":
        return "json"
    case ".xml":
        return "xml"
    case ".yaml", ".yml":
        return "yaml"
    default:
        return ""
    }
}
