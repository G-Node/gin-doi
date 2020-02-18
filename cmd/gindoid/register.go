package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func register(cmd *cobra.Command, args []string) {
	reponame := args[0]

	fmt.Printf("Registering %q\n", reponame)
}
