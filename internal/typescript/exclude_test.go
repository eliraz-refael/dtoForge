package typescript

import (
	"testing"
)

func TestExcludeConfig_ShouldExclude_ExactMatch(t *testing.T) {
	config := &ExcludeConfig{
		Exact: []string{"ExactMatch", "AnotherExact"},
	}

	tests := []struct {
		name     string
		schema   string
		expected bool
	}{
		{"exact match first", "ExactMatch", true},
		{"exact match second", "AnotherExact", true},
		{"no match", "NoMatch", false},
		{"partial match should not exclude", "ExactMatchSuffix", false},
		{"prefix match should not exclude", "PrefixExactMatch", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.ShouldExclude(tt.schema)
			if result != tt.expected {
				t.Errorf("ShouldExclude(%q) = %v, want %v", tt.schema, result, tt.expected)
			}
		})
	}
}

func TestExcludeConfig_ShouldExclude_StartsWith(t *testing.T) {
	config := &ExcludeConfig{
		StartsWith: []string{"Internal", "Debug"},
	}

	tests := []struct {
		name     string
		schema   string
		expected bool
	}{
		{"starts with Internal", "InternalSchema", true},
		{"starts with Debug", "DebugHelper", true},
		{"exact match Internal", "Internal", true},
		{"no match", "PublicSchema", false},
		{"contains but not prefix", "MyInternalSchema", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.ShouldExclude(tt.schema)
			if result != tt.expected {
				t.Errorf("ShouldExclude(%q) = %v, want %v", tt.schema, result, tt.expected)
			}
		})
	}
}

func TestExcludeConfig_ShouldExclude_EndsWith(t *testing.T) {
	config := &ExcludeConfig{
		EndsWith: []string{"Temp", "Draft"},
	}

	tests := []struct {
		name     string
		schema   string
		expected bool
	}{
		{"ends with Temp", "MyTemp", true},
		{"ends with Draft", "UserDraft", true},
		{"exact match Temp", "Temp", true},
		{"no match", "TempData", false},
		{"contains but not suffix", "TempSchema", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.ShouldExclude(tt.schema)
			if result != tt.expected {
				t.Errorf("ShouldExclude(%q) = %v, want %v", tt.schema, result, tt.expected)
			}
		})
	}
}

func TestExcludeConfig_ShouldExclude_Contains(t *testing.T) {
	config := &ExcludeConfig{
		Contains: []string{"Private", "Legacy"},
	}

	tests := []struct {
		name     string
		schema   string
		expected bool
	}{
		{"contains Private", "MyPrivateData", true},
		{"contains Legacy", "LegacySystem", true},
		{"contains in middle", "UserPrivateInfo", true},
		{"exact match", "Private", true},
		{"no match", "PublicSchema", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.ShouldExclude(tt.schema)
			if result != tt.expected {
				t.Errorf("ShouldExclude(%q) = %v, want %v", tt.schema, result, tt.expected)
			}
		})
	}
}

func TestExcludeConfig_ShouldExclude_Combined(t *testing.T) {
	config := &ExcludeConfig{
		Exact:      []string{"ExactMatch"},
		StartsWith: []string{"Internal"},
		EndsWith:   []string{"Temp"},
		Contains:   []string{"Private"},
	}

	tests := []struct {
		name     string
		schema   string
		expected bool
	}{
		{"exact match", "ExactMatch", true},
		{"starts with", "InternalData", true},
		{"ends with", "UserTemp", true},
		{"contains", "MyPrivateData", true},
		{"no match", "PublicSchema", false},
		{"multiple rules match", "InternalPrivateDataTemp", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.ShouldExclude(tt.schema)
			if result != tt.expected {
				t.Errorf("ShouldExclude(%q) = %v, want %v", tt.schema, result, tt.expected)
			}
		})
	}
}

func TestExcludeConfig_ShouldExclude_Nil(t *testing.T) {
	var config *ExcludeConfig = nil

	result := config.ShouldExclude("AnySchema")
	if result != false {
		t.Errorf("Nil config should not exclude any schema, got %v", result)
	}
}

func TestExcludeConfig_ShouldExclude_Empty(t *testing.T) {
	config := &ExcludeConfig{
		Exact:      []string{},
		StartsWith: []string{},
		EndsWith:   []string{},
		Contains:   []string{},
	}

	tests := []string{"Schema1", "Schema2", "InternalSchema"}

	for _, schema := range tests {
		result := config.ShouldExclude(schema)
		if result != false {
			t.Errorf("Empty config should not exclude schema %q, got %v", schema, result)
		}
	}
}

func TestExcludeConfig_ShouldExclude_EdgeCases(t *testing.T) {
	config := &ExcludeConfig{
		Exact:      []string{"", "EmptyTest"},
		StartsWith: []string{"Start"},
		EndsWith:   []string{"End"},
		Contains:   []string{"Middle"},
	}

	tests := []struct {
		name     string
		schema   string
		expected bool
	}{
		{"empty string schema", "", true},       // Matches empty exact rule
		{"empty string in name", "Test", false}, // Should not match
		{"case sensitive", "start", false},      // StartsWith is case-sensitive
		{"case sensitive match", "Start", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.ShouldExclude(tt.schema)
			if result != tt.expected {
				t.Errorf("ShouldExclude(%q) = %v, want %v", tt.schema, result, tt.expected)
			}
		})
	}
}
