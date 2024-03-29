package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	log "github.com/sirupsen/logrus"
)

// OTAUpdater is the structure that keeps a cache of the discovered
// devices and allows orchestration of upgrades.
type OTAUpdater struct {
	api               *APIClient
	browser           Browser
	devices           map[string]*Device
	domain            string
	downloadDir       string
	force             bool
	serverPort        int
	includeBetas      bool
	hosts             []string
	serverIP          net.IP
	service           string
	waitTimeInSeconds int
}

// OTAUpdaterOption is an option interface for OTAUpdater.
type OTAUpdaterOption func(*OTAUpdater)

// WithAPIClient is an OTAUpdater option that allows overriding the
// APIClient used to interact with the Shelly API.
func WithAPIClient(api *APIClient) OTAUpdaterOption {
	return func(o *OTAUpdater) {
		o.api = api
	}
}

// WithWaitTimeInSeconds
func WithWaitTimeInSeconds(waitTimeInSeconds int) OTAUpdaterOption {
	return func(o *OTAUpdater) {
		o.waitTimeInSeconds = waitTimeInSeconds
	}
}

// WithForcedUpgrades is an OTAUpdater option that allows overriding
// the default behaviour of confirming upgrades interactively.
func WithForcedUpgrades(force bool) OTAUpdaterOption {
	return func(o *OTAUpdater) {
		o.force = force
	}
}

// WithBeta is an OTAUpdater option that enables beta
// versions, if available.
func WithBetaVersions(beta bool) OTAUpdaterOption {
	return func(o *OTAUpdater) {
		o.includeBetas = beta
	}
}

// WithService
func WithService(service string) OTAUpdaterOption {
	return func(o *OTAUpdater) {
		o.service = service
	}
}

// WithDomain
func WithDomain(domain string) OTAUpdaterOption {
	return func(o *OTAUpdater) {
		o.domain = domain
	}
}

// WithServerPort
func WithServerPort(serverPort int) OTAUpdaterOption {
	return func(o *OTAUpdater) {
		o.serverPort = serverPort
	}
}

// WithHosts
func WithHosts(hosts []string) OTAUpdaterOption {
	return func(o *OTAUpdater) {
		o.hosts = hosts
	}
}

// NewOTAUpdater returns an instance of OTAUpdater with the default
// options. Firmware downloads are stored on the OS cache or temp
// directories.
func NewOTAUpdater(options ...OTAUpdaterOption) (OTAUpdater, error) {
	const (
		defaultDomain            = "local"
		defaultIncludeBetas      = false
		defaultService           = "_http._tcp."
		defaultWaitTimeInSeconds = 60
	)

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}

	serverIP, err := ServerIP()
	if err != nil {
		return OTAUpdater{}, err
	}

	updater := OTAUpdater{
		api:          NewAPIClient(),
		downloadDir:  filepath.Join(cacheDir, "com.github.ruimarinho.mota"),
		includeBetas: defaultIncludeBetas,
		serverIP:     serverIP,
	}

	// Apply custom OTAUpdaterOptions.
	for _, option := range options {
		option(&updater)
	}

	if updater.serverPort == 0 {
		serverPort, err := ServerPort()
		updater.serverPort = serverPort

		if err != nil {
			return OTAUpdater{}, err
		}
	}

	updater.browser = Browser{updater.domain, updater.service, updater.waitTimeInSeconds}

	if updater.includeBetas {
		updater.api.includeBetas = true
	}

	return updater, nil
}

