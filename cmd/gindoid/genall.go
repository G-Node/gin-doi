package main

import (
	"github.com/spf13/cobra"
)

// mkall calls all functions creating doi specific files
// from provided XML files or URLs.
func mkall(cmd *cobra.Command, args []string) {
	// generate root landing page file
	cliindex(cmd, args)
	// generate sitemap file
	clisitemap(cmd, args)
	// generate keyword pages
	clikeywords(cmd, args)
	// generate html dataset landing pages
	clihtml(cmd, args)
}
