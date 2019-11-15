package main

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/G-Node/gin-cli/git"
	"github.com/G-Node/libgin/libgin"
	humanize "github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"
)

const (
	tmpdir      = "tmp"
	doixmlfname = "datacite.xml"
)

func Put(job DOIJob) error {
	conf := job.Config
	repopath := job.Source
	jobname := job.Name
	dReq := &job.Request

	prepDir(jobname, dReq.DOIInfo, conf)

	targetpath := filepath.Join(conf.Storage.TargetDirectory, jobname)
	preperrors := make([]string, 0, 5)
	zipsize, err := cloneandzip(repopath, jobname, targetpath, conf)
	dReq.DOIInfo.FileSize = humanize.IBytes(uint64(zipsize))
	if err != nil {
		// failed to clone and zip
		// save the error for reporting and continue with the XML prep
		preperrors = append(preperrors, err.Error())
		dReq.DOIInfo.FileSize = ""
	}
	createIndexFile(jobname, dReq, conf)

	fp, err := os.Create(filepath.Join(targetpath, "doi.xml"))
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": jobname,
		}).Error("Could not create the metadata template")
		// XML Creation failed; return with error
		dReq.ErrorMessages = append(preperrors, fmt.Sprintf("Failed to create the XML metadata template: %s", err))
		sendMaster(dReq, conf)
		return err
	}
	defer fp.Close()

	// No registering. But the XML is provided with everything
	doixml := filepath.Join(conf.TemplatePath, doixmlfname)
	data, err := GetXML(dReq.DOIInfo, doixml)
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": jobname,
		}).Error("Could not parse the metadata file")
		dReq.ErrorMessages = append(preperrors, fmt.Sprintf("Failed to parse the XML metadata: %s", err))
		sendMaster(dReq, conf)
		return err
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

	if len(preperrors) > 0 {
		// Resend email with errors if any occurred
		dReq.ErrorMessages = preperrors
		sendMaster(dReq, conf)
	}
	return err
}

// cloneandzip clones the source repository into a temporary directory under targetpath, zips the contents, and returns the size of the zip file in bytes.
func cloneandzip(repopath string, jobname string, targetpath string, conf *Configuration) (int64, error) {
	// Clone under targetpath (will create subdirectory with repository name)
	if err := os.MkdirAll(targetpath, 0777); err != nil {
		errmsg := fmt.Sprintf("Failed to create temporary clone directory: %s", tmpdir)
		log.Error(errmsg)
		return -1, fmt.Errorf(errmsg)
	}

	// Clone
	if err := CloneRepo(repopath, targetpath, conf); err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": jobname,
		}).Error("Repository cloning failed")
		return -1, fmt.Errorf("Failed to clone repository '%s': %v", repopath, err)
	}

	// Uninit the annex and delete .git directory
	repoparts := strings.SplitN(repopath, "/", 2)
	reponame := strings.ToLower(repoparts[1]) // clone directory is always lowercase
	repodir := filepath.Join(targetpath, reponame)
	if err := derepoCloneDir(repodir); err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": jobname,
		}).Error("Repository cleanup (uninit & derepo) failed")
		return -1, fmt.Errorf("Failed to uninit and cleanup repository '%s': %v", repopath, err)
	}

	// Zip
	log.Infof("Preparing zip file for %s", jobname)
	zipfilename := filepath.Join(targetpath, jobname+".zip")
	zipsize, err := zip(repodir, zipfilename)
	if err != nil {
		log.WithFields(log.Fields{
			"source": lpStorage,
			"error":  err,
			"target": jobname,
		}).Error("Could not zip the data")
		return -1, fmt.Errorf("Failed to create the zip file: %v", err)
	}
	log.Infof("Archive size: %d", zipsize)
	return zipsize, nil
}

func zip(source, zipfilename string) (int64, error) {
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

func createIndexFile(target string, info *DOIReq, conf *Configuration) error {
	tmpl, err := template.ParseFiles(filepath.Join(conf.TemplatePath, "landingpage.tmpl"))
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

	fp, err := os.Create(filepath.Join(conf.Storage.TargetDirectory, target, "index.html"))
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

func prepDir(jobname string, info *DOIRegInfo, conf *Configuration) error {
	storagedir := conf.Storage.TargetDirectory
	err := os.MkdirAll(filepath.Join(storagedir, jobname), os.ModePerm)
	if err != nil {
		log.WithFields(log.Fields{
			"source":  lpStorage,
			"error":   err,
			"jobname": jobname,
		}).Error("Could not create the target directory")
		return err
	}
	// Deny access per default
	file, err := os.Create(filepath.Join(storagedir, jobname, ".htaccess"))
	if err != nil {
		log.WithFields(log.Fields{
			"source":  lpStorage,
			"error":   err,
			"jobname": jobname,
		}).Error("Could not create .httaccess")
		return err
	}
	defer file.Close()
	// todo check
	_, err = file.Write([]byte("deny from all"))
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"jobname": jobname,
		}).Error("Could not write to .httaccess")
		return err
	}
	return nil
}

func derepoCloneDir(directory string) error {
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

// CloneRepo clones a git repository (with git-annex) specified by URI to the
// destination directory.
func CloneRepo(URI string, destdir string, conf *Configuration) error {
	// NOTE: CloneRepo changes the working directory to the cloned repository
	// See: https://github.com/G-Node/gin-cli/issues/225
	// This will need to change when that issue is fixed
	origdir, err := os.Getwd()
	if err != nil {
		log.Errorf("%s: Failed to get working directory when cloning repository. Was our working directory removed?", lpStorage)
		return err
	}
	defer os.Chdir(origdir)
	err = os.Chdir(destdir)
	if err != nil {
		return err
	}
	log.Debugf("Cloning %s", URI)

	clonechan := make(chan git.RepoFileStatus)
	go conf.GIN.Session.CloneRepo(strings.ToLower(URI), clonechan)
	for stat := range clonechan {
		log.Debug(stat)
		if stat.Err != nil {
			log.Errorf("Repository cloning failed: %s", stat.Err)
			return stat.Err
		}
	}

	downloadchan := make(chan git.RepoFileStatus)
	go conf.GIN.Session.GetContent(nil, downloadchan)
	for stat := range downloadchan {
		log.Debug(stat)
		if stat.Err != nil {
			log.Errorf("Repository cloning failed during annex get: %s", stat.Err)
			return stat.Err
		}
	}
	return nil
}
