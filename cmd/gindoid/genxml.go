package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// mkxml reads one or more datacite YAML files from GIN, a provided URL or from 
// a direct file and generates a Datacite XML file for each.
// Reading files from GIN requires only the repository owner and the repository name
// of the GIN repository prefixed with GIN in the format "GIN:[owner]/[repository]"
func mkxml(cmd *cobra.Command, args []string) {
	fmt.Printf("Generating %d xml files\n", len(args))
}
