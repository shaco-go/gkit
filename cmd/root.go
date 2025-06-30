package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gkit",
	Short: "GKit is an efficient Go language development assistant toolkit",
	Long:  "GKit is an efficient Go language development assistant toolkit, providing the following features:\n  - Fast cloning of Golang project templates\n  - Smart package management, no need to remember complete package names to install dependencies\n  - Practical tool collection to improve Go development efficiency",
}


func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
