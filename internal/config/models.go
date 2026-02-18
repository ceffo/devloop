package config

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// FetchModels runs the given binary with --help and parses the available model
// names from the --model flag's choices list.
//
// Returns a non-nil map when models are successfully discovered, or nil (with no
// error) when the binary is not found in PATH or does not advertise choices.
// An error is returned only on unexpected execution failures.
func FetchModels(binary string) (map[string]bool, error) {
	if _, err := exec.LookPath(binary); err != nil {
		// Binary not installed — skip model validation silently.
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "--help")
	out, err := cmd.CombinedOutput()
	// Many CLIs exit non-zero for --help; treat that as OK as long as we got output.
	if err != nil && len(out) == 0 {
		return nil, nil
	}

	models := parseModelsFromHelp(string(out))
	return models, nil
}

// parseModelsFromHelp extracts the set of valid model names from the help text
// of an AI CLI tool.  It looks for a block of the form:
//
//	--model <model>   ... (choices: "m1", "m2", ...)
//
// The choices block may span multiple lines but must belong to the --model flag
// itself (i.e. it must appear before the next flag definition).  Returns nil
// when no such block is found (i.e. the tool does not enumerate its supported
// models).
func parseModelsFromHelp(help string) map[string]bool {
	// Locate the --model flag.
	modelIdx := strings.Index(help, "--model")
	if modelIdx < 0 {
		return nil
	}

	// Find where the *next* flag starts so we don't bleed into another flag's
	// choices block (e.g. --output-format or --permission-mode in claude --help).
	afterModel := help[modelIdx+len("--model"):]
	nextFlagIdx := strings.Index(afterModel, "\n  --")
	var modelSection string
	if nextFlagIdx >= 0 {
		modelSection = afterModel[:nextFlagIdx]
	} else {
		modelSection = afterModel
	}

	// Look for a choices list only within this flag's own description.
	choicesIdx := strings.Index(modelSection, "(choices:")
	if choicesIdx < 0 {
		return nil
	}

	// Grab everything from "(choices:" up to (and including) the closing ")".
	closingIdx := strings.Index(modelSection[choicesIdx:], ")")
	if closingIdx < 0 {
		return nil
	}
	choicesBlock := modelSection[choicesIdx : choicesIdx+closingIdx+1]

	// Extract every double-quoted string from the choices block.
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindAllStringSubmatch(choicesBlock, -1)
	if len(matches) == 0 {
		return nil
	}

	models := make(map[string]bool, len(matches))
	for _, m := range matches {
		models[m[1]] = true
	}
	return models
}
