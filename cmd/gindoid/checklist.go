package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/G-Node/libgin/libgin"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

// checklist allows loading basic information from a config YAML file
type checklist struct {
	// Entries required for every DOI request
	// Paste basic information from the corresponding issue on
	//   https://gin.g-node.org/G-Node/DOIMetadata
	// Automated registration [id] from "10.12751/g-node.[id]"
	Regid string `yaml:"reg_id"`
	// Repository owner
	Repoown string `yaml:"repo_own"`
	// Repository name
	Repo string `yaml:"repo"`
	// Date issued from doi.xml; Format YYYY-MM-DD
	Regdate string `yaml:"reg_date"`
	// DOI requestee email address
	Email string `yaml:"email"`
	// DOI requestee full name
	Userfullname string `yaml:"user_full_name"`
	// Entries that are usually handled automatically via repo datacite entry
	// DOI request title; usually handled automatically via repo datacite entry
	Title string `yaml:"title"`
	// Author citation list; usually handled automatically via repo datacite entry
	Citation string `yaml:"citation"`
	// Entries that are set once and remain unchanged for future DOI requests
	// Full ssh access name of the server hosting the DOI server instance
	Doiserver string `yaml:"doi_server"`
	// DOI Server repo preparation directory
	Dirdoiprep string `yaml:"dir_doi_prep"`
	// DOI Server root doi hosting directory
	Dirdoi string `yaml:"dir_doi"`
}

// ChecklistTemplate is the full data struct required to properly render
// the checklist file template. It contains processed information that is
// not available in the basic checklist struct.
type ChecklistTemplate struct {
	CL              checklist
	RepoLower       string
	RepoownLower    string
	SemiDOIScreenID string
	FullDOIScreenID string
	SemiDOICleanup  string
	SemiDOIDirpath  string
	FullDOIDirpath  string
	Forklog         string
	Logfiles        string
	Ziplog          string
	Zipfile         string
	Citeyear        string
}

// outFilename constructs a filename for the output markdown file
// from the checklist repository and registration information and
// returns it. An optional output path can be specified.
func outFilename(cl checklist, outpath string) string {
	owner := strings.ToLower(cl.Repoown)
	if len(cl.Repoown) > 5 {
		owner = owner[0:5]
	}
	reponame := strings.ToLower(cl.Repo)
	if len(cl.Repo) > 15 {
		reponame = reponame[0:15]
	}

	currdate := time.Now().Format("20060102")
	outfile := fmt.Sprintf("%s_%s-%s-%s.md", currdate, strings.ToLower(cl.Regid), owner, reponame)
	if outpath != "" {
		outfile = filepath.Join(outpath, outfile)
	}
	return outfile
}

// mkchecklistFile creates an output markdown file with the contents
// of the passed checklist struct. A path for the output file can
// be provided.
func mkchecklistFile(cl checklist, outpath string) error {
	outfile := outFilename(cl, outpath)
	fip, err := os.Create(outfile)
	if err != nil {
		return fmt.Errorf("could not create checklist file: %s", err.Error())
	}
	defer fip.Close()

	tmpl, err := prepareTemplates("Checklist")
	if err != nil {
		return fmt.Errorf("error preparing checklist template: %s", err.Error())
	}

	fullcl := ChecklistTemplate{
		CL:              cl,
		RepoLower:       strings.ToLower(cl.Repo),
		RepoownLower:    strings.ToLower(cl.Repoown),
		SemiDOIScreenID: fmt.Sprintf("%s-%s", strings.ToLower(cl.Repoown), randAlnum(5)),
		FullDOIScreenID: fmt.Sprintf("%s-%s", strings.ToLower(cl.Repoown), randAlnum(5)),
		SemiDOICleanup:  fmt.Sprintf("%s/10.12751/g-node.%s", cl.Dirdoiprep, cl.Regid),
		SemiDOIDirpath:  fmt.Sprintf("%s/10.12751/g-node.%s/%s", cl.Dirdoiprep, cl.Regid, strings.ToLower(cl.Repo)),
		FullDOIDirpath:  fmt.Sprintf("%s/%s", cl.Dirdoiprep, strings.ToLower(cl.Repo)),
		Forklog:         fmt.Sprintf("%s-%s.log", strings.ToLower(cl.Repoown), strings.ToLower(cl.Repo)),
		Logfiles:        fmt.Sprintf("%s-%s*.log", strings.ToLower(cl.Repoown), strings.ToLower(cl.Repo)),
		Ziplog:          fmt.Sprintf("%s-%s_zip.log", strings.ToLower(cl.Repoown), strings.ToLower(cl.Repo)),
		Zipfile:         fmt.Sprintf("%s/10.12751/g-node.%s/10.12751_g-node.%s.zip", cl.Dirdoi, cl.Regid, cl.Regid),
		Citeyear:        time.Now().Format("2006"),
	}

	if err := tmpl.Execute(fip, fullcl); err != nil {
		return fmt.Errorf("error writing checklist file: %s", err.Error())
	}

	return nil
}

