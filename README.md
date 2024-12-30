# DiffDeck

DiffDeck is a powerful, flexible code difference analysis tool that helps developers understand, document, and secure their code changes. It provides rich diff visualization, security scanning, and various output formats to suit different needs.

[![Go Report Card](https://goreportcard.com/badge/github.com/KnockOutEZ/diffdeck)](https://goreportcard.com/report/github.com/KnockOutEZ/diffdeck)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- 🔍 **Smart Diff Analysis**: Compare branches, commits, or local files with intelligent diff algorithms
- 🛡️ **Security Scanning**: Built-in security checks for sensitive data and potential vulnerabilities
- 📊 **Multiple Output Formats**: Support for plain text, Markdown, MDX, and XML outputs
- 🎨 **Customizable Styling**: Line numbers, custom headers, and formatting options
- 🚀 **Performance Optimized**: Efficient processing of large codebases
- 🔒 **Security First**: Automatic scanning for sensitive data and security issues
- 📋 **Clipboard Integration**: Direct copying of diff output to clipboard

## Installation

```bash
go install github.com/KnockOutEZ/diffdeck/cmd/diffdeck@latest
```

Or build from source:

```bash
git clone https://github.com/KnockOutEZ/diffdeck.git
cd diffdeck
go build ./cmd/diffdeck
```

## Quick Start

1. Compare two branches:
```bash
diffdeck --from-branch develop --to-branch main
```

2. Compare with custom output:
```bash
diffdeck --from-branch develop --to-branch main --output diff.md --style markdown
```

3. Enable security scanning:
```bash
diffdeck --from-branch develop --to-branch main --security-check
```

## Configuration

DiffDeck can be configured via JSON configuration file. Default location is `diffdeck.config.json`.

### Basic Configuration Example:

```json
{
  "output": {
    "filePath": "diffdeck-output.txt",
    "style": "mdx",
    "showLineNumbers": true,
    "copyToClipboard": true,
    "topFilesLength": 5
  },
  "include": ["**/*"],
  "ignore": {
    "patterns": [
      ".git/**",
      "node_modules/**",
      "vendor/**"
    ]
  },
  "security": {
    "disableSecurityCheck": false,
    "maxFileSize": 10485760
  }
}
```

### Configuration Options Explained

#### Output Configuration
```json
"output": {
  "filePath": "diffdeck-output.txt",  // Output file path
  "style": "mdx",                     // Output format: plain, markdown, mdx, xml
  "showLineNumbers": true,            // Include line numbers in diff
  "copyToClipboard": true,           // Copy output to clipboard
  "topFilesLength": 5                // Number of files in summary
}
```

#### File Patterns
```json
"include": ["**/*"],              // Files to include
"ignore": {
  "patterns": [                   // Files to ignore
    ".git/**",
    "node_modules/**",
    "vendor/**"
  ]
}
```

#### Security Settings
```json
"security": {
  "disableSecurityCheck": false,  // Enable/disable security scanning
  "maxFileSize": 10485760        // Max file size to scan (10MB)
}
```

#### Git Settings
```json
"git": {
  "defaultRemote": "",           // Default remote repository
  "cacheDir": "/tmp/cache",      // Cache directory location
  "timeout": "5m"               // Git operation timeout
}
```

## Command Line Usage

### Basic Commands

```bash
# Compare branches
diffdeck --from-branch feature --to-branch main

# Use specific config file
diffdeck --config my-config.json --from-branch feature --to-branch main

# Generate markdown output
diffdeck --style markdown --output diff.md --from-branch feature --to-branch main

# Disable security checks
diffdeck --no-security-check --from-branch feature --to-branch main

# Show line numbers
diffdeck --show-line-numbers --from-branch feature --to-branch main
```

### Advanced Usage

```bash
# Custom include/ignore patterns
diffdeck --include "src/**/*.js" --ignore "**/*.test.js"

# Set custom file size limit
diffdeck --max-file-size 20971520

# Compare specific files
diffdeck --files "src/main.js,src/utils.js"

# Use custom cache directory
diffdeck --cache-dir "/custom/cache/path"
```

## Output Formats

### Plain Text
```bash
diffdeck --style plain
```
Generates simple text output with basic formatting.

### Markdown
```bash
diffdeck --style markdown
```
Generates GitHub-flavored Markdown with syntax highlighting.

### MDX
```bash
diffdeck --style mdx
```
Generates MDX format suitable for React documentation.

### XML
```bash
diffdeck --style xml
```
Generates structured XML output for programmatic processing.

## Security Scanning

DiffDeck includes built-in security scanning capabilities:

- Sensitive data detection (API keys, tokens)
- Password pattern matching
- Private key detection
- Internal URL/IP detection
- AWS credentials scanning

Enable/disable security scanning:
```bash
# Enable
diffdeck --security-check

# Disable
diffdeck --no-security-check
```

## File Pattern Syntax

DiffDeck uses glob patterns for file matching:

- `**/*` - Match all files
- `*.{js,ts}` - Match JavaScript and TypeScript files
- `src/**/*.go` - Match Go files in src directory
- `!test/**` - Exclude test directory

## Performance Tips

1. Use appropriate `maxFileSize` limit
2. Leverage ignore patterns for large directories
3. Use cache directory for repeated operations
4. Consider disabling security checks for large diffs
5. Use specific include patterns when possible

## Integration Examples

### GitHub Actions
```yaml
- name: Run DiffDeck
  run: |
    diffdeck --from-branch ${{ github.event.pull_request.base.ref }} \
             --to-branch ${{ github.event.pull_request.head.ref }} \
             --output diff.md \
             --style markdown
```

### GitLab CI
```yaml
diff_check:
  script:
    - diffdeck --from-branch $CI_MERGE_REQUEST_TARGET_BRANCH_NAME \
               --to-branch $CI_MERGE_REQUEST_SOURCE_BRANCH_NAME \
               --style markdown \
               --output diff.md
```

## Common Issues and Solutions

1. **Large Files**
   - Increase `maxFileSize` in config
   - Use more specific include patterns

2. **Performance Issues**
   - Optimize ignore patterns
   - Use cache directory
   - Consider disabling security checks

3. **Git Integration**
   - Ensure correct branch names
   - Check timeout settings
   - Verify git credentials

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

MIT License - see LICENSE file for details.