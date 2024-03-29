package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/G-Node/gin-cli/ginclient"
	"github.com/G-Node/gin-cli/ginclient/config"
	"github.com/G-Node/gin-cli/git"
	"github.com/G-Node/libgin/libgin"
)

// Configuration is used to store and pass the configuration settings
// throughout the service.
type Configuration struct {
	// Port for the GIN DOI service to listen on
	Port uint16
	// The encryption key, shared with GIN Web for verification
	Key string
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
	// XMLRepo is the repository where the registered dataset XML files are
	// stored
	XMLRepo string
	// Settings related to the storage location for published data and landing
	// pages
	Storage struct {
		// Directory for cloning and zip creation
		PreparationDirectory string
		// Root storage location
		TargetDirectory string
		// URL where the published data is served from (used for email
		// notifications, linking, and redirection)
		StoreURL string
		// Used in email notification for convenient XML file retrieval (SCP
		// format host:/path/)
		XMLURL string
	}
	// LockedContentCutoffSize defines the git annex size above which a repository
	// containing locked annex files is no longer handled by the server.
	// The size number always refers to gigabytes.
	LockedContentCutoffSize float64
}

// parseconfigvars loads all DOI server config vars from the
// OS environment and handles default values and necessary
// type conversions.
func parseconfigvars(cfg *Configuration) error {
	cfg.DOIBase = libgin.ReadConf("doibase")

	cfg.Email.Server = libgin.ReadConf("mailserver")
	cfg.Email.From = libgin.ReadConf("mailfrom")
	cfg.Email.RecipientsFile = libgin.ReadConf("mailtofile")

	cfg.Storage.PreparationDirectory = libgin.ReadConf("preparation")
	cfg.Storage.TargetDirectory = libgin.ReadConf("target")
	cfg.Storage.StoreURL = libgin.ReadConf("storeurl")
	cfg.Storage.XMLURL = libgin.ReadConf("xmlurl")

	cfg.XMLRepo = libgin.ReadConf("xmlrepo")

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
		return err
	}

	cfg.Port = uint16(port)

	cutsize, err := strconv.ParseFloat(libgin.ReadConfDefault("lockedcutoffsize", "250.0"), 64)
	if err != nil {
		log.Printf("Error while parsing locked content cutoff size flag: %s", err.Error())
		log.Printf("Using default %.1f", 250.0)
		cutsize = 250.0
	}
	cfg.LockedContentCutoffSize = cutsize

	return nil
}

// loadconfig reads all the configuration variables (from the environment).
// It also creates and provides a gin client session to the specified
// gin server.
func loadconfig() (*Configuration, error) {
	cfg := Configuration{}

	// NOTE: Temporary workaround. GIN Client internals need a bit of a
	// redesign to support in-memory configurations.
	confdir := libgin.ReadConf("configdir")
	confdir, err := filepath.Abs(confdir)
	if err != nil {
		return nil, err
	}
	err = os.Setenv("GIN_CONFIG_DIR", confdir)
	if err != nil {
		log.Printf("Could not set GIN_CONFIG_DIR env: %q", err.Error())
	}

	err = parseconfigvars(&cfg)
	if err != nil {
		return nil, err
	}

	// Set up GIN client configuration (for cloning)
	ginurl := libgin.ReadConf("ginurl")
	giturl := libgin.ReadConf("giturl")
	log.Printf("gin: %s -- git: %s", ginurl, giturl)

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
	log.Printf("Got hostkey with fingerprint:\n%s", fingerprint)
	err = config.AddServerConf("gin", srvcfg)
	if err != nil {
		log.Printf("Could not add gin-cli server config: %q", err.Error())
	}
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
