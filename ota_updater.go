package main

import (
	"fmt"
	"io"
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
	api         *APIClient
	browser     Browser
	devices     map[string]*Device
	downloadDir string
	httpPort    int
	serverIP    net.IP
}

// OTAUpdaterOption is an option interface for OTAUpdater.
type OTAUpdaterOption func(*OTAUpdater)

// WithAPIClient is an OTAUpdater option that allows overriding the
// APIClient used to interact with the Shelly API.
func WithAPIClient(api *APIClient) func(*OTAUpdater) {
	return func(o *OTAUpdater) {
		o.api = api
	}
}

// NewOTAUpdater returns an instance of OTAUpdater with the default
// options. Firmware downloads are stored on the OS cache or temp
// directories.
func NewOTAUpdater(httpPort int, service string, domain string, waitTime int, options ...OTAUpdaterOption) (OTAUpdater, error) {
	downloadDir, err := os.UserCacheDir()
	if err != nil {
		downloadDir = os.TempDir()
	}

	serverIP, err := ServerIP()
	if err != nil {
		return OTAUpdater{}, err
	}

	serverPort := httpPort
	if serverPort == 0 {
		serverPort, err = ServerPort()
		if err != nil {
			return OTAUpdater{}, err
		}
	}

	updater := OTAUpdater{
		api:         NewAPIClient(),
		browser:     Browser{domain, service, waitTime},
		httpPort:    serverPort,
		downloadDir: filepath.Join(downloadDir, "io.github.ruimarinho.shelly-updater"),
		serverIP:    serverIP,
	}

	// Apply custom OTAUpdaterOptions.
	for i := range options {
		options[i](&updater)
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
	log.Debugf("Listening for HTTP server on port %v", o.httpPort)
	go http.ListenAndServe(fmt.Sprintf(":%v", o.httpPort), nil)

	devices, err := o.Devices()
	if err != nil {
		return err
	}

	firmwares, err := o.api.FetchFirmwares()
	if err != nil {
		return err
	}

	models := make(map[string]bool)
	for _, device := range devices {
		o.devices[device.IP.String()].NewFWVersion = firmwares[device.Model].Version

		// If a model has already been marked as seen or out-of-date, make sure to respect
		// the flag independently of what future devices may suggest.
		if models[device.Model] == true {
			continue
		}

		// Only set the model flag if a discovered device has an out-of-date firmware,
		// otherwise its firmware will be downloaded and not used.
		if o.devices[device.IP.String()].CurrentFWVersion != firmwares[device.Model].Version {
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

			http.HandleFunc("/"+model, func(w http.ResponseWriter, r *http.Request) {
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
	body, err := o.api.GetFirmware(model)
	defer body.Close()
	if err != nil {
		return "", err
	}

	err = os.MkdirAll(o.downloadDir, 0700)
	if err != nil {
		return "", err
	}

	filename := strings.Join([]string{strings.Join([]string{model, strings.Replace(firmware.Version, "/", "-", -1)}, "-"), path.Ext(firmware.URL)}, "")
	out, err := os.Create(filepath.Join(o.downloadDir, filename))
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, body)
	if err != nil {
		return "", err
	}

	log.Debugf("Downloaded firmware %v to %v\n", path.Base(firmware.URL), filepath.Join(o.downloadDir, filename))

	return filepath.Join(o.downloadDir, filename), nil
}

// Devices returns a list of discovered devices on the local network
// along with their current settings state.
func (o *OTAUpdater) Devices() (map[string]*Device, error) {
	if o.devices != nil {
		return o.devices, nil
	}

	devices, err := o.browser.DiscoverDevices()
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
	response, err := http.Get(fmt.Sprintf("%s/ota?url=http://%s:%d/%s", device.GetBaseURL(), o.serverIP.String(), o.httpPort, device.Model))
	if err != nil {
		log.Debug(err)
		return err
	}

	defer response.Body.Close()

	time.Sleep(5 * time.Second)

	return nil
}

// PromptForUpgrade prompts the end-user to decide whether or not to
// perform an upgrade of a device.
func (o *OTAUpdater) PromptForUpgrade() error {
	devices, err := o.Devices()
	if err != nil {
		log.Fatal(err)
	}

	for _, device := range devices {
		if device.CurrentFWVersion == device.NewFWVersion {
			log.Infof("Skipping %v (%v) as firmware version is up-to-date (%v)", device.ModelName(), device.IP, device.CurrentFWVersion)
			continue
		}

		upgrade := false
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

		o.UpgradeDevice(device)
	}

	return nil
}
