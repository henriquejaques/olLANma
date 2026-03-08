package cli

import (
	"fmt"
	"strings"

	"github.com/henriquejaques/olLANma/internal/config"
)

// ParsedCommand represents the structured intent from the user
type ParsedCommand struct {
	IsInternal bool     // True if it's an olLANma specific command like 'config'
	Command    string   // E.g., 'run', 'list', 'pull', 'config'
	Args       []string // Arguments for that command
}

var supportedCommands = map[string]struct{}{
	"run":      {},
	"generate": {},
	"chat":     {},
	"list":     {},
	"tags":     {},
	"pull":     {},
	"rm":       {},
	"cp":       {},
	"ps":       {},
	"show":     {},
}

var reservedUnsupportedCommands = map[string]struct{}{
	"create": {},
	"serve":  {},
}

// Parse converts raw os.Args into a structured instruction for the rest of the app.
// It implements the "Fast Default" behavior.
func Parse(rawArgs []string, cfg config.Config) ParsedCommand {
	if len(rawArgs) < 2 {
		return ParsedCommand{} // Empty suggests usage text should be printed
	}

	arg1 := rawArgs[1]

	// 1. Check for Internal Commands
	if arg1 == "config" {
		return ParsedCommand{
			IsInternal: true,
			Command:    "config",
			Args:       rawArgs[2:],
		}
	}

	// 2. Check for supported passthrough commands
	if _, ok := supportedCommands[arg1]; ok {
		return ParsedCommand{
			IsInternal: false,
			Command:    arg1,
			Args:       rawArgs[2:],
		}
	}

	// 3. Keep unsupported-but-reserved commands out of Fast Default.
	// This ensures they fail explicitly instead of being reinterpreted as prompt/model text.
	if _, ok := reservedUnsupportedCommands[arg1]; ok {
		return ParsedCommand{
			IsInternal: false,
			Command:    arg1,
			Args:       rawArgs[2:],
		}
	}

	// 4. Fast Default Triggered Check: If arg1 is NOT a known command...
	// If there are exactly two arguments and arg1 is not a known command,
	// we will treat arg1 as the model, and the rest as the prompt.
	// E.g., `olLANma gemma2 "why is the sky blue?"` -> translates to -> `olLANma run gemma2 "why is the sky blue?"`
	// OR if there is only one un-matched arg, use the default model.
	// E.g., `olLANma "why is the sky blue?"` -> translates to -> `olLANma run <default_model> "why is the sky blue?"`

	model := cfg.DefaultModel
	var promptArgs []string

	if len(rawArgs) == 2 {
		// Just a prompt provided
		promptArgs = []string{rawArgs[1]}
	} else {
		// First arg is taken as model, rest as prompt
		model = rawArgs[1]
		promptArgs = rawArgs[2:]
	}

	fastArgs := []string{model}
	fastArgs = append(fastArgs, strings.Join(promptArgs, " "))

	if model == cfg.DefaultModel {
		fmt.Printf("[olLANma Notice: Fast Default Triggered for default model '%s']\n", model)
	} else {
		fmt.Printf("[olLANma Notice: Fast Default Triggered for model '%s']\n", model)
	}

	return ParsedCommand{
		IsInternal: false,
		Command:    "run",
		Args:       fastArgs,
	}
}
