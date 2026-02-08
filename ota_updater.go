package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	log "github.com/sirupsen/logrus"
)

const (
	defaultDomain            = "local"
	defaultIncludeBetas      = false
	defaultService           = "_http._tcp."
	defaultWaitTimeInSeconds = 60
)

// OTAService is the structure that keeps a cache of the discovered
// devices and allows orchestration of upgrades.
type OTAService struct {
	api               *APIClient
	browser           Browser
	devices           map[string]*Device
	domain            string
	downloadDir       string
	force             bool
	serverPort        int
	listener          net.Listener
	includeBetas      bool
	hosts             []string
	serverIP          net.IP
	service           string
	waitTimeInSeconds int
	muxServer         *http.ServeMux
	server            *http.Server
	deviceHandlers    sync.Map // map[string]http.HandlerFunc
	modelFilter       []string
	excludePatterns   []string
	subnets           []string
	username          string
	password          string
}

// OTAServiceOption is an option interface for OTAUpdater.
type OTAServiceOption func(*OTAService)

// WithAPIClient is an OTAUpdater option that allows overriding the
// APIClient used to interact with the Shelly API.
func WithAPIClient(api *APIClient) OTAServiceOption {
	return func(o *OTAService) {
		o.api = api
	}
}

// WithWaitTimeInSeconds
func WithWaitTimeInSeconds(waitTimeInSeconds int) OTAServiceOption {
	return func(o *OTAService) {
		o.waitTimeInSeconds = waitTimeInSeconds
	}
}

// WithForcedUpgrades is an OTAUpdater option that allows overriding
// the default behaviour of confirming upgrades interactively.
func WithForcedUpgrades(force bool) OTAServiceOption {
	return func(o *OTAService) {
		o.force = force
	}
}

// WithBeta is an OTAUpdater option that enables beta
// versions, if available.
func WithBetaVersions(beta bool) OTAServiceOption {
	return func(o *OTAService) {
		o.includeBetas = beta
	}
}

// WithService
func WithService(service string) OTAServiceOption {
	return func(o *OTAService) {
		o.service = service
	}
}

// WithDomain
func WithDomain(domain string) OTAServiceOption {
	return func(o *OTAService) {
		o.domain = domain
	}
}

// WithServerPort
func WithServerPort(serverPort int) OTAServiceOption {
	return func(o *OTAService) {
		o.serverPort = serverPort
	}
}

// WithDevices
func WithDevices(hosts []string) OTAServiceOption {
	return func(o *OTAService) {
		o.hosts = hosts
	}
}

// WithUsername is an OTAServiceOption that sets a global username
// used as fallback when no .netrc entry exists for a device.
func WithUsername(username string) OTAServiceOption {
	return func(o *OTAService) {
		o.username = username
	}
}

// WithPassword is an OTAServiceOption that sets a global password
// used as fallback when no .netrc entry exists for a device.
func WithPassword(password string) OTAServiceOption {
	return func(o *OTAService) {
		o.password = password
	}
}

// NewOTAService returns an instance of OTAUpdater with the default
// options. Firmware downloads are stored on the OS cache or temp
// directories.
func NewOTAService(options ...OTAServiceOption) (*OTAService, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}

	serverIP, err := ServerIP()
	if err != nil {
		return nil, err
	}

	otaService := &OTAService{
		api:               NewAPIClient(),
		domain:            defaultDomain,
		downloadDir:       filepath.Join(cacheDir, "com.github.ruimarinho.mota"),
		includeBetas:      defaultIncludeBetas,
		serverIP:          serverIP,
		service:           defaultService,
		waitTimeInSeconds: defaultWaitTimeInSeconds,
	}

	// Apply custom OTAUpdaterOptions.
	for _, option := range options {
		option(otaService)
	}

	listener, err := ServerListener(otaService.serverPort)
	if err != nil {
		return nil, err
	}
	otaService.listener = listener
	otaService.serverPort = listener.Addr().(*net.TCPAddr).Port

	otaService.browser = Browser{
		domain:   otaService.domain,
		service:  otaService.service,
		waitTime: otaService.waitTimeInSeconds,
		subnets:  otaService.subnets,
		username: otaService.username,
		password: otaService.password,
	}

	if otaService.includeBetas {
		otaService.api.includeBetas = true
	}

	return otaService, nil
}

