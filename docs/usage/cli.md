# Command Line Interface

This document describes the VPR command line interface for running and managing vulnerability PoCs.

## Installation

```bash
# Install from source
go install github.com/your-org/vpr/cmd/vpr@latest

# Or download a pre-built binary from the releases page
```

## Basic Usage

```bash
# Run a PoC file
vpr run path/to/poc.yaml

# Validate a PoC file without running it
vpr validate path/to/poc.yaml

# Get information about a PoC file
vpr info path/to/poc.yaml
```

## Command Reference

### `vpr run`

Executes a PoC definition file.

```bash
vpr run [options] <poc-file>
```

Options:
- `--env-file <file>`: Load environment variables from file
- `--env <key=value>`: Set environment variable
- `--creds-dir <dir>`: Directory for credential files
- `--interactive-creds`: Prompt for credentials interactively
- `--manual-confirm`: Prompt for confirmation before manual steps
- `--report-format <format>`: Output format (json, yaml, text)
- `--report-file <file>`: File to write report to
- `--log-level <level>`: Logging level (debug, info, warn, error)
- `--timeout <seconds>`: Global timeout for execution
- `--skip-setup`: Skip setup phase
- `--skip-verification`: Skip verification phase
- `--steps <step-ids>`: Only run specific steps (comma-separated)
- `--dry-run`: Show what would be run without executing

### `vpr validate`

Validates a PoC definition file against the schema.

```bash
vpr validate [options] <poc-file>
```

Options:
- `--schema <file>`: Custom JSON schema file
- `--strict`: Fail on warnings
- `--format <format>`: Output format (json, yaml, text)

### `vpr info`

Displays information about a PoC definition file.

```bash
vpr info [options] <poc-file>
```

Options:
- `--format <format>`: Output format (json, yaml, text)
- `--detail <level>`: Detail level (basic, full, steps)

### `vpr init`

Creates a new PoC definition file from a template.

```bash
vpr init [options] <output-file>
```

Options:
- `--template <template>`: Template to use (http, idor, xss, etc.)
- `--id <id>`: PoC ID
- `--title <title>`: PoC title

### `vpr convert`

Converts a PoC file between formats.

```bash
vpr convert [options] <input-file> <output-file>
```

Options:
- `--from <format>`: Input format (yaml, json)
- `--to <format>`: Output format (yaml, json)
- `--pretty`: Pretty-print output

## Environment Variables

VPR recognizes the following environment variables:

- `VPR_CONFIG`: Path to configuration file
- `VPR_CREDS_DIR`: Path to credentials directory
- `VPR_LOG_LEVEL`: Default log level
- `VPR_REPORT_FORMAT`: Default report format
- `VPR_TIMEOUT`: Default execution timeout
- `VPR_CRED_*`: Credential values (see Credential Management)

## Configuration File

VPR can be configured using a YAML config file (`~/.vpr/config.yaml`):

```yaml
# ~/.vpr/config.yaml
log_level: info
report_format: text
creds_dir: ~/.vpr/credentials
timeout: 300
credential_providers:
  - type: env
  - type: file
    directory: ~/.vpr/credentials
```

## Examples

```bash
# Run a PoC with environment variables
vpr run --env TARGET_URL=https://example.com examples/idor.yaml

# Validate multiple PoCs
vpr validate path/to/*.yaml

# Run specific steps only
vpr run --steps 3,4,5 complex_poc.yaml

# Run with interactive credential prompting
vpr run --interactive-creds auth_required_poc.yaml
