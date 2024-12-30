package utils

import (
    "path/filepath"
    "strings"

    "github.com/bmatcuk/doublestar/v4"
)

// PatternMatcher handles glob pattern matching
type PatternMatcher struct {
    includePatterns []string
    ignorePatterns  []string
    caseSensitive   bool
}

// NewPatternMatcher creates a new pattern matcher
func NewPatternMatcher(includes, ignores []string, caseSensitive bool) *PatternMatcher {
    return &PatternMatcher{
        includePatterns: includes,
        ignorePatterns:  ignores,
        caseSensitive:   caseSensitive,
    }
}

// ShouldInclude checks if a path should be included based on patterns
func (pm *PatternMatcher) ShouldInclude(path string) bool {
    // Normalize path separators
    path = filepath.ToSlash(path)
    
    if !pm.caseSensitive {
        path = strings.ToLower(path)
    }

    // Check ignore patterns first
    for _, pattern := range pm.ignorePatterns {
        if !pm.caseSensitive {
            pattern = strings.ToLower(pattern)
        }
        matched, _ := doublestar.Match(pattern, path)
        if matched {
            return false
        }
    }

    // If no include patterns, include everything not ignored
    if len(pm.includePatterns) == 0 {
        return true
    }

    // Check include patterns
    for _, pattern := range pm.includePatterns {
        if !pm.caseSensitive {
            pattern = strings.ToLower(pattern)
        }
        matched, _ := doublestar.Match(pattern, path)
        if matched {
            return true
        }
    }

    return false
}

// ParsePatternList parses a comma-separated pattern list
func ParsePatternList(patterns string) []string {
    if patterns == "" {
        return nil
    }

    parts := strings.Split(patterns, ",")
    result := make([]string, 0, len(parts))

    for _, part := range parts {
        part = strings.TrimSpace(part)
        if part != "" {
            result = append(result, part)
        }
    }

    return result
}

// NormalizePatterns normalizes a list of patterns
func NormalizePatterns(patterns []string) []string {
    result := make([]string, 0, len(patterns))
    
    for _, pattern := range patterns {
        // Convert backslashes to forward slashes
        pattern = filepath.ToSlash(pattern)
        
        // Ensure pattern starts with **/ if it doesn't start with /
        if !strings.HasPrefix(pattern, "/") && !strings.HasPrefix(pattern, "**/") {
            pattern = "**/" + pattern
        }
        
        result = append(result, pattern)
    }
    
    return result
}
