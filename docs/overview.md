# VPR - Vulnerability Proof Runner

VPR is a Go-based execution engine for security vulnerability PoCs defined in the PoC DSL v1.0 specification.

## Purpose

VPR enables security professionals to:
- Document complex security vulnerabilities in a standardized, machine-readable format
- Automate vulnerability verification across environments
- Create a library of reproducible security tests
- Share and collaborate on vulnerability research

## Core Features

- Parse and validate PoC files (YAML)
- Manage execution context with variable substitution
- Support HTTP-based exploits with extensibility for other protocols
- Extract and transform data during execution
- Generate detailed reports on execution results

## Architecture Overview

VPR is built around these key components:

- **Parser**: Loads and validates PoC files
- **Context Manager**: Handles variables and substitution
- **Executor**: Controls execution flow through phases
- **Actions/Checks**: Implements operations and verifications
- **Extractors**: Processes response data
- **Reporter**: Generates output on results

## Key Concepts

- **BDD Approach**: Given (setup) → When (exploit) → Then (assertions)
- **Context Management**: Dynamic variables with substitution
- **Phases**: Setup → Exploit → Assertions → Verification
- **Security**: No direct credential storage
