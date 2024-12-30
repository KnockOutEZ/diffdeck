// internal/security/checker.go
package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/KnockOutEZ/diffdeck/internal/scanner"
)

// Issue represents a security issue found in the code
type Issue struct {
    FilePath    string   `json:"filePath"`
    Line        int      `json:"line"`
    Column      int      `json:"column"`
    RuleID      string   `json:"ruleId"`
    Message     string   `json:"message"`
    Severity    string   `json:"severity"`
    Matches     []string `json:"matches,omitempty"`
    Suggestion  string   `json:"suggestion,omitempty"`
}

// Checker handles security checks for files
type Checker struct {
    secretlintPath string
    rules          []string
    mu             sync.Mutex
}

// CheckerOptions configures the security checker
type CheckerOptions struct {
    CustomRules []string
    ExcludeRules []string
    Severity     string // "error", "warn", or "info"
}

// New creates a new security checker
func New(opts *CheckerOptions) (*Checker, error) {
    // Ensure Secretlint is installed
    secretlintPath, err := exec.LookPath("secretlint")
    if err != nil {
        return nil, fmt.Errorf("secretlint not found: %w", err)
    }

    // Default rules
    rules := []string{
        "@secretlint/secretlint-rule-preset-recommend",
        "@secretlint/secretlint-rule-pattern",
        "@secretlint/secretlint-rule-aws",
        "@secretlint/secretlint-rule-gcp",
        "@secretlint/secretlint-rule-privatekey",
    }

    // Add custom rules
    if opts != nil && len(opts.CustomRules) > 0 {
        rules = append(rules, opts.CustomRules...)
    }

    return &Checker{
        secretlintPath: secretlintPath,
        rules:         rules,
    }, nil
}

// Check performs security checks on the given files
func (c *Checker) Check(files []scanner.File) ([]Issue, error) {
    var issues []Issue
    var mu sync.Mutex
    var wg sync.WaitGroup
    semaphore := make(chan struct{}, 5) // Limit concurrent checks

    for _, file := range files {
        if file.IsDir {
            continue
        }

        wg.Add(1)
        go func(f scanner.File) {
            defer wg.Done()
            semaphore <- struct{}{} // Acquire semaphore
            defer func() { <-semaphore }() // Release semaphore

            // Create temporary file for checking
            tempFile, err := c.createTempFile(f)
            if err != nil {
                fmt.Fprintf(os.Stderr, "Error creating temp file for %s: %v\n", f.Path, err)
                return
            }
            defer os.Remove(tempFile)

            // Run Secretlint
            fileIssues, err := c.checkFile(tempFile, f.Path)
            if err != nil {
                fmt.Fprintf(os.Stderr, "Error checking %s: %v\n", f.Path, err)
                return
            }

            // Add found issues
            if len(fileIssues) > 0 {
                mu.Lock()
                issues = append(issues, fileIssues...)
                mu.Unlock()
            }
        }(file)
    }

    wg.Wait()

    // Sort issues by file path and line number
    sort.Slice(issues, func(i, j int) bool {
        if issues[i].FilePath == issues[j].FilePath {
            return issues[i].Line < issues[j].Line
        }
        return issues[i].FilePath < issues[j].FilePath
    })

    return issues, nil
}

// createTempFile creates a temporary file with the given content
func (c *Checker) createTempFile(file scanner.File) (string, error) {
    tempDir, err := os.MkdirTemp("", "diffdeck-security-*")
    if err != nil {
        return "", err
    }

    tempFile := filepath.Join(tempDir, filepath.Base(file.Path))
    if err := os.WriteFile(tempFile, []byte(file.Content), 0644); err != nil {
        os.RemoveAll(tempDir)
        return "", err
    }

    return tempFile, nil
}

