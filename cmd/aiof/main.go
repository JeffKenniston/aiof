package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "aiof",
	Short: "AIOF Orchestrator CLI",
	Long:  `AIOF (Antigravity Orchestration Framework) CLI for managing agents and workspaces.`,
}

func main() {
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(agentCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
