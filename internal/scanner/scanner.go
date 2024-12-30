package scanner

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/KnockOutEZ/diffdeck/internal/config"
	"github.com/bmatcuk/doublestar/v4"
)

type File struct {
    Path     string
    Content  string
    Size     int64
    IsDir    bool
    Children []File
}

type Scanner struct {
    cfg        *config.Config
    patterns   []string
    ignorePats []string
}

func New(cfg *config.Config) (*Scanner, error) {
    ignorePats, err := cfg.GetIgnorePatterns()
    if err != nil {
        return nil, err
    }

    return &Scanner{
        cfg:        cfg,
        patterns:   cfg.Include,
        ignorePats: ignorePats,
    }, nil
}

// Scan scans the given paths and returns a slice of File structs
func (s *Scanner) Scan(paths []string) ([]File, error) {
    if len(paths) == 0 {
        paths = []string{"."}
    }

    var files []File
    for _, path := range paths {
        scanned, err := s.scanPath(path)
        if err != nil {
            return nil, err
        }
        files = append(files, scanned...)
    }

    // Sort files by path for consistent output
    sort.Slice(files, func(i, j int) bool {
        return files[i].Path < files[j].Path
    })

    return files, nil
}

// scanPath scans a single path and returns found files
func (s *Scanner) scanPath(root string) ([]File, error) {
    var files []File

    err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }

        // Convert path to relative
        relPath, err := filepath.Rel(root, path)
        if err != nil {
            return err
        }

        // Skip if path matches ignore patterns
        if s.shouldIgnore(relPath) {
            if d.IsDir() {
                return filepath.SkipDir
            }
            return nil
        }

        // Check if path matches include patterns
        if !s.shouldInclude(relPath) {
            return nil
        }

        file := File{
            Path:  relPath,
            IsDir: d.IsDir(),
        }

        if !d.IsDir() {
            // Read file content
            content, err := os.ReadFile(path)
            if err != nil {
                return err
            }

            file.Content = string(content)
            info, err := d.Info()
            if err != nil {
                return err
            }
            file.Size = info.Size()

            // Process content according to config
            if s.cfg.Output.RemoveComments {
                file.Content = s.removeComments(file.Content, filepath.Ext(path))
            }
            if s.cfg.Output.RemoveEmptyLines {
                file.Content = s.removeEmptyLines(file.Content)
            }
            if s.cfg.Output.ShowLineNumbers {
                file.Content = s.addLineNumbers(file.Content)
            }
        }

        files = append(files, file)
        return nil
    })

    if err != nil {
        return nil, err
    }

    // Build directory tree if needed
    if s.cfg.Output.DirectoryStructure {
        files = s.buildDirectoryTree(files)
    }

    return files, nil
}

// shouldIgnore checks if a path should be ignored
func (s *Scanner) shouldIgnore(path string) bool {
    for _, pattern := range s.ignorePats {
        matched, err := doublestar.Match(pattern, path)
        if err == nil && matched {
            return true
        }
    }
    return false
}

// shouldInclude checks if a path should be included
func (s *Scanner) shouldInclude(path string) bool {
    if len(s.patterns) == 0 {
        return true
    }

    for _, pattern := range s.patterns {
        matched, err := doublestar.Match(pattern, path)
        if err == nil && matched {
            return true
        }
    }
    return false
}

// removeComments removes comments from the content based on file extension
func (s *Scanner) removeComments(content, ext string) string {
    // Simple comment removal for common file types
    // In a production environment, you might want to use a proper parser
    lines := strings.Split(content, "\n")
    var result []string

    inMultilineComment := false

    for _, line := range lines {
        trimmed := strings.TrimSpace(line)

        switch ext {
        case ".go", ".java", ".js", ".ts":
            if inMultilineComment {
                if strings.Contains(trimmed, "*/") {
                    inMultilineComment = false
                }
                continue
            }

            if strings.HasPrefix(trimmed, "//") {
                continue
            }

            if strings.HasPrefix(trimmed, "/*") {
                inMultilineComment = true
                continue
            }

        case ".py":
            if strings.HasPrefix(trimmed, "#") {
                continue
            }

        case ".html", ".xml":
            if strings.HasPrefix(trimmed, "<!--") && strings.HasSuffix(trimmed, "-->") {
                continue
            }
        }

        result = append(result, line)
    }

    return strings.Join(result, "\n")
}

// removeEmptyLines removes empty lines from the content
func (s *Scanner) removeEmptyLines(content string) string {
    lines := strings.Split(content, "\n")
    var result []string

    for _, line := range lines {
        if strings.TrimSpace(line) != "" {
            result = append(result, line)
        }
    }

    return strings.Join(result, "\n")
}

// addLineNumbers adds line numbers to the content
func (s *Scanner) addLineNumbers(content string) string {
    lines := strings.Split(content, "\n")
    var result []string

    for i, line := range lines {
        result = append(result, fmt.Sprintf("%5d | %s", i+1, line))
    }

    return strings.Join(result, "\n")
}

// buildDirectoryTree builds a tree structure from flat files list
func (s *Scanner) buildDirectoryTree(files []File) []File {
    // Create a map of path to file for easy lookup
    fileMap := make(map[string]*File)
    var roots []File

    // First pass: create all directories and files
    for _, f := range files {
        dir := filepath.Dir(f.Path)
        if dir == "." {
            roots = append(roots, f)
        } else {
            // Ensure all parent directories exist
            parts := strings.Split(dir, string(filepath.Separator))
            currentPath := ""
            for _, part := range parts {
                if currentPath == "" {
                    currentPath = part
                } else {
                    currentPath = filepath.Join(currentPath, part)
                }
                
                if _, exists := fileMap[currentPath]; !exists {
                    dirFile := File{
                        Path:  currentPath,
                        IsDir: true,
                    }
                    fileMap[currentPath] = &dirFile
                }
            }

            // Add file to its parent directory's children
            parent := fileMap[dir]
            if parent != nil {
                parent.Children = append(parent.Children, f)
            }
        }
    }

    // Sort children in each directory
    for _, file := range fileMap {
        if file.IsDir {
            sort.Slice(file.Children, func(i, j int) bool {
                return file.Children[i].Path < file.Children[j].Path
            })
        }
    }

    return roots
}
