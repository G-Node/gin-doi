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

// Default configuration struct containing non problematic test values
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
	// User working on the DOI server
	Serveruser string `yaml:"server_user"`
	// Local staging dir to create index and keyword pages
	Dirlocalstage string `yaml:"dir_local_stage"`
	// Full ssh access name of the server hosting the DOI server instance
	Doiserver string `yaml:"doi_server"`
	// DOI Server repo preparation directory
	Dirdoiprep string `yaml:"dir_doi_prep"`
	// DOI Server root doi hosting directory
	Dirdoi string `yaml:"dir_doi"`
}

// outFilename constructs a filename for the output markdown file
// from the checklist repo and registration information and
// returns it. Optionally an output path can be specified.
func outFilename(cl checklist, outpath string) string {
	owner := strings.ToLower(cl.Repoown)
	if len(cl.Repoown) > 5 {
		owner = owner[0:5]
	}
	reponame := strings.ToLower(cl.Repo)
	if len(cl.Repo) > 10 {
		reponame = reponame[0:15]
	}

	currdate := time.Now().Format("20060102")
	outfile := fmt.Sprintf("%s_%s-%s-%s.md", currdate, strings.ToLower(cl.Regid), owner, reponame)
	if outpath != "" {
		outfile = filepath.Join(outpath, outfile)
	}
	return outfile
}

// ChecklistTemplate is the data struct required to properly render
// the checklist file template.
type ChecklistTemplate struct {
	CL               checklist
	RepoLower        string
	RepoownLower     string
	SemiDOIScreenID  string
	FullDOIScreenID  string
	SemiDOICleanup   string
	SemiDOIDirpath   string
	FullDOIDirpath   string
	Forklog          string
	Logfiles         string
	Ziplog           string
	Zipfile          string
	KeywordsLocalDir string
	ToServer         string
	Citeyear         string
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
		CL:               cl,
		RepoLower:        strings.ToLower(cl.Repo),
		RepoownLower:     strings.ToLower(cl.Repoown),
		SemiDOIScreenID:  fmt.Sprintf("%s-%s", strings.ToLower(cl.Repoown), randAlnum(5)),
		FullDOIScreenID:  fmt.Sprintf("%s-%s", strings.ToLower(cl.Repoown), randAlnum(5)),
		SemiDOICleanup:   fmt.Sprintf("%s/10.12751/g-node.%s", cl.Dirdoiprep, cl.Regid),
		SemiDOIDirpath:   fmt.Sprintf("%s/10.12751/g-node.%s/%s", cl.Dirdoiprep, cl.Regid, strings.ToLower(cl.Repo)),
		FullDOIDirpath:   fmt.Sprintf("%s/%s", cl.Dirdoiprep, strings.ToLower(cl.Repo)),
		Forklog:          fmt.Sprintf("%s-%s.log", strings.ToLower(cl.Repoown), strings.ToLower(cl.Repo)),
		Logfiles:         fmt.Sprintf("%s-%s*.log", strings.ToLower(cl.Repoown), strings.ToLower(cl.Repo)),
		Ziplog:           fmt.Sprintf("%s-%s_zip.log", strings.ToLower(cl.Repoown), strings.ToLower(cl.Repo)),
		Zipfile:          fmt.Sprintf("%s/10.12751/g-node.%s/10.12751_g-node.%s.zip", cl.Dirdoi, cl.Regid, cl.Regid),
		KeywordsLocalDir: fmt.Sprintf("%s/keywords", cl.Dirlocalstage),
		ToServer:         fmt.Sprintf("%s@%s:/home/%s/staging", cl.Serveruser, cl.Doiserver, cl.Serveruser),
		Citeyear:         time.Now().Format("2006"),
	}

	if err := tmpl.Execute(fip, fullcl); err != nil {
		return fmt.Errorf("error writing checklist file: %s", err.Error())
	}

	return nil
}

// writeChecklistConfigYAML serializes the content of a checklist struct
// to a YAML config file.
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

