package security

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/KnockOutEZ/diffdeck/internal/git"
	"github.com/schollz/progressbar/v3"
)

type Options struct {
    MaxFileSize int64
    Progress    *progressbar.ProgressBar
    CustomPatterns map[string]string
    SkipBinaries  bool
    Severity      string
}

type Issue struct {
    FilePath    string
    Line        int
    Column      int
    Rule        string
    Description string
    Severity    string
    Content     string
}

type Checker struct {
    patterns  map[string]*regexp.Regexp
    progress  *progressbar.ProgressBar
    maxSize   int64
    skipBinaries bool
    severity     string
    mu        sync.Mutex
}

func NewChecker(opts Options) *Checker {
    patterns := defaultPatterns()
    
    // Add custom patterns if provided
    if opts.CustomPatterns != nil {
        for name, pattern := range opts.CustomPatterns {
            compiled, err := regexp.Compile(pattern)
            if err == nil {
                patterns[name] = compiled
            }
        }
    }

    return &Checker{
        patterns: patterns,
        progress: opts.Progress,
        maxSize:  opts.MaxFileSize,
        skipBinaries: opts.SkipBinaries,
        severity: opts.Severity,
    }
}

func (c *Checker) Check(changes []git.FileChange) ([]Issue, error) {
    var issues []Issue
    var wg sync.WaitGroup
    semaphore := make(chan struct{}, 10) // Limit concurrent checks

    for _, change := range changes {
        wg.Add(1)
        go func(fc git.FileChange) {
            defer wg.Done()
            semaphore <- struct{}{} // Acquire
            defer func() { <-semaphore }() // Release

            fileIssues := c.checkFile(fc)
            
            c.mu.Lock()
            issues = append(issues, fileIssues...)
            c.mu.Unlock()

            if c.progress != nil {
                c.progress.Add(1)
            }
        }(change)
    }

    wg.Wait()
    return issues, nil
}

func (c *Checker) checkFile(change git.FileChange) []Issue {
    var issues []Issue

    // Skip large files
    if int64(len(change.Content)) > c.maxSize {
        return issues
    }

    // Skip if the file looks like a Go file with imports
    isGoFile := strings.HasSuffix(change.Path, ".go")
    
    lines := strings.Split(change.Content, "\n")
    inImportBlock := false

    for lineNum, line := range lines {
        // Skip import blocks in Go files
        if isGoFile {
            if strings.HasPrefix(strings.TrimSpace(line), "import (") {
                inImportBlock = true
                continue
            }
            if inImportBlock {
                if strings.HasPrefix(strings.TrimSpace(line), ")") {
                    inImportBlock = false
                }
                continue
            }
        }

        // Check each pattern
        for name, pattern := range c.patterns {
            matches := pattern.FindAllStringIndex(line, -1)
            for _, match := range matches {
                start, end := match[0], match[1]

                // Skip if the match is part of a Go import path
                if isGoFile && strings.Contains(line[:start], "import") {
                    continue
                }

                // Get some context around the match
                contextStart := max(0, start-20)
                contextEnd := min(len(line), end+20)
                context := line[contextStart:contextEnd]

                issues = append(issues, Issue{
                    FilePath:    change.Path,
                    Line:       lineNum + 1,
                    Column:     start + 1,
                    Rule:       name,
                    Description: fmt.Sprintf("Found potential %s", name),
                    Severity:   "WARNING",
                    Content:    context,
                })
            }
        }
    }

    return issues
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func defaultPatterns() map[string]*regexp.Regexp {
    return map[string]*regexp.Regexp{
        "AWS Access Key":     regexp.MustCompile(`(?i)AKIA[0-9A-Z]{16}`),
        "AWS Secret Key":     regexp.MustCompile(`(?i)(aws_secret|aws_key|aws_token|aws_access).{0,20}[A-Za-z0-9/+=]{40}`),
        "Private Key":        regexp.MustCompile(`-----BEGIN (?:RSA |DSA |EC )?PRIVATE KEY-----`),
        "SSH Private Key":    regexp.MustCompile(`-----BEGIN OPENSSH PRIVATE KEY-----`),
        "GitHub Token":       regexp.MustCompile(`(?i)(github|gh)[0-9a-zA-Z_-]*token[ :="\']([0-9a-zA-Z]{35,40})`),
        "Google API Key":     regexp.MustCompile(`AIza[0-9A-Za-z\-_]{35}`),
        "Password in Code":   regexp.MustCompile(`(?i)(?:password|passwd|pwd)[ :=]+['"][^'"\n]{8,}['"]`),
        "API Key in Code":    regexp.MustCompile(`(?i)(?:api[_-]?key|api[_-]?secret|api[_-]?token)[ :=]+['"][^'"\n]{8,}['"]`),
        "IP Address":         regexp.MustCompile(`(?:^|\s|=)(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9])(?:\s|$)`),
        "Internal URL":       regexp.MustCompile(`(?i)(?:localhost|127\.0\.0\.1|0\.0\.0\.0):\d+`),
    }
}

func findPosition(content string, offset int) (line, column int) {
    line = 1
    column = 1
    for i := 0; i < offset; i++ {
        if content[i] == '\n' {
            line++
            column = 1
        } else {
            column++
        }
    }
    return
}
