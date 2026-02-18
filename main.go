package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	version = "master"
	commit  = "none"
	date    = "unknown"
)

// Shared flags.
var (
	flagBeta     bool
	flagDevices  []string
	flagDomain   string
	flagExclude  []string
	flagForce    bool
	flagHTTPPort int
	flagJSON     bool
	flagModel    []string
	flagPassword string
	flagSubnets  []string
	flagUsername string
	flagVerbose  bool
	flagWaitTime int
)

func newOTAServiceFromFlags() (*OTAService, error) {
	return NewOTAService(
		WithBetaVersions(flagBeta),
		WithDomain(flagDomain),
		WithExcludeFilter(flagExclude),
		WithForcedUpgrades(flagForce),
		WithDevices(flagDevices),
		WithModelFilter(flagModel),
		WithPassword(flagPassword),
		WithServerPort(flagHTTPPort),
		WithSubnets(flagSubnets),
		WithUsername(flagUsername),
		WithWaitTimeInSeconds(flagWaitTime),
	)
}

func configureLogging() {
	if flagJSON {
		log.SetOutput(io.Discard)
	} else if flagVerbose {
		log.SetFormatter(&log.TextFormatter{DisableColors: true})
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

var rootCmd = &cobra.Command{
	Use:   "mota",
	Short: "Shelly firmware updater",
	Long:  "mota discovers Shelly devices on the local network and upgrades their firmware.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		configureLogging()
	},
	// Running bare `mota` with no subcommand behaves like `mota upgrade`.
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpgrade(cmd, args)
	},
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Discover devices and upgrade firmware",
	RunE:  runUpgrade,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Discover devices and list available updates",
	RunE:  runList,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("mota %s (%s %s)\n", version, commit, date)
	},
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	otaUpdater, err := newOTAServiceFromFlags()
	if err != nil {
		return err
	}

	err = otaUpdater.Setup()
	if err != nil {
		return err
	}
	defer otaUpdater.Shutdown()

	otaUpdater.FilterDevices()

	if flagJSON {
		return printJSON(otaUpdater.ListDeviceStatus())
	}

	err = otaUpdater.PromptForUpgrades()
	if err != nil {
		return err
	}

	log.Infof("Done!")
	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	otaUpdater, err := newOTAServiceFromFlags()
	if err != nil {
		return err
	}

	err = otaUpdater.Setup()
	if err != nil {
		return err
	}
	defer otaUpdater.Shutdown()

	otaUpdater.FilterDevices()

	statuses := otaUpdater.ListDeviceStatus()

	if flagJSON {
		return printJSON(statuses)
	}

	if len(statuses) == 0 {
		fmt.Println("No devices found.")
		return nil
	}

	fmt.Printf("%-40s %-14s %-20s %-20s %s\n", "DEVICE", "MODEL", "CURRENT", "TARGET", "NOTE")
	fmt.Printf("%-40s %-14s %-20s %-20s %s\n", "------", "-----", "-------", "------", "----")

	for _, s := range statuses {
		target := s.TargetVersion
		note := ""
		if s.UpToDate {
			target = "(up to date)"
		}
		if s.ManualUpgradeRequired {
			note = "manual upgrade required"
		} else if s.SteppingStone {
			note = "stepping-stone"
		}
		fmt.Printf("%-40s %-14s %-20s %-20s %s\n", s.Name, s.Model, s.CurrentVersion, target, note)
	}

	return nil
}

func addSharedFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&flagBeta, "beta", false, "Include beta firmwares in the list of available updates")
	cmd.Flags().StringVar(&flagDomain, "domain", "local", "Set the search domain for the local network")
	cmd.Flags().StringSliceVar(&flagDevices, "device", []string{}, "Use device IP address(es) instead of device discovery (can be specified multiple times or be comma-separated)")
	cmd.Flags().StringSliceVar(&flagExclude, "exclude", []string{}, "Exclude devices matching glob pattern(s) (can be specified multiple times or be comma-separated)")
	cmd.Flags().IntVarP(&flagHTTPPort, "http-port", "p", 0, "HTTP port to listen for OTA requests. If not specified, a random port is chosen")
	cmd.Flags().BoolVar(&flagJSON, "json", false, "Output results as JSON")
	cmd.Flags().StringSliceVar(&flagModel, "model", []string{}, "Only include devices matching model name(s) (can be specified multiple times or be comma-separated)")
	cmd.Flags().StringVar(&flagPassword, "password", "", "Global password for device authentication (fallback when no .netrc entry exists)")
	cmd.Flags().StringSliceVar(&flagSubnets, "subnet", []string{}, "Additional subnet(s) to scan in CIDR notation (e.g. 192.168.100.0/24)")
	cmd.Flags().StringVar(&flagUsername, "username", "", "Global username for device authentication (fallback when no .netrc entry exists)")
	cmd.Flags().IntVarP(&flagWaitTime, "wait", "w", 60, "Duration in [s] to run discovery")
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "Enable verbose mode")

	// Add shared flags to root (for backward compat running bare `mota --flags`).
	addSharedFlags(rootCmd)
	rootCmd.Flags().BoolVarP(&flagForce, "force", "f", false, "Force upgrades without asking for confirmation")

	// Add shared flags to upgrade subcommand.
	addSharedFlags(upgradeCmd)
	upgradeCmd.Flags().BoolVarP(&flagForce, "force", "f", false, "Force upgrades without asking for confirmation")

	// Add shared flags to list subcommand.
	addSharedFlags(listCmd)

	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
