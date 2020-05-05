package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	appversion string
	build      string
	commit     string
)

func init() {
	if appversion == "" {
		appversion = "[dev]"
	}
}

func setUpCommands(verstr string) *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:                   "gindoid",
		Long:                  "GIN DOI",
		Version:               fmt.Sprintln(verstr),
		DisableFlagsInUseLine: true,
	}
	cmds := make([]*cobra.Command, 3)
	cmds[0] = &cobra.Command{
		Use:                   "start",
		Short:                 "Start the GIN DOI service",
		Args:                  cobra.NoArgs,
		Run:                   web,
		Version:               verstr,
		DisableFlagsInUseLine: true,
	}
	cmds[1] = &cobra.Command{
		Use:                   "register <repopath>",
		Short:                 "Register a repository",
		Args:                  cobra.ExactArgs(1),
		Run:                   register,
		Version:               verstr,
		DisableFlagsInUseLine: true,
	}
	cmds[2] = &cobra.Command{
		Use:   "make-html <xml file>...",
		Short: "Generate the HTML landing page from one or more DataCite XML files",
		Long: `Generate the HTML landing page from one or more DataCite XML files.

The command accepts file paths and URLs (mixing allowed) and will generate one HTML page for each XML file found. If the page generation requires information that is missing from the XML file (e.g., archive file size, repository URLs), the program will attempt to retrieve the metadata by querying the online resources. If that fails, a warning is printed and the page is still generated with the available information.`,
		Args:                  cobra.MinimumNArgs(1),
		Run:                   mkhtml,
		Version:               verstr,
		DisableFlagsInUseLine: true,
	}

	rootCmd.AddCommand(cmds...)
	return rootCmd
}

func main() {
	verstr := fmt.Sprintf("GIN DOI %s Build %s (%s)", appversion, build, commit)

	rootCmd := setUpCommands(verstr)
	rootCmd.SetVersionTemplate("{{.Version}}")

	// Engage
	rootCmd.Execute()
}
