package main

import (
	"flag"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

var (
	version = "master"
	commit  = "none"
	date    = "unknown"
)

var (
	domain      = flag.String("domain", "local", "Set the search domain for the local network.")
	waitTime    = flag.Int("wait", 60, "Duration in [s] to run discovery.")
	httpPort    = flag.Int("http-port", 0, "HTTP port to listen for OTA requests. If not specified, a random port is chosen.")
	verbose     = flag.Bool("verbose", false, "Enable verbose mode.")
	showVersion = flag.Bool("version", false, "Show version information")
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

	updater, err := NewOTAUpdater(*httpPort, "_http._tcp.", *domain, *waitTime)
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
