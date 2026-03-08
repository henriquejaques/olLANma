package cli

import (
	"reflect"
	"testing"

	"github.com/henriquejaques/olLANma/internal/config"
)

func TestParse(t *testing.T) {
	mockCfg := config.Config{
		Host:         "127.0.0.1",
		Port:         "11434",
		DefaultModel: "llama3",
	}

	tests := []struct {
		name     string
		rawArgs  []string
		expected ParsedCommand
	}{
		{
			name:    "Too few arguments (empty string)",
			rawArgs: []string{"olLANma"},
			expected: ParsedCommand{
				IsInternal: false,
				Command:    "",
				Args:       nil,
			},
		},
		{
			name:    "Internal config command",
			rawArgs: []string{"olLANma", "config"},
			expected: ParsedCommand{
				IsInternal: true,
				Command:    "config",
				Args:       []string{},
			},
		},
		{
			name:    "Internal config command with extra args (should ignore/pass along)",
			rawArgs: []string{"olLANma", "config", "extra"},
			expected: ParsedCommand{
				IsInternal: true,
				Command:    "config",
				Args:       []string{"extra"},
			},
		},
		{
			name:    "Standard passthrough Command (list)",
			rawArgs: []string{"olLANma", "list"},
			expected: ParsedCommand{
				IsInternal: false,
				Command:    "list",
				Args:       []string{},
			},
		},
		{
			name:    "Standard passthrough Command with args (pull)",
			rawArgs: []string{"olLANma", "pull", "mistral"},
			expected: ParsedCommand{
				IsInternal: false,
				Command:    "pull",
				Args:       []string{"mistral"},
			},
		},
		{
			name:    "Standard passthrough Command (generate)",
			rawArgs: []string{"olLANma", "generate", "llama3", "hello"},
			expected: ParsedCommand{
				IsInternal: false,
				Command:    "generate",
				Args:       []string{"llama3", "hello"},
			},
		},
		{
			name:    "Standard passthrough Command (chat)",
			rawArgs: []string{"olLANma", "chat", "llama3", "hello"},
			expected: ParsedCommand{
				IsInternal: false,
				Command:    "chat",
				Args:       []string{"llama3", "hello"},
			},
		},
		{
			name:    "Standard passthrough Command (tags)",
			rawArgs: []string{"olLANma", "tags"},
			expected: ParsedCommand{
				IsInternal: false,
				Command:    "tags",
				Args:       []string{},
			},
		},
		{
			name:    "Reserved unsupported command stays explicit",
			rawArgs: []string{"olLANma", "create", "mymodel"},
			expected: ParsedCommand{
				IsInternal: false,
				Command:    "create",
				Args:       []string{"mymodel"},
			},
		},
		{
			name:    "Fast Default 1: Just a prompt (uses default model)",
			rawArgs: []string{"olLANma", "why is the sky blue?"},
			expected: ParsedCommand{
				IsInternal: false,
				Command:    "run",
				Args:       []string{"llama3", "why is the sky blue?"},
			},
		},
		{
			name:    "Fast Default 2: Unknown model name + prompt",
			rawArgs: []string{"olLANma", "gemma2", "tell me a joke"},
			expected: ParsedCommand{
				IsInternal: false,
				Command:    "run",
				Args:       []string{"gemma2", "tell me a joke"},
			},
		},
		{
			name:    "Fast Default 3: Multi-word prompt combined",
			rawArgs: []string{"olLANma", "gemma2", "tell", "me", "a", "joke"},
			expected: ParsedCommand{
				IsInternal: false,
				Command:    "run",
				Args:       []string{"gemma2", "tell me a joke"}, // join spaces behavior
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := Parse(tc.rawArgs, mockCfg)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Parse() failed for '%s'.\nGot:  %+v\nWant: %+v", tc.name, result, tc.expected)
			}
		})
	}
}
