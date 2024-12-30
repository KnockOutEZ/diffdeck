package config

var DefaultConfig = Config{
    Output: OutputConfig{
        FilePath:            "diffdeck-output.txt",
        Style:              "plain",
        FileSummary:        true,
        DirectoryStructure: true,
        RemoveComments:     false,
        RemoveEmptyLines:   false,
        ShowLineNumbers:    false,
        CopyToClipboard:    false,
        TopFilesLength:     5,
        IncludeEmptyDirs:   false,
    },
    Include: []string{"**/*"},
    Ignore: IgnoreConfig{
        UseGitignore:       true,
        UseDefaultPatterns: true,
        CustomPatterns:     []string{},
    },
    Security: SecurityConfig{
        EnableSecurityCheck: true,
    },
}

var DefaultIgnorePatterns = []string{
    ".git/**",
    "node_modules/**",
    "*.log",
    "*.tmp",
    "*.temp",
    ".DS_Store",
    "Thumbs.db",
}
