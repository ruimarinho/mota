package main

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	version = "master"
	commit  = "none"
	date    = "unknown"
)

var (
	domain      = flag.String("domain", "local", "Set the search domain for the local network.")
	waitTime    = flag.IntP("wait", "w", 60, "Duration in [s] to run discovery.")
	httpPort    = flag.IntP("http-port", "p", 0, "HTTP port to listen for OTA requests. If not specified, a random port is chosen.")
	verbose     = flag.Bool("verbose", false, "Enable verbose mode.")
	showVersion = flag.BoolP("version", "v", false, "Show version information")
	force       = flag.BoolP("force", "f", false, "Force upgrades without asking for confirmation")
)

func main() {
	flag.Parse()

	// Only log the warning severity or above when verbose mode is disabled.
	if *verbose {
		log.SetFormatter(&log.TextFormatter{DisableColors: true})
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if *showVersion {
		fmt.Printf("shelly-updater %s (%s %s)\n", version, commit, date)
		os.Exit(0)
	}

	updater, err := NewOTAUpdater(*httpPort, "_http._tcp.", *domain, *waitTime, WithForcedUpgrades(*force))
	if err != nil {
		log.Fatal(err)
	}

	err = updater.Start()
	if err != nil {
		log.Fatal(err)
	}

	err = updater.PromptForUpgrade()
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("Done!")
}
