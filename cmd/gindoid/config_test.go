package main

import (
	"os"
	"strings"
	"testing"
)

func TestParseConfigVars(t *testing.T) {
	cfg := Configuration{}

	// might be a better way to test server default values
	defport := uint16(10443)
	defqueue := 100
	defwork := 3
	defcutoff := 250.0

	// test no error on basic load
	err := parseconfigvars(&cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %q", err.Error())
	}

	// test default values in config
	if cfg.Port != defport || cfg.MaxQueue != defqueue || cfg.MaxWorkers != defwork {
		t.Fatalf("Encountered unexpected default value(s): port (%d), queue (%d), workers (%d)", cfg.Port, cfg.MaxQueue, cfg.MaxWorkers)
	}

	// test invalid port entry handling
	if err = os.Setenv("port", "abc"); err != nil {
		t.Fatalf("Error setting 'port': %q", err.Error())
	}
	err = parseconfigvars(&cfg)
	if err == nil {
		t.Fatal("Expected error on invalid port entry")
	} else if err != nil && !strings.Contains(err.Error(), "invalid syntax") {
		t.Fatalf("Unexpected port error: %q", err.Error())
	}

	if err = os.Setenv("port", "500000000"); err != nil {
		t.Fatalf("Error re-setting invalid 'port': %q", err.Error())
	}
	err = parseconfigvars(&cfg)
	if err == nil {
		t.Fatal("Expected error on invalid port entry")
	} else if err != nil && !strings.Contains(err.Error(), "value out of range") {
		t.Fatalf("Unexpected port error: %q", err.Error())
	}

	if err = os.Setenv("port", "12"); err != nil {
		t.Fatalf("Error re-setting valid 'port': %q", err.Error())
	}
	err = parseconfigvars(&cfg)
	if err != nil {
		t.Fatalf("Unexpected port set error: %q", err.Error())
	}
	if cfg.Port != 12 {
		t.Fatalf("Expected port '12' but got %d", cfg.Port)
	}

	// test maxqueue entry handling
	if err = os.Setenv("maxqueue", "abc"); err != nil {
		t.Fatalf("Error setting 'maxqueue': %q", err.Error())
	}
	err = parseconfigvars(&cfg)
	if err != nil {
		t.Fatalf("Unexpected maxqueue error: %q", err.Error())
	} else if cfg.MaxQueue != defqueue {
		t.Fatalf("Unexpected maxqueue default value: %d", cfg.MaxQueue)
	}

	// valid entry
	if err = os.Setenv("maxqueue", "50"); err != nil {
		t.Fatalf("Error re-setting 'maxqueue': %q", err.Error())
	}
	err = parseconfigvars(&cfg)
	if err != nil {
		t.Fatalf("Unexpected maxqueue error: %q", err.Error())
	} else if cfg.MaxQueue != 50 {
		t.Fatalf("Unexpected maxqueue value: %d", cfg.MaxQueue)
	}

	// check maxworkers entry handling
	if err = os.Setenv("maxworkers", "abc"); err != nil {
		t.Fatalf("Error setting 'maxworkers': %q", err.Error())
	}
	err = parseconfigvars(&cfg)
	if err != nil {
		t.Fatalf("Unexpected maxworkers error: %q", err.Error())
	} else if cfg.MaxWorkers != defwork {
		t.Fatalf("Unexpected maxworkers default value: %d", cfg.MaxWorkers)
	}

	// valid entry
	if err = os.Setenv("maxworkers", "5"); err != nil {
		t.Fatalf("Error re-setting 'maxworkers': %q", err.Error())
	}
	err = parseconfigvars(&cfg)
	if err != nil {
		t.Fatalf("Unexpected maxworkers error: %q", err.Error())
	} else if cfg.MaxWorkers != 5 {
		t.Fatalf("Unexpected maxworkers value: %d", cfg.MaxWorkers)
	}

	// check cutoff size entry handling
	if err = os.Setenv("lockedcutoffsize", "abc"); err != nil {
		t.Fatalf("Error setting 'cutoff': %q", err.Error())
	}
	err = parseconfigvars(&cfg)
	if err != nil {
		t.Fatalf("Unexpected cutoff error: %q", err.Error())
	} else if cfg.LockedContentCutoffSize != defcutoff {
		t.Fatalf("Unexpected cutoff default value: %.1f", cfg.LockedContentCutoffSize)
	}

	// valid entry
	if err = os.Setenv("lockedcutoffsize", "5"); err != nil {
		t.Fatalf("Error re-setting 'cutoff': %q", err.Error())
	}
	err = parseconfigvars(&cfg)
	if err != nil {
		t.Fatalf("Unexpected cutoff error: %q", err.Error())
	} else if cfg.LockedContentCutoffSize != 5 {
		t.Fatalf("Unexpected cutoff value: %.1f", cfg.LockedContentCutoffSize)
	}

	// test no panic on unset variables
	// check access of all config field after loading
	if cfg.DOIBase != "" {
		t.Fatalf("Unexpected DOIbase %q", cfg.DOIBase)
	}
	if cfg.Key != "" {
		t.Fatalf("Unexpected Key %q", cfg.Key)
	}
	if cfg.XMLRepo != "" {
		t.Fatalf("Unexpected XMLRepo %q", cfg.XMLRepo)
	}
	if cfg.Email.Server != "" {
		t.Fatalf("Unexpected Email.Server %q", cfg.Email.Server)
	}
	if cfg.Email.From != "" {
		t.Fatalf("Unexpected Email.From %q", cfg.Email.From)
	}
	if cfg.Email.RecipientsFile != "" {
		t.Fatalf("Unexpected Email.RecipientsFile %q", cfg.Email.RecipientsFile)
	}
	if cfg.GIN.Password != "" {
		t.Fatalf("Unexpected GIN.Password %q", cfg.GIN.Password)
	}
	if cfg.GIN.Username != "" {
		t.Fatalf("Unexpected GIN.Username %q", cfg.GIN.Username)
	}
	if cfg.Storage.PreparationDirectory != "" {
		t.Fatalf("Unexpected prep dir %q", cfg.Storage.PreparationDirectory)
	}
	if cfg.Storage.StoreURL != "" {
		t.Fatalf("Unexpected StoreURL %q", cfg.Storage.StoreURL)
	}
	if cfg.Storage.TargetDirectory != "" {
		t.Fatalf("Unexpected TargetDirectory %q", cfg.Storage.TargetDirectory)
	}
	if cfg.Storage.XMLURL != "" {
		t.Fatalf("Unexpected XMLURL %q", cfg.Storage.XMLURL)
	}
}