// checklistFromMetadata checks all relevant entries in a received struct and
// returns a filled checklist struct. Will return an error, if any issues occur.
func checklistFromMetadata(md *libgin.RepositoryMetadata, doihost string) (checklist, error) {
	if md == nil || md.DataCite == nil || md.YAMLData == nil {
		return checklist{}, fmt.Errorf("encountered libgin.RepositoryMetadata nil pointer: %v", md)
	}
	if md.RequestingUser == nil {
		return checklist{}, fmt.Errorf("encountered libgin.RequestingUser nil pointer: %v", md)
	}
	repoinfo := strings.Split(md.SourceRepository, "/")
	if len(repoinfo) != 2 {
		return checklist{}, fmt.Errorf("cannot parse SourceRepository: %v", md.SourceRepository)
	}
	if len(md.Dates) < 1 {
		return checklist{}, fmt.Errorf("missing pubication date: %v", md.Dates)
	}
	repoown := repoinfo[0]
	repo := repoinfo[1]
	regid := strings.Replace(md.Identifier.ID, "10.12751/g-node.", "", 1)
	published := md.Dates[0].Value
	email := md.RequestingUser.Email
	fullname := md.RequestingUser.RealName
	if fullname == "" {
		fullname = md.RequestingUser.Username
	}
	title := md.YAMLData.Title
	hostinfo := strings.Split(doihost, ":")
	host := "__DOI_HOST__"
	if len(hostinfo) > 0 && hostinfo[0] != "" {
		host = hostinfo[0]
	}
	prepdir := "__DOI_PREP_DIR__"
	hostdir := "__DOI_HOST_DIR__"
	if len(hostinfo) > 1 && hostinfo[1] != "" {
		hostdir = hostinfo[1]
		// Unadvisable hack to get to the preparation path
		// outside the docker container. It depends on the
		// hosting and the preparation directory residing
		// side by side and being named 'doi' and 'doiprep'
		// respectively.
		prepdir = fmt.Sprintf("%sprep", hostdir)
	}
	cl := checklist{
		Regid:        regid,
		Repoown:      repoown,
		Repo:         repo,
		Regdate:      published,
		Email:        email,
		Userfullname: fullname,
		Title:        title,
		Citation:     FormatAuthorList(md),
		Doiserver:    host,
		Dirdoiprep:   prepdir,
		Dirdoi:       hostdir,
	}
	return cl, nil
}

// mkchecklistserver handles a checklist request via the DOI server. It creates
// a checklist config yaml file and a checklist markdown file at the provided
// directory path.
func mkchecklistserver(md *libgin.RepositoryMetadata, preppath string, doihost string) error {
	cl, err := checklistFromMetadata(md, doihost)
	if err != nil {
		return fmt.Errorf("error parsing checklist information: %s", err.Error())
	}

	err = writeChecklistConfigYAML(cl, preppath)
	if err != nil {
		return err
	}

	err = mkchecklistFile(cl, preppath)
	if err != nil {
		return err
	}

	return nil
}

// readChecklistConfigYAML parses config information from a provided yaml file and
// returns a checklist struct containing the config information.
func readChecklistConfigYAML(yamlInfo *checklist, confile string) (*checklist, error) {
	infoyml, err := readFileAtPath(confile)
	if err != nil {
		return nil, fmt.Errorf("-- Error reading config file: %s", err.Error())
	}
	err = yaml.Unmarshal(infoyml, yamlInfo)
	if err != nil {
		return nil, fmt.Errorf("-- Error unmarshalling config file: %s", err.Error())
	}
	return yamlInfo, nil
}

