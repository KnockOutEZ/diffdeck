package utils

import (
    "path/filepath"
    "strings"
)

func ParsePatternList(patterns string) []string {
    if patterns == "" {
        return nil
    }

    var result []string
    for _, p := range strings.Split(patterns, ",") {
        if pattern := strings.TrimSpace(p); pattern != "" {
            result = append(result, pattern)
        }
    }
    return result
}

func MatchesAny(path string, patterns []string) bool {
    if len(patterns) == 0 {
        return false
    }

    path = filepath.Clean(path)
    for _, pattern := range patterns {
        if matched, _ := filepath.Match(pattern, path); matched {
            return true
        }
        // Handle **/ pattern
        if strings.HasPrefix(pattern, "**/") {
            if matched, _ := filepath.Match(pattern[3:], path); matched {
                return true
            }
        }
    }
    return false
}
