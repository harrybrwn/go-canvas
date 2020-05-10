package cmd

import "github.com/spf13/cobra"

// Execute will execute the root comand on the cli
func Execute() {
	root.Execute()
}

var root = &cobra.Command{
	Use: "canvas",
}