// checkFile runs Secretlint on a single file
func (c *Checker) checkFile(filePath, originalPath string) ([]Issue, error) {
    // Prepare Secretlint command
    cmd := exec.Command(c.secretlintPath, "--format", "json", filePath)
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    // Run Secretlint
    err := cmd.Run()
    if err != nil && !isExpectedError(err) {
        return nil, fmt.Errorf("secretlint error: %s", stderr.String())
    }

    // Parse results
    var result struct {
        Messages []struct {
            RuleID   string `json:"ruleId"`
            Message  string `json:"message"`
            Line     int    `json:"line"`
            Column   int    `json:"column"`
            Severity string `json:"severity"`
        } `json:"messages"`
    }

    if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
        return nil, fmt.Errorf("error parsing secretlint output: %w", err)
    }

    // Convert to our Issue format
    var issues []Issue
    for _, msg := range result.Messages {
        issue := Issue{
            FilePath: originalPath,
            Line:     msg.Line,
            Column:   msg.Column,
            RuleID:   msg.RuleID,
            Message:  msg.Message,
            Severity: msg.Severity,
        }
        issues = append(issues, issue)
    }

    return issues, nil
}

// isExpectedError checks if the error is an expected Secretlint error
// (Secretlint exits with code 1 when it finds issues)
func isExpectedError(err error) bool {
    if exitErr, ok := err.(*exec.ExitError); ok {
        return exitErr.ExitCode() == 1
    }
    return false
}

// Additional security checks

// checkPatterns checks for common sensitive patterns
func (c *Checker) checkPatterns(content string) []Issue {
    var issues []Issue
    patterns := map[string]string{
        "AWS Access Key":     `AKIA[0-9A-Z]{16}`,
        "AWS Secret Key":     `[0-9a-zA-Z/+]{40}`,
        "Private Key":        `-----BEGIN.*PRIVATE KEY-----`,
        "SSH Private Key":    `-----BEGIN.*SSH.*PRIVATE KEY-----`,
        "GitHub Token":       `gh[ps]_[0-9a-zA-Z]{36}`,
        "Google API Key":     `AIza[0-9A-Za-z\\-_]{35}`,
        "Password in Code":   `(?i)(?:password|passwd|pwd)\s*=\s*['"][^'"]+['"]`,
        "API Key in Code":    `(?i)(?:api_key|apikey|api_secret|apisecret)\s*=\s*['"][^'"]+['"]`,
        "IP Address":         `\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`,
        "Internal URL":       `(?i)(?:localhost|127\.0\.0\.1|0\.0\.0\.0):\d+`,
    }

    for name, pattern := range patterns {
        matches := regexp.MustCompile(pattern).FindAllString(content, -1)
        if len(matches) > 0 {
            issues = append(issues, Issue{
                Message:  fmt.Sprintf("Potential %s found", name),
                Matches: matches,
                Severity: "warning",
            })
        }
    }

    return issues
}

// CheckContent performs a quick security check on a string content
func (c *Checker) CheckContent(content string) []Issue {
    return c.checkPatterns(content)
}

// CreateReport generates a security report in various formats
func (c *Checker) CreateReport(issues []Issue, format string) (string, error) {
    switch format {
    case "json":
        data, err := json.MarshalIndent(issues, "", "  ")
        if err != nil {
            return "", err
        }
        return string(data), nil

    case "text":
        var sb strings.Builder
        sb.WriteString("Security Check Report\n")
        sb.WriteString("====================\n\n")

        if len(issues) == 0 {
            sb.WriteString("No security issues found.\n")
            return sb.String(), nil
        }

        for _, issue := range issues {
            sb.WriteString(fmt.Sprintf("File: %s\n", issue.FilePath))
            sb.WriteString(fmt.Sprintf("Line: %d, Column: %d\n", issue.Line, issue.Column))
            sb.WriteString(fmt.Sprintf("Rule: %s\n", issue.RuleID))
            sb.WriteString(fmt.Sprintf("Severity: %s\n", issue.Severity))
            sb.WriteString(fmt.Sprintf("Message: %s\n", issue.Message))
            if len(issue.Matches) > 0 {
                sb.WriteString("Matches:\n")
                for _, match := range issue.Matches {
                    sb.WriteString(fmt.Sprintf("  - %s\n", match))
                }
            }
            if issue.Suggestion != "" {
                sb.WriteString(fmt.Sprintf("Suggestion: %s\n", issue.Suggestion))
            }
            sb.WriteString("\n")
        }

        return sb.String(), nil

    default:
        return "", fmt.Errorf("unsupported format: %s", format)
    }
}