// writeChecklistConfigYAML serializes the content of a checklist struct
// to a YAML checklist config file.
func writeChecklistConfigYAML(cl checklist, outpath string) error {
	data, err := yaml.Marshal(&cl)
	if err != nil {
		return fmt.Errorf("error marshalling checklist yaml: %s", err.Error())
	}
	fn := filepath.Join(outpath, fmt.Sprintf("conf_%s.yml", cl.Regid))
	err = ioutil.WriteFile(fn, data, 0664)
	if err != nil {
		return fmt.Errorf("error writing checklist yaml: %s", err.Error())
	}
	return nil
}

// formatYAMLAuthors parses Author lastnames and firstname initials
// from a libgin.RepositoryYAML.Authors list and returns them
// separated by comma as a single string.
func formatYAMLAuthors(yamlInfo *libgin.RepositoryYAML) string {
	authors := make([]string, len(yamlInfo.Authors))
	for idx, author := range yamlInfo.Authors {
		if author.FirstName == "" {
			authors[idx] = strings.TrimSpace(author.LastName)
			continue
		}
		firstnames := strings.Fields(author.FirstName)

		var initials string
		for _, name := range firstnames {
			initials += string(name[0])
		}
		authors[idx] = fmt.Sprintf("%s %s", strings.TrimSpace(author.LastName), strings.TrimSpace(initials))
	}
	return strings.Join(authors, ", ")
}

// parseRepoDatacite accesses a datacite YAML file from a provided
// URL and parses and returns the 'title' and formatted author list names.
func parseRepoDatacite(dcURL string) (string, string, error) {
	fmt.Printf("-- Loading datacite file at '%s'\n", dcURL)

	contents, err := readFileAtURL(dcURL)
	if err != nil {
		return "", "", err
	}

	yamlInfo := new(libgin.RepositoryYAML)
	err = yaml.Unmarshal(contents, yamlInfo)
	if err != nil {
		return "", "", fmt.Errorf("-- Error unmarshalling config file: %s", err.Error())
	}

	title := yamlInfo.Title
	authlist := formatYAMLAuthors(yamlInfo)

	return title, authlist, nil
}

// defaultChecklist returns a checklist with default string values.
func defaultChecklist() checklist {
	return checklist{
		Regid:        "__ID__",
		Repoown:      "__OWN__",
		Repo:         "__REPO__",
		Regdate:      "__DATE__",
		Email:        "__MAIL__",
		Userfullname: "__USER_FULL__",
		Title:        "__TITLE__",
		Citation:     "__CITATION__",
		Doiserver:    "__DOI_SERVER__",
		Dirdoiprep:   "__DIR_DOI_PREP__",
		Dirdoi:       "__DIR_DOI__",
	}
}

// mkchecklistcli handles command line input options and ensures
// default values for missing entries.
func mkchecklistcli(cmd *cobra.Command, args []string) {
	// default configuration
	defaultcl := defaultChecklist()

	// handling CLI config yaml; missing fields will keep the default values
	confile, err := cmd.Flags().GetString("config")
	if err != nil {
		fmt.Printf("-- Error parsing config flag: %s\n-- Exiting\n", err.Error())
		return
	}
	if confile != "" {
		loadedconf, err := readChecklistConfigYAML(&defaultcl, confile)
		if err != nil {
			fmt.Printf("%s\n-- Exiting\n", err.Error())
			return
		}
		defaultcl = *loadedconf

		// try to load title and citation from the gin datacite.yml
		baseURL := "https://gin.g-node.org"
		dcURL := fmt.Sprintf("%s/%s/%s/raw/master/datacite.yml", baseURL, defaultcl.Repoown, defaultcl.Repo)
		title, authors, err := parseRepoDatacite(dcURL)
		if err != nil {
			fmt.Printf("-- Error fetching repo datacite.yml: %s\n", err.Error())
		} else {
			if title != "" {
				defaultcl.Title = title
			}
			if authors != "" {
				defaultcl.Citation = authors
			}
		}
	}

	// handling CLI output file path; default is current directory
	var outpath string
	oval, err := cmd.Flags().GetString("out")
	if err != nil {
		fmt.Printf("-- Error parsing output directory flag: %s\n", err.Error())
	} else if oval != "" {
		outpath = oval
		fmt.Printf("-- Using output directory '%s'\n", outpath)
	}

	fmt.Println("-- Writing checklist file")
	err = mkchecklistFile(defaultcl, outpath)
	if err != nil {
		fmt.Printf("-- ERROR: %s\n", err.Error())
	}
	fmt.Println("-- Done")
}