// Start is the main orchestrator of device updates. First, it
// discovers them and then, for each model found, it fetches the
// most recent firmware available. If there are any devices of that
// model available for update, it downloads that firmware and installs
// a handler on the local OTA server to serve it when requested by the
// device OTA service.
func (o *OTAUpdater) Start() error {
	log.Infof("Listening for HTTP server on port %v", o.serverPort)
	mux := http.NewServeMux()
	server := &http.Server{Addr: fmt.Sprintf(":%v", o.serverPort), Handler: mux}
	go server.ListenAndServe()

	devices, err := o.Devices()
	if err != nil {
		return err
	}

	firmwares, err := o.api.FetchVersions()
	if err != nil {
		return err
	}

	models := make(map[string]bool)
	for _, device := range devices {
		newFWVersion, err := o.api.GetVersion(device.Model)
		if err != nil {
			return err
		}

		o.devices[device.IP.String()].NewFWVersion = newFWVersion

		// If a model has already been marked as seen or out-of-date, make sure to respect
		// the flag independently of what future devices may suggest.
		if models[device.Model] {
			continue
		}

		// Only set the model flag if a discovered device has an out-of-date firmware,
		// otherwise its firmware will be downloaded and not used.
		if o.devices[device.IP.String()].CurrentFWVersion != newFWVersion {
			models[device.Model] = true
		}
	}

	var wg sync.WaitGroup
	for model, firmware := range firmwares {
		if !models[model] {
			log.Debugf("Skipping model %v as devices of this type have not been found on the local network or firmware is up-to-date", model)
			continue
		}

		wg.Add(1)
		go func(model string, firmware Firmware) {
			defer wg.Done()

			filename, err := o.DownloadFirmware(model, firmware)
			if err != nil {
				log.Errorf("Unable to download firmware for %v (%v)", firmware.Model, err)
				return
			}

			log.Debugf("Adding HTTP handler for /%v", model)

			mux.HandleFunc("/"+model, func(w http.ResponseWriter, r *http.Request) {
				log.Debugf("Serving file %v to %v", filename, r.RemoteAddr)
				http.ServeFile(w, r, filename)
			})
		}(model, firmware)
	}
	wg.Wait()

	return nil
}

// DownloadFirmware returns the final destination of the firmware that
// it has been requested to download for a particular model.
func (o *OTAUpdater) DownloadFirmware(model string, firmware Firmware) (string, error) {
	body, err := o.api.FetchFirmware(model)
	if err != nil {
		return "", err
	}

	defer body.Close()

	err = os.MkdirAll(o.downloadDir, 0700)
	if err != nil {
		return "", err
	}

	newFWVersion, err := o.api.GetVersion(model)
	if err != nil {
		return "", err
	}

	newFWURL, err := o.api.GetURL(model)
	if err != nil {
		return "", err
	}

	filename := strings.Join([]string{strings.Join([]string{model, strings.Replace(newFWVersion, "/", "-", -1)}, "-"), path.Ext(newFWURL)}, "")
	out, err := os.Create(filepath.Join(o.downloadDir, filename))
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, body)
	if err != nil {
		return "", err
	}

	log.Debugf("Downloaded firmware %v to %v\n", path.Base(newFWURL), filepath.Join(o.downloadDir, filename))

	return filepath.Join(o.downloadDir, filename), nil
}

// Devices returns a list of discovered devices on the local network
// along with their current settings state.
func (o *OTAUpdater) Devices() (map[string]*Device, error) {
	if o.devices != nil {
		return o.devices, nil
	}

	devices, err := o.browser.DiscoverDevices(o.hosts)
	if err != nil {
		return nil, err
	}

	o.devices = map[string]*Device{}
	for i, device := range devices {
		o.devices[device.IP.String()] = &devices[i]
	}

	return o.devices, nil
}

// UpgradeDevice requests a device to be upgraded by asking it
// to contact the OTA server for the most recent firmware version.
func (o *OTAUpdater) UpgradeDevice(device *Device) error {
	url := fmt.Sprintf("%s/ota?url=http://%s:%d/%s", device.GetBaseURL(), o.serverIP.String(), o.serverPort, device.Model)

	log.Debugf("Making OTA request to %s", url)

	response, err := http.Get(url)
	if err != nil {
		log.Debug(err)
		return err
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error(err)
		return err
	}

	log.Debugf("Received OTA response: %s", string(responseData))

	defer response.Body.Close()

	time.Sleep(10 * time.Second)

	return nil
}

// Upgrade prompts the end-user to decide whether or not to
// perform an upgrade of a device.
func (o *OTAUpdater) Upgrade() error {
	devices, err := o.Devices()
	if err != nil {
		return err
	}

	for _, device := range devices {
		if device.CurrentFWVersion == device.NewFWVersion {
			log.Infof("Skipping %v (%v) as firmware version is up-to-date (%v)", device.ModelName(), device.IP, device.CurrentFWVersion)
			continue
		}

		upgrade := false

		if !o.force {
			prompt := &survey.Confirm{
				Message: fmt.Sprintf("Would you like to upgrade %v (%v) from %v to %v?", device.ModelName(), device.IP, device.CurrentFWVersion, device.NewFWVersion),
			}

			err := survey.AskOne(prompt, &upgrade)
			if err == terminal.InterruptErr {
				break
			} else if err != nil {
				return err
			}

			if !upgrade {
				continue
			}
		}

		o.UpgradeDevice(device)
	}

	return nil
}