// Setup is the main orchestrator of device updates. First, it
// discovers them and then, for each model found, it fetches the
// most recent firmware available. If there are any devices of that
// model available for update, it downloads that firmware and installs
// a handler on the local OTA server to serve it when requested by the
// device OTA service.
func (o *OTAService) Setup() error {
	o.muxServer = http.NewServeMux()
	o.muxServer.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		deviceID := r.URL.Path[1:]
		if handler, ok := o.deviceHandlers.Load(deviceID); ok {
			handler.(http.HandlerFunc)(w, r)
			return
		}
		http.NotFound(w, r)
	})
	o.server = &http.Server{Handler: o.muxServer}

	go o.server.Serve(o.listener)

	log.Infof("OTA HTTP server listening on port %v", o.serverPort)

	devices, err := o.DiscoverDevices()
	if err != nil {
		return err
	}

	// Detect which models have newer versions available.
	for _, device := range devices {
		remoteFirmware, err := o.api.GetLatestFirmwareAvailable(device.Model)
		if err != nil {
			log.Warnf("No remote firmware available for %v (model %v), skipping", device.String(), device.Model)
			continue
		}

		// Check if stepping-stone upgrade is required.
		if steppingStone, needed := NeedsSteppingStone(o.devices[device.ID]); needed {
			log.Warnf("%v requires stepping-stone upgrade to %v before upgrading to %v",
				device.String(), steppingStone.Version, remoteFirmware.Version)
			o.devices[device.ID].FirmwareNewestVersion = steppingStone
			continue
		}

		// Only set the model flag if a discovered device has an out-of-date firmware.
		deviceVersion := extractSemanticVersion(o.devices[device.ID].FirmwareVersion)
		remoteVersion := extractSemanticVersion(remoteFirmware.Version)
		remoteBetaVersion := extractSemanticVersion(remoteFirmware.BetaVersion)
		if isVersionLessThan(deviceVersion, remoteVersion) || (o.includeBetas && remoteBetaVersion != "" && isVersionLessThan(deviceVersion, remoteBetaVersion)) {
			o.devices[device.ID].FirmwareNewestVersion = remoteFirmware
		}
	}

	return nil
}

// Shutdown gracefully shuts down the OTA HTTP server, waiting up to 5
// seconds for in-flight requests to complete.
func (o *OTAService) Shutdown() {
	if o.server == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := o.server.Shutdown(ctx); err != nil {
		log.Warnf("OTA HTTP server shutdown error: %v", err)
	} else {
		log.Debugf("OTA HTTP server shut down gracefully")
	}
}

// DiscoverDevices returns a list of discovered devices on the local network
// along with their current settings state.
func (o *OTAService) DiscoverDevices() (map[string]*Device, error) {
	// Run discovery once only.
	if o.devices != nil {
		return o.devices, nil
	}

	// Use mDNS/zeroconf to listen for device announcement.
	devices, err := o.browser.ListenForAnnouncements(o.hosts)
	if err != nil {
		return nil, err
	}

	o.devices = map[string]*Device{}
	for i, device := range devices {
		o.devices[device.ID] = &devices[i]
	}

	return o.devices, nil
}

// UpgradeDevice requests a device to be upgraded by asking it
// to contact the OTA server for the most recent firmware version.
func (o *OTAService) UpgradeDevice(device *Device, filename string, wg *sync.WaitGroup) error {
	defer wg.Done()
	url := device.OTAURL(o.serverIP.String(), o.serverPort, device.ID)

	log.Debugf("Adding HTTP handler for /%s", device.ID)

	doneCh := make(chan bool)
	o.deviceHandlers.Store(device.ID, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		notifier := r.Context().Done()

		log.Debugf("Serving file %v to %v", filename, r.RemoteAddr)
		http.ServeFile(w, r, filename)
		<-notifier
		doneCh <- true

		log.Debugf("Served file %v to %v", filename, r.RemoteAddr)
	}))

	log.Debugf("Making OTA request to %s to serve local firmware %s", url, filename)

	response, err := http.Get(url)
	if err != nil {
		log.Debug(err)
		return err
	}

	defer response.Body.Close()
	responseContent, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error(err)
		return err
	}

	log.Debugf("Received OTA response: %s", string(responseContent))

	select {
	case <-doneCh:
		log.Debugf("Completed OTA request")
	case <-time.After(time.Second * 120):
		log.Warn(fmt.Sprintf("Client did not complete OTA request within 120 seconds. Network might be unreachable or device is too busy to acknowledge the OTA request. Check the UI at http://%s for more details.", device.IP))
		return nil
	}

	o.verifyUpgrade(device)

	return nil
}