// checklistFromMetadata checks all relevant entries in a received struct and
// returns a filled checklist struct. Will return an error, if any issues occur.
func checklistFromMetadata(md *libgin.RepositoryMetadata, doihost string) (checklist, error) {
	if md == nil || md.DataCite == nil || md.YAMLData == nil {
		return checklist{}, fmt.Errorf("encountered libgin.RepositoryMetadata nil pointer: %v", md)
	}
	if !strings.Contains(md.Identifier.ID, "10.12751/g-node.") {
		return checklist{}, fmt.Errorf("could not identify request ID")
	}
	if !strings.Contains(md.SourceRepository, "/") {
		return checklist{}, fmt.Errorf("could not parse source repository")
	}
	if len(md.Dates) < 1 {
		return checklist{}, fmt.Errorf("could not access publication dates")
	} else if md.Dates[0].Value == "" {
		return checklist{}, fmt.Errorf("publication date was empty")
	}
	if md.RelatedIdentifiers == nil {
		return checklist{}, fmt.Errorf("could not access requesting user")
	}
	if md.YAMLData == nil {
		return checklist{}, fmt.Errorf("YAMLData was unavailable")
	} else if md.YAMLData.Title == "" {
		return checklist{}, fmt.Errorf("title was unavailable")
	}
	if !strings.Contains(doihost, ":") {
		return checklist{}, fmt.Errorf("could not parse doihost")
	}
	regid := strings.Replace(md.Identifier.ID, "10.12751/g-node.", "", 1)
	repoinfo := strings.Split(md.SourceRepository, "/")
	repoown := repoinfo[0]
	repo := repoinfo[1]
	published := md.Dates[0].Value
	email := md.RequestingUser.Email
	fullname := md.RequestingUser.RealName
	if fullname == "" {
		fullname = md.RequestingUser.Username
	}
	title := md.YAMLData.Title
	hostinfo := strings.Split(doihost, ":")
	host := "__DOI_HOST__"
	if hostinfo[0] != "" {
		host = hostinfo[0]
	}
	prepdir := "__DOI_PREP_DIR__"
	hostdir := "__DOI_HOST_DIR__"
	if hostinfo[1] != "" {
		hostdir = hostinfo[1]
		// Unadvisable hack to get to the preparation path
		// outside the docker container. It depends on the
		// hosting and the preparation directory residing
		// side by side and being named 'doi' and 'doiprep'
		// respectively.
		prepdir = fmt.Sprintf("%sprep", hostdir)
	}
	cl := checklist{
		Regid:         regid,
		Repoown:       repoown,
		Repo:          repo,
		Regdate:       published,
		Email:         email,
		Userfullname:  fullname,
		Title:         title,
		Citation:      FormatAuthorList(md),
		Serveruser:    "__SERVER_USER__",
		Dirlocalstage: "__DIR_LOCAL_STAGE__",
		Doiserver:     host,
		Dirdoiprep:    prepdir,
		Dirdoi:        hostdir,
	}
	return cl, nil
}

// mkchecklistserver handles a checklist request via the DOI server. It creates
// a checklist config yaml file and a checklist markdown file in the preparation
// directory.
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

// parseRepoDatacite tries to access the request repository datacite file and
// parse the 'title' and the 'citation' from the files authors list.
// If the file cannot be accessed or there are any issues the script continues
// since both title and citation are not essential for the checklist.
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
	authors := make([]string, len(yamlInfo.Authors))
	for idx, author := range yamlInfo.Authors {
		firstnames := strings.Fields(author.FirstName)

		var initials string
		for _, name := range firstnames {
			initials += string(name[0])
		}
		authors[idx] = fmt.Sprintf("%s %s", strings.TrimSpace(author.LastName), strings.TrimSpace(initials))
	}
	authlist := strings.Join(authors, ", ")
	return title, authlist, nil
}

// mkchecklistcli handles command line input options and ensures
// default values for missing entries.
func mkchecklistcli(cmd *cobra.Command, args []string) {
	// default configuration
	defaultcl := checklist{
		Regid:         "__ID__",
		Repoown:       "__OWN__",
		Repo:          "__REPO__",
		Regdate:       "__DATE__",
		Email:         "__MAIL__",
		Userfullname:  "__USER_FULL__",
		Title:         "__TITLE__",
		Citation:      "__CITATION__",
		Serveruser:    "__SERVER_USER__",
		Dirlocalstage: "__DIR_LOCAL_STAGE__",
		Doiserver:     "__DOI.SERVER__",
		Dirdoiprep:    "__DIR_DOI_PREP__",
		Dirdoi:        "__DIR_DOI__",
	}

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
		fmt.Printf("Error parsing output directory flag: %s\n", err.Error())
	} else if oval != "" {
		outpath = oval
		fmt.Printf("-- Using output directory '%s'", outpath)
	}
	fmt.Println("-- Writing checklist file")
	err = mkchecklistFile(defaultcl, outpath)
	if err != nil {
		fmt.Printf("-- ERROR: %s", err.Error())
	}
	fmt.Println("-- Done")
}
