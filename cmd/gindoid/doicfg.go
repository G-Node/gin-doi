package main

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/G-Node/gin-cli/ginclient"
	"github.com/G-Node/gin-cli/ginclient/config"
	"github.com/G-Node/gin-cli/git"
	"github.com/G-Node/libgin/libgin"
	log "github.com/sirupsen/logrus"
)

// Configuration is used to store and pass the configuration settings
// throughout the service.
type Configuration struct {
	// Port for the GIN DOI service to listen on
	Port uint16
	// The encryption key, shared with GIN Web for verification
	Key string
	// HTML templates location for the GIN DOI service
	TemplatePath string
	// Processing queue length and max concurrent workers
	MaxQueue   int
	MaxWorkers int
	// GIN server configuration (web and git URLs) and DOI username and
	// password for cloning
	GIN struct {
		Username string
		Password string
		Session  *ginclient.Client
	}
	// DOI prefix
	DOIBase string
	// Email related settings (for sending notifications)
	Email struct {
		Server string
		// From address
		From string
		// File path with email addresses to which notifications are sent
		RecipientsFile string
	}
	// Settings related to the storage location for published data and landing
	// pages
	Storage struct {
		// Root storage location
		TargetDirectory string
		// URL where the published data is served from (used for email
		// notifications, linking, and redirection)
		StoreURL string
		// Used in email notification for convenient XML file retrieval (SCP
		// format host:/path/)
		XMLURL string
	}
}

// loadconfig reads all the configuration variables (from the environment)
func loadconfig() (*Configuration, error) {
	cfg := Configuration{}

	// NOTE: Temporary workaround. GIN Client internals need a bit of a
	// redesign to support in-memory configurations.
	confdir := libgin.ReadConf("configdir")
	confdir, err := filepath.Abs(confdir)
	if err != nil {
		return nil, err
	}
	os.Setenv("GIN_CONFIG_DIR", confdir)

	cfg.DOIBase = libgin.ReadConf("doibase")

	cfg.Email.Server = libgin.ReadConf("mailserver")
	cfg.Email.From = libgin.ReadConf("mailfrom")
	cfg.Email.RecipientsFile = libgin.ReadConf("mailtofile")

	cfg.Storage.TargetDirectory = libgin.ReadConf("target")
	cfg.Storage.StoreURL = libgin.ReadConf("storeurl")
	cfg.Storage.XMLURL = libgin.ReadConf("xmlurl")

	cfg.TemplatePath = libgin.ReadConf("templates")

	cfg.Key = libgin.ReadConf("key")
	maxqueue, err := strconv.Atoi(libgin.ReadConfDefault("maxqueue", "100"))
	if err != nil {
		log.Printf("Error while parsing maxqueue flag: %s", err.Error())
		log.Print("Using default")
		maxqueue = 100
	}
	cfg.MaxQueue = maxqueue

	maxworkers, err := strconv.Atoi(libgin.ReadConfDefault("maxworkers", "3"))
	if err != nil {
		log.Printf("Error while parsing maxworkers flag: %s", err.Error())
		log.Print("Using default")
		maxworkers = 3
	}
	cfg.MaxWorkers = maxworkers

	portstr := libgin.ReadConfDefault("port", "10443")
	port, err := strconv.ParseUint(portstr, 10, 16)
	if err != nil {
		return nil, err
	}

	cfg.Port = uint16(port)

	// Set up GIN client configuration (for cloning)

	ginurl := libgin.ReadConf("ginurl")
	giturl := libgin.ReadConf("giturl")
	log.Debugf("gin: %s -- git: %s", ginurl, giturl)

	webcfg, err := config.ParseWebString(ginurl)
	if err != nil {
		return nil, err
	}

	gitcfg, err := config.ParseGitString(giturl)
	if err != nil {
		return nil, err
	}

	srvcfg := config.ServerCfg{Web: webcfg, Git: gitcfg}
	hostkeystr, fingerprint, err := git.GetHostKey(gitcfg)
	if err != nil {
		return nil, err
	}
	srvcfg.Git.HostKey = hostkeystr
	log.Debugf("Got hostkey with fingerprint:\n%s", fingerprint)
	config.AddServerConf("gin", srvcfg)
	// Update known hosts file
	err = git.WriteKnownHosts()
	if err != nil {
		return nil, err
	}
	cfg.GIN.Username = libgin.ReadConf("ginuser")
	cfg.GIN.Password = libgin.ReadConf("ginpassword")

	cfg.GIN.Session = ginclient.New("gin")

	return &cfg, nil
}
