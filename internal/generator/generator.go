package generator

import (
	"fmt"
)

// DTO represents a Data Transfer Object in our IR.
type DTO struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Properties  []Property        `json:"properties"`
	Required    []string          `json:"required"`
	Type        string            `json:"type"` // object, enum, etc.
	EnumValues  []string          `json:"enumValues,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Property represents a field within a DTO.
type Property struct {
	Name          string            `json:"name"`
	Type          IRType            `json:"type"`
	Description   string            `json:"description"`
	Nullable      bool              `json:"nullable"`
	Required      bool              `json:"required"`
	CustomBranded string            `json:"customBranded,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// IRType is an interface for our type representations.
type IRType interface {
	TypeName() string
}

// Config holds generation configuration
type Config struct {
	OutputFolder   string
	PackageName    string
	TargetLanguage string
	ConfigFile     string // Path to the custom types config file
}

// Generator is the interface that all language generators must implement
type Generator interface {
	Generate(dtos []DTO, config Config) error
	Language() string
	FileExtension() string
}

// Registry holds all available generators
type Registry struct {
	generators map[string]Generator
}

// NewRegistry creates a new generator registry
func NewRegistry() *Registry {
	return &Registry{
		generators: make(map[string]Generator),
	}
}

// Register adds a generator to the registry
func (r *Registry) Register(gen Generator) {
	r.generators[gen.Language()] = gen
}

// Get retrieves a generator by language name
func (r *Registry) Get(language string) (Generator, error) {
	gen, exists := r.generators[language]
	if !exists {
		return nil, fmt.Errorf("no generator found for language: %s", language)
	}
	return gen, nil
}

// Available returns all available language generators
func (r *Registry) Available() []string {
	var languages []string
	for lang := range r.generators {
		languages = append(languages, lang)
	}
	return languages
}
