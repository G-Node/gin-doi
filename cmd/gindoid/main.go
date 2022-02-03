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
	cmds := make([]*cobra.Command, 8)
	cmds[0] = &cobra.Command{
		Use:                   "start",
		Short:                 "Start the GIN DOI service",
		Args:                  cobra.NoArgs,
		Run:                   web,
		Version:               verstr,
		DisableFlagsInUseLine: true,
	}
	cmds[1] = &cobra.Command{
		Use:   "make-html <xml file>...",
		Short: "Generate the HTML landing page from one or more DataCite XML files",
		Long: `Generate the HTML landing page from one or more DataCite XML files.

The command accepts file paths and URLs (mixing allowed) and will generate one HTML page for each XML file found. If the page generation requires information that is missing from the XML file (e.g., archive file size, repository URLs), the program will attempt to retrieve the metadata by querying the online resources. If that fails, a warning is printed and the page is still generated with the available information.
Using the optional '-o' argument an alternative output path can be specified.`,
		Args:                  cobra.MinimumNArgs(1),
		Run:                   clihtml,
		Version:               verstr,
		DisableFlagsInUseLine: true,
	}
	cmds[1].Flags().StringP("out", "o", "", "[OPTIONAL] output file directory; must exist")
	cmds[2] = &cobra.Command{
		Use:   "make-keyword-pages <xml file>...",
		Short: "Generate keyword index pages",
		Long: `Generate keyword index pages.

The command accepts file paths and URLs (mixing allowed) and will generate one HTML page for each unique keyword found in the XML files. Each page lists (and links to) all datasets that use the keyword.

Previously generated pages are overwritten, so this command only makes sense if using all published XML files to generate complete listings.
Using the optional '-o' argument an alternative output path can be specified.`,
		Args:                  cobra.MinimumNArgs(1),
		Run:                   clikeywords,
		Version:               verstr,
		DisableFlagsInUseLine: true,
	}
	cmds[2].Flags().StringP("out", "o", "", "[OPTIONAL] output file directory; must exist")
	cmds[3] = &cobra.Command{
		Use:   "make-xml <yml file>...",
		Short: "Generate the doi.xml file from one or more DataCite YAML files",
		Long: `Generate the doi.xml file from one or more DataCite YAML files.

The command accepts GIN repositories of format "GIN:owner/repository", yaml file paths and URLs to yaml files (mixing allowed) and will generate one XML file for each YAML file found. If the page generation requires information that is missing from the XML file (e.g., archive file size, repository URLs), the program will attempt to retrieve the metadata by querying the online resources. If that fails, a warning is printed and the file is still generated with the available information. Contextual information like size or date have to be added manually.`,
		Args:                  cobra.MinimumNArgs(1),
		Run:                   mkxml,
		Version:               verstr,
		DisableFlagsInUseLine: true,
	}
	cmds[4] = &cobra.Command{
		Use:   "make-index <xml file>...",
		Short: "Generate the index.html file from one or more DataCite XML files",
		Long: `Generate the index.html file from one or more DataCite XML files.

The command accepts file paths and URLs (mixing allowed) and will generate one index HTML page containing the information of all XML files found.
Using the optional '-o' argument an alternative output path can be specified.`,
		Args:                  cobra.MinimumNArgs(1),
		Run:                   cliindex,
		Version:               verstr,
		DisableFlagsInUseLine: true,
	}
	cmds[4].Flags().StringP("out", "o", "", "[OPTIONAL] output file directory; must exist")
	cmds[5] = &cobra.Command{
		Use:   "make-sitemap <xml file>...",
		Short: "Generate the urls.txt google sitemap file from one or more DataCite XML files",
		Long: `Generate the urls.txt google sitemap file from one or more DataCite XML files.

The command accepts file paths and URLs (mixing allowed) and will generate one index HTML page containing the information of all XML files found.
Using the optional '-o' argument an alternative output path can be specified.`,
		Args:                  cobra.MinimumNArgs(1),
		Run:                   clisitemap,
		Version:               verstr,
		DisableFlagsInUseLine: true,
	}
	cmds[5].Flags().StringP("out", "o", "", "[OPTIONAL] output file directory; must exist")
	cmds[6] = &cobra.Command{
		Use:   "make-all <xml file>...",
		Short: "Generate all html files and the google sitemap file.",
		Long: `Generate all html files and the google sitemap file.

The command accepts file paths and URLs (mixing allowed) of DOI XML files 
and will generate the root landing HTML page, the google sitemap urls.txt file, 
the keywords html pages and all DOI html landing pages from the XML files.
Using the optional '-o' argument an alternative output path can be specified.`,
		Args:                  cobra.MinimumNArgs(1),
		Run:                   mkall,
		Version:               verstr,
		DisableFlagsInUseLine: true,
	}
	cmds[6].Flags().StringP("out", "o", "", "[OPTIONAL] output file directory; must exist")
	cmds[7] = &cobra.Command{
		Use:   "make-checklist",
		Short: "Generate a DOI registration checklist file.",
		Long: `Generate a DOI registration checklist file.

The command will create a markdown file containing a DOI dataset registration checklist.
By default all variables will contain placeholder text and the file will be placed at the
executing path.
Using the optional '-o' argument an alternative output path can be specified.
Using the optional '-c' argument a yaml config file can be specified to automatically
replace the default variable values. If a config file is specified, the service will 
additionally try to fetch dataset 'title' and 'authors' from the corresponding gin repository.`,
		Args:                  cobra.MinimumNArgs(0),
		Run:                   mkchecklistcli,
		Version:               verstr,
		DisableFlagsInUseLine: true,
	}
	cmds[7].Flags().StringP("config", "c", "", "[OPTIONAL] config yaml file")
	cmds[7].Flags().StringP("out", "o", "", "[OPTIONAL] output file directory; must exist")

	rootCmd.AddCommand(cmds...)
	return rootCmd
}

func main() {
	verstr := fmt.Sprintf("GIN DOI %s Build %s (%s)", appversion, build, commit)

	rootCmd := setUpCommands(verstr)
	rootCmd.SetVersionTemplate("{{.Version}}")

	// Engage
	err := rootCmd.Execute()
	if err != nil {
		fmt.Printf("Error running gin-doi: %q\n", err.Error())
	}
}
