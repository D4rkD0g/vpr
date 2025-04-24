# vulnerability proof-of-concept runner

Work in progress by LLM...

A modular Proof-of-Concept (PoC) runner for security testing, supporting YAML-defined PoCs (DSL v1.0).

## Directory Structure

- `cmd/vpr/`: CLI entry point
- `pkg/`: Core logic, organized by responsibility
- `examples/`: Example PoC YAML files
- `docs/`: Documentation

## Getting Started

1. Build the CLI: `go build ./cmd/vpr`
2. Run with a PoC YAML: `./vpr -p examples/example-v1.yaml`

## Development
Each Go file contains a comment describing its intended purpose. See `pkg/` for details on each module.

## License
MIT
