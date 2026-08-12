package main

import (
	"fmt"
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage and run agents",
}

var agentRunCmd = &cobra.Command{
	Use:   "run [prompt]",
	Short: "Run an agent headlessly",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		prompt := args[0]
		fmt.Println("Headless agent run not yet fully implemented. Prompt:", prompt)
	},
}

func init() {
	agentCmd.AddCommand(agentRunCmd)
}
