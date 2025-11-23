# Task Runner Quick Reference

This project uses [Task](https://taskfile.dev) for build automation.

## Installation

```bash
# macOS
brew install go-task/tap/go-task

# Or download from https://taskfile.dev/installation/
```

## Common Commands

### Building

```bash
task build              # Build for current platform
task build:all          # Build for all platforms (Linux, macOS, Windows)
task build:wasm         # Build WASM binary
task install            # Install to $GOPATH/bin
```

### Testing

```bash
task test               # Run all tests
task test:short         # Run tests (skip slow ones)
task test:coverage      # Generate coverage report
task test:lexer         # Test lexer only
task test:parser        # Test parser only
task test:interpreter   # Test interpreter only
```

### Development

```bash
task dev                # Run in dev mode
task dev:repl           # Start REPL
task example:dates      # Run date examples
task example:units      # Run unit examples
```

### Quality

```bash
task lint               # Run linters
task bench              # Run benchmarks
task clean              # Clean build artifacts
```

### Release

```bash
task release            # Build all platforms + WASM
```

## Build Outputs

- Current platform: `./cm`
- All platforms: `dist/cm-{os}-{arch}`
- WASM: `dist/calcmark.wasm`

## WASM Build Tags

The WASM build uses `-tags wasm` to exclude TUI and platform-specific code.

See `Taskfile.yml` for all available tasks.
