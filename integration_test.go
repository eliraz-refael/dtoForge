package main

import (
	"path/filepath"
	"testing"

	"dtoForge/internal/generator"
	"dtoForge/internal/testutils"
	"dtoForge/internal/typescript"
)

func TestGenerateTypeScriptFromOpenAPI(t *testing.T) {
	tests := []struct {
		name        string
		openAPISpec string
		config      string
		wantFiles   []string
		wantContent map[string][]string // file -> expected content snippets
	}{
		{
			name: "Basic schema generation",
			openAPISpec: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      required:
        - id
        - name
      properties:
        id:
          type: string
        name:
          type: string
        email:
          type: string
          format: email
`,
			wantFiles: []string{"index.ts", "user.ts", "package.json"},
			wantContent: map[string][]string{
				"user.ts": {
					"export const UserCodec = t.type({",
					"id: t.string,",
					"name: t.string,",
					"email: t.union([t.string, t.undefined]),",
					"export type User = t.TypeOf<typeof UserCodec>;",
				},
				"index.ts": {
					"export * from './user';",
					"import * as t from 'io-ts';",
				},
			},
		},
		{
			name: "Custom format mapping",
			openAPISpec: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      required:
        - id
      properties:
        id:
          type: string
          format: uuid
        createdAt:
          type: string
          format: date-time
`,
			config: `
customTypes:
  uuid:
    ioTsType: "UUID"
    typeScriptType: "UUID"
    import: "import { UUID } from './branded-types';"
  date-time:
    ioTsType: "DateTimeString"
    typeScriptType: "DateTimeString"
    import: "import { DateTimeString } from './branded-types';"
`,
			wantFiles: []string{"user.ts"},
			wantContent: map[string][]string{
				"user.ts": {
					"import { UUID } from './branded-types';",
					"import { DateTimeString } from './branded-types';",
					"id: UUID,",
					"createdAt: t.union([DateTimeString, t.undefined]),",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			tempDir := testutils.TempDir(t)
			outputDir := filepath.Join(tempDir, "output")

			openAPIPath := testutils.WriteFile(t, tempDir, "api.yaml", tt.openAPISpec)

			var configPath string
			if tt.config != "" {
				configPath = testutils.WriteFile(t, tempDir, "dtoforge.config.yaml", tt.config)
			}

			// Parse OpenAPI spec
			spec, err := readOpenAPISpec(openAPIPath)
			if err != nil {
				t.Fatalf("Failed to read OpenAPI spec: %v", err)
			}

			// Convert to DTOs
			dtos, err := convertToGeneratorDTOs(spec)
			if err != nil {
				t.Fatalf("Failed to convert to DTOs: %v", err)
			}

			// Generate TypeScript code
			tsGen := typescript.NewTypeScriptGenerator()
			genConfig := generator.Config{
				OutputFolder:   outputDir,
				PackageName:    "test-package",
				TargetLanguage: "typescript",
				ConfigFile:     configPath,
			}

			if err := tsGen.Generate(dtos, genConfig); err != nil {
				t.Fatalf("Failed to generate code: %v", err)
			}

			// Verify expected files were created
			for _, expectedFile := range tt.wantFiles {
				testutils.AssertFileExists(t, filepath.Join(outputDir, expectedFile))
			}

			// Verify file contents
			for filename, expectedSnippets := range tt.wantContent {
				filePath := filepath.Join(outputDir, filename)
				for _, snippet := range expectedSnippets {
					testutils.AssertFileContains(t, filePath, snippet)
				}
			}
		})
	}
}