func TestLoadconfig(t *testing.T) {
	// check 'configdir' env var
	_, err := loadconfig()
	if err == nil {
		t.Fatal("Expected error on missing 'configdir' env var")
	}

	confpath := t.TempDir()
	err = os.Setenv("configdir", confpath)
	if err != nil {
		t.Fatalf("Error setting 'confdir': %q", err.Error())
	}

	// check 'ginurl' env var
	_, err = loadconfig()
	if err == nil {
		t.Fatal("Expected error on missing 'ginurl' env var")
	} else if err != nil && !strings.Contains(err.Error(), "invalid web configuration") {
		t.Fatalf("Error loading config: %q", err.Error())
	}

	err = os.Setenv("ginurl", "invalidurl")
	if err != nil {
		t.Fatalf("Error setting 'ginurl': %q", err.Error())
	}
	_, err = loadconfig()
	if err == nil {
		t.Fatal("Expected error on invalid 'ginurl' env var")
	} else if err != nil && !strings.Contains(err.Error(), "invalid web configuration") {
		t.Fatalf("Error loading config: %q", err.Error())
	}

	// check invalid giturl
	// we are not testing the gin-cli.config.ParseWebString here
	err = os.Setenv("ginurl", "https://a.valid.url:1221")
	if err != nil {
		t.Fatalf("Error setting 'ginurl': %q", err.Error())
	}
	_, err = loadconfig()
	if err == nil {
		t.Fatal("Expected error on invalid 'giturl' env var")
	} else if err != nil && !strings.Contains(err.Error(), "invalid git configuration") {
		t.Fatalf("Error loading config: %q", err.Error())
	}

	// check error on invalid server setup
	err = os.Setenv("giturl", "git@a.valid.url:2222")
	if err != nil {
		t.Fatalf("Error setting 'giturl': %q", err.Error())
	}
	_, err = loadconfig()
	if err == nil {
		t.Fatal("Expected error on invalid server configuration")
	} else if err != nil && !strings.Contains(err.Error(), "no such host") {
		t.Fatalf("Error loading config: %q", err.Error())
	}

	// further tests require a local git server.
}
