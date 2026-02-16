//go:build tools
// +build tools

// This file ensures dependencies are tracked in go.mod even if not yet used in code.
// These dependencies will be used in upcoming tasks.
package tools

import (
	_ "github.com/fatih/color"
	_ "github.com/olekukonko/tablewriter"
	_ "github.com/spf13/cobra"
)
