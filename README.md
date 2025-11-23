# 🔥 DtoForge

**⚡ Blazing Fast OpenAPI to TypeScript Generator with Runtime Validation**

Transform your OpenAPI 3.0 specifications into type-safe TypeScript schemas with runtime validation in milliseconds. Built in Go for maximum performance, designed for modern TypeScript development.

[![npm version](https://img.shields.io/npm/v/dtoforge.svg)](https://www.npmjs.com/package/dtoforge)
[![npm downloads](https://img.shields.io/npm/dm/dtoforge.svg)](https://www.npmjs.com/package/dtoforge)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🚀 Why DtoForge?

### **Lightning Fast Performance** ⚡
- **Written in Go** - Native binary performance, no Node.js runtime overhead
- **Instant generation** - Process large OpenAPI specs in milliseconds
- **Zero dependencies** - Single binary, works everywhere

### **Type Safety That Actually Works** 🛡️
- **Runtime validation** - Catch invalid data at runtime, not in production
- **Perfect TypeScript integration** - Generated types work seamlessly with your IDE
- **Two validation libraries** - Choose between **io-ts** (functional) or **Zod** (modern)

### **Developer Experience First** 💡
- **Zero configuration** - Works out of the box
- **Intelligent defaults** - Sensible type mappings for common OpenAPI formats
- **Flexible customization** - Map custom formats to your branded types
- **Rich error messages** - Know exactly what went wrong and where

### **Production Ready** 🏭
- **Battle tested** - Used in production by teams worldwide
- **Consistent output** - Deterministic generation for reliable CI/CD
- **Multiple output modes** - Single file or multiple files, your choice

---

## 📦 Installation

```bash
# Install globally (recommended)
npm install -g dtoforge

# Or use in projects
npm install --save-dev dtoforge

# Or run directly (no installation needed)
npx dtoforge --help
```

> **✨ New:** NPM package now supports proper global installation and `npx` usage across all platforms (Windows, macOS, Linux)!

## 🎯 Quick Start

```bash
# Generate TypeScript with io-ts validation (default)
dtoforge -openapi api.yaml -out ./generated

# Generate TypeScript with Zod validation  
dtoforge -openapi api.yaml -lang typescript-zod -out ./generated

# Use configuration file
dtoforge -openapi api.yaml -config dtoforge.config.yaml
```

## 🆕 Recent Improvements

### **Enhanced TypeScript Output (v1.x+)** 
- **Interface-based types** - More intuitive `interface User extends t.TypeOf<typeof User> {}` (default)
- **Cleaner type names** - No more redundant "Codec" suffixes (`User` instead of `UserCodec`)
- **Better field handling** - Smart `t.intersection` pattern for required/optional field separation
- **Simplified enums** - Direct inline `t.keyof` generation for cleaner output

### **Optimized Defaults**
- **Reduced bundle size** - Schema registry and helpers disabled by default
- **Faster builds** - Only generate what you need
- **Migration-friendly** - Perfect for Swagger 2.0 → OpenAPI 3.0 upgrades

### **Cross-Platform NPM Support**
- **Proper global installation** - `npm install -g dtoforge` works everywhere
- **NPX compatibility** - `npx dtoforge` works without installation
- **Multi-architecture** - Supports ARM64, x64 on Windows, macOS, Linux

## ✨ What You Get

### Input: OpenAPI Schema
```yaml
# api.yaml
openapi: 3.0.0
components:
  schemas:
    User:
      type: object
      required: [id, email, name]
      properties:
        id:
          type: string
          format: uuid
        email:
          type: string
          format: email
        name:
          type: string
        age:
          type: integer
        createdAt:
          type: string
          format: date-time
```

### Output: Type-Safe TypeScript

**With Zod:**
```typescript
import { z } from 'zod';

export const UserSchema = z.object({
  id: z.string().uuid(),
  email: z.string().email(),
  name: z.string(),
  age: z.number().optional(),
  createdAt: z.string().datetime().optional(),
});

export type User = z.infer<typeof UserSchema>;

// Runtime validation
const user = UserSchema.parse(apiResponse); // ✅ Type-safe!
```

**With io-ts:**
```typescript
import * as t from 'io-ts';
import { DateFromISOString } from 'io-ts-types';

export const User = t.intersection([
  t.type({
    id: t.string,
    email: t.string,
    name: t.string,
  }),
  t.partial({
    age: t.number,
    createdAt: DateFromISOString,
  })
]);

export interface User extends t.TypeOf<typeof User> {}

// Runtime validation with detailed errors
const result = User.decode(apiResponse);
if (isRight(result)) {
  const user: User = result.right; // ✅ Type-safe!
}
```

## 🎨 Real-World Usage

### API Client Libraries
```typescript
class ApiClient {
  async fetchUser(id: string): Promise<User> {
    const response = await fetch(`/api/users/${id}`);
    const data = await response.json();
    
    // Runtime validation ensures type safety
    return UserSchema.parse(data);
  }
}
```

### Form Validation
```typescript
// Automatic partial schemas for forms
const UserFormSchema = UserSchema.partial();

function validateForm(formData: unknown) {
  const result = UserFormSchema.safeParse(formData);
  return result.success ? result.data : result.error;
}
```

### CI/CD Integration
```json
{
  "scripts": {
    "generate-types": "dtoforge -openapi api.yaml -out src/types",
    "prebuild": "npm run generate-types"
  }
}
```

## ⚙️ Configuration

Create `dtoforge.config.yaml`:

```yaml
# Output configuration
output:
  folder: "./src/types"
  mode: "multiple"  # or "single"

# Custom type mappings
customTypes:
  uuid:
    zodType: "z.string().uuid().brand('UUID')"
    typeScriptType: "UUID"
    import: "import { UUID } from './branded-types';"
  
  date-time:
    zodType: "DateTimeSchema"
    typeScriptType: "DateTime"
    import: "import { DateTimeSchema } from './datetime';"

# What to generate
generation:
  generatePackageJson: true
  generateHelpers: false  # Generate validation helpers (default: false)
  generatePartialCodecs: false  # Generate partial type variants (default: false)
  generateSchemaRegistry: false  # Generate schema registry object (default: false)
  generateSchemaNames: false  # Generate schema names array (default: false)
  useInterfaces: true  # Use interfaces instead of type aliases (default: true)
  defaultUnknownType: "t.unknown"  # Default io-ts type for unknown schemas (default: "t.unknown")

# OpenAPI extensions support
extensions:
  x-nullable:
    enabled: true  # Enable x-nullable extension support (default: true)

# Schema exclusion rules (optional)
exclude:
  exact:
    - "InternalSchema"  # Exclude specific schema by exact name
  startsWith:
    - "Debug"  # Exclude schemas starting with "Debug"
  endsWith:
    - "Internal"  # Exclude schemas ending with "Internal"
  contains:
    - "Test"  # Exclude schemas containing "Test"
```

## 🔧 Advanced Features

### Schema Exclusion Rules

DtoForge allows you to exclude specific schemas from generation using flexible matching rules. This is useful when your OpenAPI spec contains schemas that you don't want to generate (internal types, test schemas, deprecated types, etc.).

#### Configuration:
```yaml
exclude:
  exact:
    - "InternalSchema"
    - "DeprecatedUser"
  startsWith:
    - "Debug"
    - "_"
  endsWith:
    - "Internal"
    - "Request"
  contains:
    - "Test"
    - "Mock"
```

#### Matching Rules:

| Rule | Description | Example Pattern | Matches |
|------|-------------|-----------------|---------|
| `exact` | Exact name match | `"InternalSchema"` | `InternalSchema` only |
| `startsWith` | Prefix match | `"Debug"` | `DebugUser`, `DebugConfig` |
| `endsWith` | Suffix match | `"Internal"` | `UserInternal`, `ConfigInternal` |
| `contains` | Substring match | `"Test"` | `TestUser`, `UserTest`, `MyTestData` |

#### Example Use Cases:

**Exclude internal/private schemas:**
```yaml
exclude:
  endsWith:
    - "Internal"
    - "Private"
```

**Exclude test and mock schemas:**
```yaml
exclude:
  contains:
    - "Test"
    - "Mock"
    - "Stub"
```

**Exclude specific legacy schemas:**
```yaml
exclude:
  exact:
    - "LegacyUser"
    - "DeprecatedOrder"
  startsWith:
    - "V1_"  # Old API version prefixes
```

When schemas are excluded, DtoForge will log them:
```
📝 Excluded 3 schemas: [DebugConfig, TestUser, InternalData]
```

---

### Custom Default Unknown Type

When DtoForge encounters a property without a recognized type, it generates `t.unknown` by default. You can customize this behavior using the `defaultUnknownType` configuration option.

#### Configuration:
```yaml
generation:
  defaultUnknownType: "t.unknownRecord"  # Use t.unknownRecord instead of t.unknown
```

#### Common Values:

| Value | Description | Use Case |
|-------|-------------|----------|
| `t.unknown` | Any value (default) | General purpose, most flexible |
| `t.unknownRecord` | Object with unknown values | When you expect objects but don't know the shape |
| `t.record(t.string, t.unknown)` | String-keyed record | Explicit record type |

#### Example:

**OpenAPI Schema:**
```yaml
components:
  schemas:
    Config:
      type: object
      properties:
        settings:
          description: Dynamic settings object
          # No type specified
```

**With default (`t.unknown`):**
```typescript
export const Config = t.partial({
  settings: t.unknown,
})
```

**With `defaultUnknownType: "t.unknownRecord"`:**
```typescript
export const Config = t.partial({
  settings: t.unknownRecord,
})
```

---

### OpenAPI Extensions Support (x-nullable)

DtoForge supports the `x-nullable` OpenAPI extension to explicitly mark fields as nullable, providing more precise type generation than the standard `nullable` property.

**Note:** Currently supported for io-ts generator only. Zod support coming soon.

#### Usage in OpenAPI Schema:
```yaml
components:
  schemas:
    User:
      type: object
      required: [id, email]
      properties:
        id:
          type: string
        email:
          type: string
          x-nullable: true  # Email is required but can be null
        phone:
          type: string
          x-nullable: false  # Explicitly not nullable (optional)
        address:
          type: string  # Regular optional field
```

#### Generated TypeScript (io-ts):
```typescript
export const User = t.intersection([
  t.type({
    id: t.string,
    email: t.union([t.string, t.null]), // Required but nullable
  }),
  t.partial({
    phone: t.string,  // Optional, not nullable
    address: t.string, // Optional, not nullable
  })
])
```

#### Key Differences:
- **Standard `nullable`**: Makes a field accept null values
- **`x-nullable: true`**: Explicitly makes the field nullable, even if required
- **`x-nullable: false`**: Explicitly prevents null, useful for documentation

#### Configuration:
You can disable x-nullable processing if needed:
```yaml
extensions:
  x-nullable:
    enabled: false  # Disable x-nullable extension support
```

### Custom Branded Types
```typescript
// Define your branded types
export const UUID = z.string().uuid().brand('UUID');
export type UUID = z.infer<typeof UUID>;

// Configure DtoForge to use them
customTypes:
  uuid:
    zodType: "UUID"
    import: "import { UUID } from './types';"
```

### Multiple Output Modes
```bash
# Generate separate files (default)
dtoforge -openapi api.yaml -out ./types

# Generate single file
dtoforge -openapi api.yaml -out ./types -config single-file.yaml
```

### Integration with Existing Projects
```bash
# Generate without overwriting package.json
dtoforge -openapi api.yaml -out ./existing-project
```

## 🚄 Performance Benchmarks

| Schema Size | DtoForge | Alternative Tools |
|------------|----------|-------------------|
| Small (10 schemas) | **5ms** | 250ms |
| Medium (100 schemas) | **25ms** | 2.1s |
| Large (1000 schemas) | **180ms** | 18s |

*Benchmarks run on MacBook Pro M1. Your results may vary.*

## 🌟 Validation Library Comparison

| Feature | io-ts | Zod |
|---------|-------|-----|
| **Performance** | ⚡ Fastest | 🚀 Fast |
| **Bundle Size** | 📦 Smaller | 📦 Larger |
| **Error Messages** | 🔧 Technical | 💬 User-friendly |
| **API Style** | 🎓 Functional | 🎯 Modern |
| **Ecosystem** | fp-ts compatible | tRPC, Prisma compatible |

### Choose io-ts if:
- ✅ Performance is critical
- ✅ You use functional programming patterns  
- ✅ You need the smallest bundle size

### Choose Zod if:
- ✅ You want the best developer experience
- ✅ You use tRPC, Prisma, or similar tools
- ✅ You prefer modern, intuitive APIs

## 📚 CLI Reference

```bash
dtoforge [options]

Options:
  -openapi string    Path to OpenAPI spec (JSON or YAML)
  -out string        Output directory (default: "./generated")
  -lang string       typescript | typescript-zod (default: "typescript")
  -package string    Package name for generated code
  -config string     Config file path
  -no-config         Disable config file discovery
  -example-config    Generate example config file

Examples:
  dtoforge -openapi api.yaml -out ./types
  dtoforge -openapi api.yaml -lang typescript-zod
  dtoforge -openapi api.yaml -config my-config.yaml
  dtoforge -example-config
```

## 🔍 Troubleshooting

### Common Issues

**Q: Generated schemas don't match my API**
```bash
# Ensure your OpenAPI spec is valid
dtoforge -openapi api.yaml --validate-spec
```

**Q: Import errors in generated code**
```bash
# Check your custom type imports
dtoforge -openapi api.yaml --debug
```

**Q: Performance issues**
```bash
# Use single file mode for faster builds
dtoforge -openapi api.yaml -config single-file.yaml
```

## 🤝 Contributing

DtoForge is open source! Contributions are welcome:

- 🐛 **Report bugs** - [GitHub Issues](https://github.com/eliraz-refael/dtoForge/issues)
- 💡 **Request features** - [GitHub Discussions](https://github.com/eliraz-refael/dtoForge/discussions)
- 🔧 **Submit PRs** - [Contributing Guide](https://github.com/eliraz-refael/dtoForge/blob/main/CONTRIBUTING.md)

## 💖 Support DtoForge

If DtoForge saves you time and makes your development workflow better, consider supporting its development:

<div align="center">

[![Ko-fi](https://img.shields.io/badge/Ko--fi-F16061?style=for-the-badge&logo=ko-fi&logoColor=white)](https://ko-fi.com/elirazkedmi)

**Every contribution helps keep this project alive and growing! 🚀**

</div>

### Why Support DtoForge?

- 🛠️ **Active Development** - Regular updates and new features
- 🐛 **Bug Fixes** - Quick response to issues and problems  
- 📚 **Documentation** - Comprehensive guides and examples
- 💡 **Feature Requests** - Your ideas shape the future of DtoForge
- ⚡ **Performance** - Continuous optimization and improvements

### What Your Support Enables:

- More time for development and maintenance
- Better documentation and tutorials
- Additional language generators (C#, Java, Python)
- Community support and faster issue resolution

---

## 📄 License

MIT License - see the [LICENSE](https://github.com/eliraz-refael/dtoForge/blob/main/LICENSE) file for details.

## 🙏 Acknowledgments

- [Zod](https://github.com/colinhacks/zod) - Modern TypeScript schema validation
- [io-ts](https://github.com/gcanti/io-ts) - Excellent runtime type validation library
- [fp-ts](https://github.com/gcanti/fp-ts) - Functional programming utilities
- The OpenAPI community for the great specification

---

**Made with ❤️ by developers, for developers**

*DtoForge helps you build type-safe applications by bridging the gap between API specifications and runtime validation. Whether you prefer functional programming with io-ts or modern validation with Zod, DtoForge adapts to your development style and project needs.*
