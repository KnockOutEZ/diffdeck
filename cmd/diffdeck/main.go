package main

import (
    "flag"
    "fmt"
    "os"

    "github.com/KnockOutEZ/diffdeck/internal/config"
    "github.com/KnockOutEZ/diffdeck/internal/formatter"
    "github.com/KnockOutEZ/diffdeck/internal/git"
    "github.com/KnockOutEZ/diffdeck/internal/scanner"
    "github.com/KnockOutEZ/diffdeck/internal/security"
    "github.com/KnockOutEZ/diffdeck/internal/utils"
)

var (
    version = "1.0.0"

    // Command line flags
    configPath       string
    outputPath      string
    outputStyle     string
    includePatterns string
    ignorePatterns  string
    remoteURL       string
    remoteBranch    string
    showVersion     bool
    initConfig      bool
    topFilesLen     int
    showLineNumbers bool
    copyToClipboard bool
    noSecurityCheck bool
    verbose         bool
)

func init() {
    // Basic flags
    flag.StringVar(&configPath, "config", "", "Path to config file")
    flag.StringVar(&outputPath, "output", "", "Output file path")
    flag.StringVar(&outputStyle, "style", "", "Output style (plain, xml, markdown)")
    flag.StringVar(&includePatterns, "include", "", "Include patterns (comma-separated)")
    flag.StringVar(&ignorePatterns, "ignore", "", "Ignore patterns (comma-separated)")
    
    // Git-related flags
    flag.StringVar(&remoteURL, "remote", "", "Remote repository URL")
    flag.StringVar(&remoteBranch, "remote-branch", "", "Remote branch, tag, or commit")
    
    // Other flags
    flag.BoolVar(&showVersion, "version", false, "Show version")
    flag.BoolVar(&initConfig, "init", false, "Initialize config file")
    flag.IntVar(&topFilesLen, "top-files-len", 0, "Number of top files to display")
    flag.BoolVar(&showLineNumbers, "output-show-line-numbers", false, "Show line numbers")
    flag.BoolVar(&copyToClipboard, "copy", false, "Copy output to clipboard")
    flag.BoolVar(&noSecurityCheck, "no-security-check", false, "Disable security check")
    flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")

    // Add short versions for common flags
    flag.StringVar(&outputPath, "o", "", "Output file path (shorthand)")
    flag.StringVar(&configPath, "c", "", "Config file path (shorthand)")
    flag.StringVar(&ignorePatterns, "i", "", "Ignore patterns (shorthand)")
    flag.BoolVar(&showVersion, "v", false, "Show version (shorthand)")
}

func main() {
    flag.Parse()

    if err := run(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}

func run() error {
    // Show version if requested
    if showVersion {
        fmt.Printf("diffdeck version %s\n", version)
        return nil
    }

    // Initialize config if requested
    if initConfig {
        return initializeConfig()
    }

    // Load configuration
    cfg, err := loadConfig()
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }

    // Apply command line overrides
    applyCommandLineOverrides(cfg)

    // Handle remote repository if specified
    var files []scanner.File
    if remoteURL != "" {
        files, err = processRemoteRepository()
    } else {
        files, err = processLocalFiles(cfg)
    }
    if err != nil {
        return err
    }

    // Run security check if enabled
    if !cfg.Security.EnableSecurityCheck {
        if err := runSecurityCheck(files); err != nil {
            return err
        }
    }

    // Format output
    output, err := formatOutput(files, cfg)
    if err != nil {
        return err
    }

    // Write output
    if err := writeOutput(output, cfg); err != nil {
        return err
    }

    return nil
}

func loadConfig() (*config.Config, error) {
    if configPath == "" {
        configPath = "diffdeck.config.json"
    }
    return config.Load(configPath)
}

