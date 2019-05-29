package main

import (
	"fmt"
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/G-Node/gin-cli/git"
	"github.com/G-Node/libgin/libgin"
	log "github.com/sirupsen/logrus"
)

const (
	tmpdir      = "tmp"
	doixmlfname = "datacite.xml"
)

type LocalStorage struct {
	Path         string
	Source       DataSource
	HTTPBase     string
	MServer      *MailServer
	TemplatePath string
	SCPURL       string
	KnownHosts   string
}

func (ls *LocalStorage) Exists(target string) (bool, error) {
	return false, nil
}

func (ls LocalStorage) Put(job DOIJob) error {
	repopath := job.Source
	jobname := job.Name
	dReq := &job.Request

	ls.prepDir(jobname, dReq.DOIInfo)

	targetpath := filepath.Join(ls.Path, jobname)
	preperrors := make([]string, 0, 5)
	zipsize, err := ls.cloneandzip(repopath, jobname, targetpath)
	if err != nil {
		// failed to clone and zip
		// save the error for reporting and continue with the XML prep
		preperrors = append(preperrors, err.Error())
	}
	ls.createIndexFile(jobname, dReq)
	dReq.DOIInfo.FileSize = zipsize/(1024*1000) + 1 // Proper size conversion to closest human-readable size

	fp, err := os.Create(filepath.Join(targetpath, "doi.xml"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": jobname,
		}).Error("Could not create the metadata template")
		preperrors = append(preperrors, fmt.Sprintf("Failed to create the XML metadata template: %s", err))
	}
	defer fp.Close()

	// No registering. But the XML is provided with everything
	doixml := filepath.Join(ls.TemplatePath, doixmlfname)
	data, err := GetXML(dReq.DOIInfo, doixml)
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": jobname,
		}).Error("Could not parse the metadata file")
		preperrors = append(preperrors, fmt.Sprintf("Failed to parse the XML metadata: %s", err))
	}
	_, err = fp.Write([]byte(data))
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": jobname,
		}).Error("Could not write to the metadata file")
		preperrors = append(preperrors, fmt.Sprintf("Failed to write the metadata XML file: %s", err))
	}
	dReq.ErrorMessages = preperrors
	ls.sendMaster(dReq)
	return err
}

// cloneandzip clones the source repository into a temporary directory under targetpath, zips the contents, and returns the size of the zip file in bytes.
func (ls LocalStorage) cloneandzip(repopath string, jobname string, targetpath string) (int64, error) {
	// Clone under targetpath (will create subdirectory with repository name)
	if err := os.MkdirAll(targetpath, 0777); err != nil {
		errmsg := fmt.Sprintf("Failed to create temporary clone directory: %s", tmpdir)
		log.Error(errmsg)
		return -1, fmt.Errorf(errmsg)
	}

	// Clone
	ds := ls.GetDataSource()
	if err := ds.CloneRepo(repopath, targetpath); err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": jobname,
		}).Error("Repository cloning failed")
		return -1, fmt.Errorf("Failed to clone repository '%s': %v", repopath, err)
	}

	// Uninit the annex and delete .git directory
	repoparts := strings.SplitN(repopath, "/", 2)
	reponame := repoparts[1]
	repodir := filepath.Join(targetpath, reponame)
	if err := ls.derepoCloneDir(repodir); err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": jobname,
		}).Error("Repository cleanup (uninit & derepo) failed")
		return -1, fmt.Errorf("Failed to uninit and cleanup repository '%s': %v", repopath, err)
	}

	// Zip
	zipfilename := filepath.Join(targetpath, jobname+".zip")
	zipsize, err := ls.zip(repodir, zipfilename)
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": jobname,
		}).Error("Could not zip the data")
		return -1, fmt.Errorf("Failed to create the zip file: %v", err)
	}
	return zipsize, nil
}

func (ls *LocalStorage) zip(source, zipfilename string) (int64, error) {
	fn := fmt.Sprintf("zip(%s, %s)", source, zipfilename) // keep original args for errmsg
	source, err := filepath.Abs(source)
	if err != nil {
		log.Errorf("%s: Failed to get abs path for source directory in function '%s': %v", lpStorage, fn, err)
		return -1, err
	}

	zipfilename, err = filepath.Abs(zipfilename)
	if err != nil {
		log.Errorf("%s: Failed to get abs path for target zip file in function '%s': %v", lpStorage, fn, err)
		return -1, err
	}

	// Create zip file IO writer for MakeZip function
	zipfp, err := os.Create(zipfilename)
	if err != nil {
		log.Errorf("%s: Failed to create zip file for writing in function '%s': %v", lpStorage, fn, err)
		return -1, err
	}
	defer zipfp.Close()
	// Change into clone directory to make the paths in the zip archive repo
	// root-relative.
	origdir, err := os.Getwd()
	if err != nil {
		log.Errorf("%s: Failed to get working directory in function '%s': %v", lpStorage, fn, err)
		return -1, err
	}
	defer os.Chdir(origdir)
	if err := os.Chdir(source); err != nil {
		log.Errorf("%s: Failed to change to source directory to make zip file in function '%s': %v", lpStorage, fn, err)
		return -1, err
	}

	if err := libgin.MakeZip(zipfp, "."); err != nil {
		log.Errorf("%s: Failed to create zip file in function '%s': %v", lpStorage, fn, err)
		return -1, err
	}

	stat, _ := zipfp.Stat()
	return stat.Size(), nil
}

func (ls LocalStorage) GetDataSource() DataSource {
	return ls.Source
}

