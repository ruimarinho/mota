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
	beta        = flag.Bool("beta", false, "Use beta firmwares if available")
	domain      = flag.String("domain", "local", "Set the search domain for the local network.")
	force       = flag.BoolP("force", "f", false, "Force upgrades without asking for confirmation")
	hosts       = flag.StringSlice("host", []string{}, "Use host/IP address(es) instead of device discovery (can be specified multiple times or be comma-separated)")
	httpPort    = flag.IntP("http-port", "p", 0, "HTTP port to listen for OTA requests. If not specified, a random port is chosen.")
	showVersion = flag.BoolP("version", "v", false, "Show version information")
	verbose     = flag.Bool("verbose", false, "Enable verbose mode.")
	waitTime    = flag.IntP("wait", "w", 60, "Duration in [s] to run discovery.")
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
		fmt.Printf("mota %s (%s %s)\n", version, commit, date)
		os.Exit(0)
	}

	otaUpdater, err := NewOTAUpdater(
		WithBetaVersions(*beta),
		WithDomain(*domain),
		WithForcedUpgrades(*force),
		WithHosts(*hosts),
		WithServerPort(*httpPort),
		WithWaitTimeInSeconds(*waitTime),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = otaUpdater.Start()
	if err != nil {
		log.Fatal(err)
	}

	err = otaUpdater.Upgrade()
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("Done!")
}