func applyCommandLineOverrides(cfg *config.Config) {
    if outputPath != "" {
        cfg.Output.FilePath = outputPath
    }
    if outputStyle != "" {
        cfg.Output.Style = outputStyle
    }
    if includePatterns != "" {
        cfg.Include = utils.ParsePatternList(includePatterns)
    }
    if ignorePatterns != "" {
        cfg.Ignore.CustomPatterns = utils.ParsePatternList(ignorePatterns)
    }
    if topFilesLen > 0 {
        cfg.Output.TopFilesLength = topFilesLen
    }
    if showLineNumbers {
        cfg.Output.ShowLineNumbers = true
    }
    if copyToClipboard {
        cfg.Output.CopyToClipboard = true
    }
    if noSecurityCheck {
        cfg.Security.EnableSecurityCheck = false
    }
}

func initializeConfig() error {
    cfg := config.DefaultConfig
    return cfg.Save("diffdeck.config.json")
}

func processRemoteRepository() ([]scanner.File, error) {
    opts := git.CloneOptions{
        Branch:   remoteBranch,
        Progress: os.Stderr,
    }

    repo, err := git.New(remoteURL, opts)
    if err != nil {
        return nil, fmt.Errorf("failed to create repository: %w", err)
    }
    defer repo.Close()

    if err := repo.Clone(opts); err != nil {
        return nil, fmt.Errorf("failed to clone repository: %w", err)
    }

    changes, err := repo.GetChanges(git.DiffOptions{})
    if err != nil {
        return nil, fmt.Errorf("failed to get changes: %w", err)
    }

    // Convert git.FileChange to scanner.File
    var files []scanner.File
    for _, change := range changes {
        files = append(files, scanner.File{
            Path:    change.Path,
            Content: change.Content,
        })
    }

    return files, nil
}

func processLocalFiles(cfg *config.Config) ([]scanner.File, error) {
    paths := flag.Args()
    if len(paths) == 0 {
        paths = []string{"."}
    }

    s, err := scanner.New(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create scanner: %w", err)
    }

    return s.Scan(paths)
}

func runSecurityCheck(files []scanner.File) error {
    checker, err := security.New(nil)
    if err != nil {
        return fmt.Errorf("failed to create security checker: %w", err)
    }

    issues, err := checker.Check(files)
    if err != nil {
        return fmt.Errorf("security check failed: %w", err)
    }

    if len(issues) > 0 {
        report, err := checker.CreateReport(issues, "text")
        if err != nil {
            return fmt.Errorf("failed to create security report: %w", err)
        }
        fmt.Fprintln(os.Stderr, report)
    }

    return nil
}

func formatOutput(files []scanner.File, cfg *config.Config) (string, error) {
    f := formatter.NewFormatter(cfg.Output.Style)
    
    formatCfg := formatter.FormatConfig{
        HeaderText:       cfg.Output.HeaderText,
        ShowFileSummary:  cfg.Output.FileSummary,
        ShowDirStructure: cfg.Output.DirectoryStructure,
        ShowLineNumbers:  cfg.Output.ShowLineNumbers,
        TopFilesLength:   cfg.Output.TopFilesLength,
    }

    // Load instruction text if specified
    if cfg.Output.InstructionFilePath != "" {
        instruction, err := os.ReadFile(cfg.Output.InstructionFilePath)
        if err != nil {
            return "", fmt.Errorf("failed to read instruction file: %w", err)
        }
        formatCfg.InstructionText = string(instruction)
    }

    return f.Format(files, formatCfg)
}

func writeOutput(output string, cfg *config.Config) error {
    // Write to file
    if err := os.WriteFile(cfg.Output.FilePath, []byte(output), 0644); err != nil {
        return fmt.Errorf("failed to write output file: %w", err)
    }

    // Copy to clipboard if requested
    if cfg.Output.CopyToClipboard {
        if err := utils.CopyToClipboard(output); err != nil {
            return fmt.Errorf("failed to copy to clipboard: %w", err)
        }
    }

    return nil
}

func logVerbose(format string, args ...interface{}) {
    if verbose {
        fmt.Fprintf(os.Stderr, format+"\n", args...)
    }
}