// verifyUpgrade polls a device after OTA to confirm it rebooted with
// the expected firmware version. It retries with backoff to allow time
// for the device to reboot.
func (o *OTAService) verifyUpgrade(device *Device) {
	expectedVersion := device.FirmwareNewestVersion.Version

	log.Infof("Waiting for %v to reboot and verify firmware...", device.String())

	client := &http.Client{Timeout: 5 * time.Second}
	baseURL := fmt.Sprintf("http://%v:%v", device.IP.String(), device.Port)

	delays := []time.Duration{
		10 * time.Second,
		10 * time.Second,
		15 * time.Second,
		15 * time.Second,
		30 * time.Second,
	}

	for attempt, delay := range delays {
		time.Sleep(delay)
		log.Debugf("Verification attempt %d for %v", attempt+1, device.String())

		var currentVersion string
		if device.Generation == 1 {
			currentVersion = o.fetchGen1Version(client, baseURL, device)
		} else {
			currentVersion = o.fetchGen2Version(client, baseURL, device)
		}

		if currentVersion == "" {
			log.Debugf("Device %v not yet reachable (attempt %d/%d)", device.String(), attempt+1, len(delays))
			continue
		}

		if currentVersion == expectedVersion {
			log.Infof("Verified %v is now running firmware %v", device.String(), currentVersion)
			return
		}

		log.Debugf("Device %v reports firmware %v, expected %v (attempt %d/%d)",
			device.String(), currentVersion, expectedVersion, attempt+1, len(delays))
	}

	log.Warnf("Could not verify firmware upgrade for %v. Expected %v. Device may still be rebooting.",
		device.String(), expectedVersion)
}

func (o *OTAService) fetchGen1Version(client *http.Client, baseURL string, device *Device) string {
	resp, err := client.Get(fmt.Sprintf("%s/settings", baseURL))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var settings Gen1Settings
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return ""
	}

	return settings.Firmware
}

func (o *OTAService) fetchGen2Version(client *http.Client, baseURL string, device *Device) string {
	resp, err := client.Get(fmt.Sprintf("%s/rpc/Shelly.GetDeviceInfo", baseURL))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var settings Gen2Settings
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return ""
	}

	return settings.Firmware
}

// PromptForUpgrades prompts the end-user to decide whether or not to
// perform an upgrade of a device. When stepping-stone upgrades are
// performed, it automatically re-discovers devices and continues
// upgrading until all devices reach their latest firmware.
func (o *OTAService) PromptForUpgrades() error {
	for pass := 1; ; pass++ {
		if pass > 1 {
			log.Infof("Re-evaluating devices after stepping-stone upgrades (pass %d)...", pass)
		}

		hadSteppingStone, err := o.upgradePass()
		if err != nil {
			return err
		}

		if !hadSteppingStone {
			break
		}

		// Reset discovery so the next pass re-queries device firmware versions.
		o.ResetDiscovery()

		err = o.refreshFirmwareTargets()
		if err != nil {
			return err
		}

		o.FilterDevices()
	}

	return nil
}

// upgradePass runs a single round of upgrades. It returns true if any
// stepping-stone upgrades were performed, signalling that another pass
// may be needed.
func (o *OTAService) upgradePass() (bool, error) {
	devices, err := o.DiscoverDevices()
	if err != nil {
		return false, err
	}

	hadSteppingStone := false
	var wg sync.WaitGroup

	for _, device := range devices {
		if (device.FirmwareNewestVersion == RemoteFirmware{}) {
			log.Infof("Skipping %v as firmware version %v is the latest available", device.String(), device.FirmwareVersion)
			continue
		}

		isBeta := false
		if o.force {
			if o.includeBetas {
				isBeta = true
			}
		} else {
			upgrade := false
			var chosenUpdate string
			if o.includeBetas {
				prompt := &survey.Select{
					Message: fmt.Sprintf("Which firmware version would you like to upgrade %v to?", device.String()),
					Options: []string{device.FirmwareNewestVersion.Version, device.FirmwareNewestVersion.BetaVersion},
				}
				err := survey.AskOne(prompt, &chosenUpdate, survey.WithValidator(survey.Required))
				if err == terminal.InterruptErr {
					break
				} else if err != nil {
					return false, err
				}
			} else {
				chosenUpdate = device.FirmwareNewestVersion.Version
			}

			if chosenUpdate == device.FirmwareNewestVersion.BetaVersion {
				isBeta = true
			}

			message := fmt.Sprintf("Would you like to upgrade %v from %v to %v?", device.String(), device.FirmwareVersion, chosenUpdate)
			if _, needed := NeedsSteppingStone(device); needed {
				message = fmt.Sprintf("Would you like to upgrade %v from %v to %v (required stepping-stone before latest)?",
					device.String(), device.FirmwareVersion, chosenUpdate)
			}

			prompt := &survey.Confirm{
				Message: message,
			}

			err := survey.AskOne(prompt, &upgrade, survey.WithValidator(survey.Required))
			if err == terminal.InterruptErr {
				break
			} else if err != nil {
				return false, err
			}

			if !upgrade {
				continue
			}
		}

		filepath, err := o.api.DownloadFirmware(device.FirmwareNewestVersion, isBeta, o.downloadDir)
		if err != nil {
			return false, fmt.Errorf("error downloading firmware")
		}

		if _, needed := NeedsSteppingStone(device); needed {
			hadSteppingStone = true
		}

		wg.Add(1)
		go o.UpgradeDevice(device, filepath, &wg)
	}
	wg.Wait()

	return hadSteppingStone, nil
}