func (ls LocalStorage) createIndexFile(target string, info *DOIReq) error {
	tmpl, err := template.ParseFiles(filepath.Join(ls.TemplatePath, "doiInfo.tmpl"))
	if err != nil {
		if err != nil {
			log.WithFields(log.Fields{
				"source": lpStorage,
				"error":  err,
				"target": target,
			}).Error("Could not parse the DOI template")
			return err
		}
		return err
	}

	fp, err := os.Create(filepath.Join(ls.Path, target, "index.html"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": target,
		}).Error("Could not create the DOI index.html")
		return err
	}
	defer fp.Close()
	if err := tmpl.Execute(fp, info); err != nil {
		log.WithFields(log.Fields{
			"source":  lpStorage,
			"error":   err,
			"doiInfo": info,
		}).Error("Could not execute the DOI template")
		return err
	}
	return nil
}

func (ls *LocalStorage) prepDir(target string, info *DOIRegInfo) error {
	err := os.MkdirAll(filepath.Join(ls.Path, target), os.ModePerm)
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": target,
		}).Error("Could not create the target directory")
		return err
	}
	// Deny access per default
	file, err := os.Create(filepath.Join(ls.Path, target, ".htaccess"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": target,
		}).Error("Could not create .httaccess")
		return err
	}
	defer file.Close()
	// todo check
	_, err = file.Write([]byte("deny from all"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": target,
		}).Error("Could not write to .httaccess")
		return err
	}
	return nil
}
func (ls LocalStorage) getSCP(dReq *DOIReq) string {
	return fmt.Sprintf("%s/%s/doi.xml", ls.SCPURL, dReq.DOIInfo.UUID)
}
func (ls LocalStorage) sendMaster(dReq *DOIReq) error {
	urljoin := func(a, b string) string {
		fallback := fmt.Sprintf("%s/%s (fallback URL join)", a, b)
		base, err := url.Parse(a)
		if err != nil {
			return fallback
		}
		suffix, err := url.Parse(b)
		if err != nil {
			return fallback
		}
		return base.ResolveReference(suffix).String()
	}

	repopath := dReq.URI
	userlogin := dReq.User.MainOId.Login
	useremail := dReq.User.MainOId.Account.Email.Email
	xmlurl := ls.getSCP(dReq)
	uuid := dReq.DOIInfo.UUID
	doitarget := urljoin(ls.HTTPBase, uuid)
	repourl := fmt.Sprintf("%s/%s", ls.Source.GinURL(), repopath)

	errorlist := ""
	if len(dReq.ErrorMessages) > 0 {
		errorlist = "The following errors occurred during the dataset preparation\n"
		for idx, msg := range dReq.ErrorMessages {
			errorlist = fmt.Sprintf("%s	%d. %s\n", errorlist, idx+1, msg)
		}
	}

	subject := fmt.Sprintf("New DOI registration request: %s", repopath)

	body := `A new DOI registration request has been received.

	Repository: %s [%s]
	User: %s
	Email address: %s
	DOI XML: %s
	DOI target URL: %s
	UUID: %s

%s
`
	body = fmt.Sprintf(body, repopath, repourl, userlogin, useremail, xmlurl, doitarget, uuid, errorlist)
	return ls.MServer.SendMail(subject, body)
}
func (ls *LocalStorage) derepoCloneDir(directory string) error {
	directory, err := filepath.Abs(directory)
	if err != nil {
		log.Errorf("%s: Failed to get abs path for repo directory while cleaning up '%s'. Was our working directory removed?", lpStorage, directory)
		return err
	}
	// NOTE: Most of the functionality in this method will be moved to libgin
	// since GOGS has similar functions
	// Change into directory to cleanup and defer changing back
	origdir, err := os.Getwd()
	if err != nil {
		log.Errorf("%s: Failed to get abs path for working directory while cleaning up directory '%s'. Was our working directory removed?", lpStorage, directory)
		return err
	}
	defer os.Chdir(origdir)
	if err := os.Chdir(directory); err != nil {
		log.Errorf("%s: Failed to change working directory to '%s': %v", lpStorage, directory, err)
		return err
	}

	// Uninit annex
	cmd := git.AnnexCommand("uninit")
	// git annex uninit always returns with an error (-_-) so we ignore the
	// error and check if annex info complains instead
	cmd.Run()

	_, err = git.AnnexInfo()
	if err != nil {
		log.Errorf("%s: Failed to uninit annex in cloned repository '%s': %v", lpStorage, directory, err)
	}

	gitdir, err := filepath.Abs(filepath.Join(directory, ".git"))
	if err != nil {
		log.Errorf("%s: Failed to get abs path for git directory while cleaning up directory '%s'. Was our working directory removed?", lpStorage, directory)
		return err
	}
	// Set write permissions on everything under gitdir
	var mode os.FileMode
	walker := func(path string, info os.FileInfo, err error) error {
		// walker sets the permission for any file found to 0660 and directories to
		// 770, to allow deletion
		if info == nil {
			return nil
		}

		mode = 0660
		if info.IsDir() {
			mode = 0770
		}

		if err := os.Chmod(path, mode); err != nil {
			log.Errorf("failed to change permissions on '%s': %v", path, err)
		}
		return nil
	}
	if err := filepath.Walk(gitdir, walker); err != nil {
		log.Errorf("%s: Failed to set write permissions for directories and files under gitdir '%s': %v", lpStorage, gitdir, err)
		return err
	}

	// Delete .git directory
	if err := os.RemoveAll(gitdir); err != nil {
		log.Errorf("%s: Failed to remove git directory '%s': %v", lpStorage, gitdir, err)
		return err
	}

	return nil
}
