# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**DtoForge** is a high-performance OpenAPI to TypeScript generator written in Go. It converts OpenAPI 3.0 specifications into type-safe TypeScript schemas with runtime validation, supporting both io-ts (functional) and Zod (modern) validation libraries.

## Key Commands

### Building and Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverage ./...

# Run golden tests (integration tests with expected outputs)
go test -run TestGolden

# Run specific test
go test -run TestSpecificFunction ./internal/typescript

# Build the binary
go build -o dtoForge main.go
```

### Development Workflow
```bash
# Generate TypeScript with io-ts (default)
./dtoForge -openapi api.yaml -out ./generated

# Generate TypeScript with Zod
./dtoForge -openapi api.yaml -lang typescript-zod -out ./generated

# Use configuration file
./dtoForge -openapi api.yaml -config dtoforge.config.yaml

# Generate example config
./dtoForge -example-config
```

### NPM Package Testing and Publishing
```bash
# Test the npm package locally BEFORE publishing
./scripts/npm/test-package.sh

# The npm-publish workflow handles versioning and publishing (from CI/CD)
# Uses scripts/npm/package.json.template to generate package.json
# Downloads binaries from GitHub releases (names MUST match release.yaml output)
```

## Architecture

### Core Components

**main.go**: Entry point that handles CLI arguments, config discovery, OpenAPI parsing, and orchestrates code generation. Key functions:
- `parseCLIArgs()`: Processes command-line arguments
- `discoverConfigFile()`: Implements config file discovery logic (current dir → OpenAPI dir → binary dir)
- `convertToGeneratorDTOs()`: Transforms OpenAPI schemas to internal IR (Intermediate Representation)

**internal/generator/**: Core abstractions and types
- `generator.go`: Defines the Generator interface and Registry pattern for language generators
- `types.go`: IR type system (PrimitiveType, ArrayType, ObjectType, EnumType, ReferenceType)
- DTOs are sorted alphabetically, properties within DTOs are sorted for consistent output

**internal/typescript/**: io-ts generator implementation
- `generator.go`: Main TypeScript generator logic
- `templates.go`: io-ts code templates (codec generation, imports, helpers)
- `custom_types.go`: CustomTypeRegistry for handling OpenAPI format mappings (uuid, date-time, email, etc.)

**internal/zod/**: Zod generator implementation
- `generator.go`: Main Zod generator logic  
- `templates.go`: Zod schema generation templates
- `custom-types.go`: Zod-specific custom type mappings

### Key Design Patterns

1. **Registry Pattern**: All generators register themselves with a central registry, allowing easy extension
2. **Intermediate Representation (IR)**: OpenAPI schemas are converted to a language-agnostic IR before generation
3. **Template-based Generation**: Each generator uses templates for consistent code output
4. **Custom Type Mapping**: Extensible system for mapping OpenAPI formats to TypeScript branded types

### Testing Structure

**testdata/**: Contains test fixtures
- `basic-api.yaml`: Simple OpenAPI spec for testing
- `formats-api.yaml`: Tests custom format handling
- `golden/`: Expected outputs for golden testing

**Golden Tests**: Integration tests that compare actual output against expected files in testdata/golden/

## Configuration System

The config file (`dtoforge.config.yaml`) supports:
- Output folder and mode (single/multiple files)
- Custom type mappings for OpenAPI formats
- Generation options (package.json, partial codecs, helpers)

Custom types define:
- `ioTsType`: Type name for io-ts generator
- `zodType`: Type expression for Zod generator  
- `typeScriptType`: TypeScript type annotation
- `import`: Required import statement

## Important Implementation Details

1. **Consistent Output**: Properties and schemas are explicitly sorted alphabetically to ensure deterministic output
2. **Config Discovery**: Searches in order: current dir → OpenAPI file dir → binary dir
3. **Format Handling**: Built-in support for common formats (uuid, date-time, email, uri, binary)
4. **Error Handling**: Comprehensive error messages with context for debugging
5. **No External Dependencies**: Only uses `gopkg.in/yaml.v3` for YAML parsing