// ResetDiscovery clears the cached device list so that the next call
// to DiscoverDevices performs a fresh discovery.
func (o *OTAService) ResetDiscovery() {
	o.devices = nil
}

// refreshFirmwareTargets re-discovers devices and re-evaluates which
// firmware version each device should be upgraded to. This is used
// after stepping-stone upgrades to determine if further upgrades are
// needed.
func (o *OTAService) refreshFirmwareTargets() error {
	devices, err := o.DiscoverDevices()
	if err != nil {
		return err
	}

	for _, device := range devices {
		remoteFirmware, err := o.api.GetLatestFirmwareAvailable(device.Model)
		if err != nil {
			log.Warnf("No remote firmware available for %v (model %v), skipping", device.String(), device.Model)
			continue
		}

		if steppingStone, needed := NeedsSteppingStone(o.devices[device.ID]); needed {
			log.Warnf("%v requires stepping-stone upgrade to %v before upgrading to %v",
				device.String(), steppingStone.Version, remoteFirmware.Version)
			o.devices[device.ID].FirmwareNewestVersion = steppingStone
			continue
		}

		deviceVersion := extractSemanticVersion(o.devices[device.ID].FirmwareVersion)
		remoteVersion := extractSemanticVersion(remoteFirmware.Version)
		remoteBetaVersion := extractSemanticVersion(remoteFirmware.BetaVersion)
		if isVersionLessThan(deviceVersion, remoteVersion) || (o.includeBetas && remoteBetaVersion != "" && isVersionLessThan(deviceVersion, remoteBetaVersion)) {
			o.devices[device.ID].FirmwareNewestVersion = remoteFirmware
		}
	}

	return nil
}

// DeviceStatus holds a summary of a device's upgrade state for display.
type DeviceStatus struct {
	Name           string `json:"name"`
	ID             string `json:"id"`
	Model          string `json:"model"`
	CurrentVersion string `json:"current_version"`
	TargetVersion  string `json:"target_version,omitempty"`
	UpToDate       bool   `json:"up_to_date"`
	SteppingStone  bool   `json:"stepping_stone"`
}

// ListDeviceStatus returns a list of DeviceStatus entries after Setup has
// been called. Each entry summarises the upgrade state of a discovered device.
func (o *OTAService) ListDeviceStatus() []DeviceStatus {
	var statuses []DeviceStatus
	for _, device := range o.devices {
		ds := DeviceStatus{
			Name:           device.String(),
			ID:             device.ID,
			Model:          device.Model,
			CurrentVersion: device.FirmwareVersion,
		}

		if (device.FirmwareNewestVersion == RemoteFirmware{}) {
			ds.UpToDate = true
		} else {
			ds.TargetVersion = device.FirmwareNewestVersion.Version
			if _, needed := NeedsSteppingStone(device); needed {
				ds.SteppingStone = true
			}
		}

		statuses = append(statuses, ds)
	}
	return statuses
}

// WithSubnets is an OTAServiceOption that specifies additional subnets
// to scan via HTTP for device discovery.
func WithSubnets(subnets []string) OTAServiceOption {
	return func(o *OTAService) {
		o.subnets = subnets
	}
}

// WithModelFilter is an OTAServiceOption that restricts operations to
// devices matching any of the given model names.
func WithModelFilter(models []string) OTAServiceOption {
	return func(o *OTAService) {
		o.modelFilter = models
	}
}

// WithExcludeFilter is an OTAServiceOption that excludes devices whose
// name matches any of the given glob patterns.
func WithExcludeFilter(patterns []string) OTAServiceOption {
	return func(o *OTAService) {
		o.excludePatterns = patterns
	}
}

// FilterDevices removes devices that do not match the configured model
// filter or that match an exclude pattern. It should be called after
// Setup().
func (o *OTAService) FilterDevices() {
	if len(o.modelFilter) == 0 && len(o.excludePatterns) == 0 {
		return
	}

	for id, device := range o.devices {
		if len(o.modelFilter) > 0 && !containsString(o.modelFilter, device.Model) {
			log.Debugf("Filtering out %v: model %v not in filter", device.String(), device.Model)
			delete(o.devices, id)
			continue
		}

		if matchesAnyPattern(o.excludePatterns, device.String()) ||
			matchesAnyPattern(o.excludePatterns, device.Name) ||
			matchesAnyPattern(o.excludePatterns, device.ID) {
			log.Debugf("Filtering out %v: matched exclude pattern", device.String())
			delete(o.devices, id)
		}
	}
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func matchesAnyPattern(patterns []string, name string) bool {
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}
