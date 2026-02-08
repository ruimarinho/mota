package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// https://github.com/ALLTERCO/fleet-management/blob/ed4f8bc5c5944e39e3c3149c6573176103317e20/backend/cfg/devices.json#L11
var gen2PlusModels = []string{
	// Gen2 Plus
	"BluGw",
	"PlugUS",
	"Plus1",
	"Plus10V",
	"Plus1Mini",
	"Plus1PM",
	"Plus1PMMini",
	"Plus2PM",
	"PlusHT",
	"PlusI4",
	"PlusPlugIT",
	"PlusPlugS",
	"PlusPlugUK",
	"PlusPMMini",
	"PlusRGBWPM",
	"PlusSmoke",
	"PlusUni",
	"PlusWallDimmer",
	"WallDisplay",
	// Gen2 Pro
	"Pro1",
	"Pro1PM",
	"Pro2",
	"Pro2PM",
	"Pro3",
	"Pro3EM",
	"Pro4PM",
	"ProDimmerx",
	"ProEM",
	"ProRGBWWPM",
	// Gen3
	"Mini1G3",
	"Mini1PMG3",
	"MiniPMG3",
	"1G3",
	"1MiniG3",
	"1PMG3",
	"1PMMiniG3",
	"2PMG3",
	"0-10VDimmerG3",
	"Dimmer0110VPMG3",
	"RGBWPMminiG3",
	"EMXG3",
	"EMG3",
	"S3EMG3",
	"S1LG3",
	"S2LG3",
	"S2PMG3Shutter",
	"i4G3",
	"HTG3",
	"FloodG3",
	"PlugSG3",
	"DimmerG3",
	"PlugPMG3",
	"BluGwG3",
	"XMOD1",
	// Gen4
	"1G4",
	"1MiniG4",
	"1PMG4",
	"2PMG4",
	"FloodG4",
	"i4G4",
	"PlugSG4",
	"DimmerG4",
	"EMMiniG4",
}

// gen2PlusAPINames maps device app names to the update API model names
// where they differ. Most models use the same name, but some Gen3 models
// use a different CDN/API name.
var gen2PlusAPINames = map[string]string{
	// Gen3
	"1G3":   "S1G3",
	"1PMG3": "S1PMG3",
	"2PMG3": "S2PMG3",
	"i4G3":  "I4G3",
	// Gen4
	"1G4":   "S1G4",
	"1PMG4": "S1PMG4",
	"2PMG4": "S2PMG4",
	"i4G4":  "I4G4",
}

// gen2PlusDeviceAliases maps device-reported model names to the canonical
// internal names used in gen2PlusModels. Some devices report variant names
// (e.g. Zigbee suffix) that share firmware with the base model.
var gen2PlusDeviceAliases = map[string]string{
	"S2PMG4ZB": "2PMG4",
}

// var gen3Models = []string{
// }

// LocalFirmware is a structure that holds information about a specific
// remote firmware file.
type LocalFirmware struct {
	Model    string
	Version  string
	Beta     bool
	Filename string
}

// RemoteFirmware is a structure that holds information about a specific
// remote firmware file.
type RemoteFirmware struct {
	Model       string
	URL         string
	Version     string
	BetaURL     string `json:"beta_url"`
	BetaVersion string `json:"beta_ver"`
}

func (f *RemoteFirmware) StableID() string {
	return fmt.Sprintf("%s-%s@stable", f.Model, f.Version)
}

func (f *RemoteFirmware) BetaID() string {
	return fmt.Sprintf("%s-%s@beta", f.Model, f.BetaVersion)
}

// APIClient is a struct that represents an API client that fetches
// information from the Shelly Cloud APIs.
type APIClient struct {
	baseURL       string
	gen2BaseURL   string
	firmwareCache sync.Map
	firmwares     map[string]RemoteFirmware
	httpClient    *http.Client
	includeBetas  bool
}

type response struct {
	IsOk bool                      `json:"isok"`
	Data map[string]RemoteFirmware `json:"data"`
}

type gen2FirmwareInfo struct {
	Version string `json:"version"`
	BuildID string `json:"build_id"`
	URL     string `json:"url"`
}

type gen2AltVariant struct {
	Stable gen2FirmwareInfo `json:"stable"`
	Beta   gen2FirmwareInfo `json:"beta"`
}

type gen2response struct {
	Stable gen2FirmwareInfo           `json:"stable"`
	Beta   gen2FirmwareInfo           `json:"beta"`
	Alt    map[string]gen2AltVariant  `json:"alt"`
}

// APIClientOption is an option interface for APIClient.
type APIClientOption func(*APIClient)

// WithAPIHTTPClient is an APIClient option that allows overriding the
// HTTP client used to make requests.
func WithAPIHTTPClient(httpClient *http.Client) APIClientOption {
	return func(client *APIClient) {
		client.httpClient = httpClient
	}
}

// WithBaseURL is an APIClient option that allows overriding the
// base URL used for remote calls.
func WithBaseURL(baseURL string) APIClientOption {
	return func(client *APIClient) {
		client.baseURL = baseURL
	}
}

// WithBetaFirmware is an APIClient option that enables beta firmware
// support when available
func WithBetaFirmware(includeBetas bool) APIClientOption {
	return func(client *APIClient) {
		client.includeBetas = includeBetas
	}
}

// WithGen2BaseURL is an APIClient option that allows overriding the
// base URL used for Gen2+ firmware API calls.
func WithGen2BaseURL(baseURL string) APIClientOption {
	return func(client *APIClient) {
		client.gen2BaseURL = baseURL
	}
}

// NewAPIClient returns a new instance of the APIClient with default
// options.
func NewAPIClient(options ...APIClientOption) *APIClient {
	client := &APIClient{
		baseURL:     "https://api.shelly.cloud",
		gen2BaseURL: "https://updates.shelly.cloud",
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Timeout: 10 * time.Second,
		},
		firmwareCache: sync.Map{},
	}

	for _, option := range options {
		option(client)
	}

	return client
}

// FetchVersions returns a list of remotely available firmwares.
func (client *APIClient) FetchVersions() (map[string]RemoteFirmware, error) {
	// Return cached copy in-memory if available.
	if len(client.firmwares) > 0 {
		return client.firmwares, nil
	}

	// Gen1
	apiResponse, err := client.httpClient.Get(client.baseURL + "/files/firmware")
	if err != nil {
		return nil, err
	}
	defer apiResponse.Body.Close()

	var decoded response
	err = json.NewDecoder(apiResponse.Body).Decode(&decoded)
	if err != nil {
		return nil, err
	}

	client.firmwares = make(map[string]RemoteFirmware)

	for gen1Model, data := range decoded.Data {
		client.firmwares[gen1Model] = RemoteFirmware{
			Model:       gen1Model,
			URL:         data.URL,
			Version:     data.Version,
			BetaURL:     data.BetaURL,
			BetaVersion: data.BetaVersion,
		}
	}

	// Gen2 + Gen3 + Gen4 (fetched concurrently with bounded workers)
	const maxWorkers = 10
	sem := make(chan struct{}, maxWorkers)
	var mu sync.Mutex
	var wg sync.WaitGroup
	var fetchErr error

	for _, gen2Model := range gen2PlusModels {
		mu.Lock()
		if fetchErr != nil {
			mu.Unlock()
			break
		}
		mu.Unlock()

		wg.Add(1)
		sem <- struct{}{}

		go func(model string) {
			defer wg.Done()
			defer func() { <-sem }()

			apiModel := model
			if mapped, ok := gen2PlusAPINames[model]; ok {
				apiModel = mapped
			}

			apiResp, err := client.httpClient.Get(client.gen2BaseURL + "/update/" + apiModel)
			if err != nil {
				mu.Lock()
				if fetchErr == nil {
					fetchErr = err
				}
				mu.Unlock()
				return
			}
			defer apiResp.Body.Close()

			if apiResp.StatusCode != http.StatusOK {
				log.Debugf("No firmware available from update API for model %s (HTTP %d)", model, apiResp.StatusCode)
				return
			}

			var decoded gen2response
			err = json.NewDecoder(apiResp.Body).Decode(&decoded)
			if err != nil {
				log.Debugf("Failed to decode Gen2+ firmware response for model %s", model)
				return
			}

			if decoded.Stable.Version == "" {
				log.Debugf("No stable firmware version available for model %s", model)
				return
			}

			fw := RemoteFirmware{
				Model:       model,
				URL:         decoded.Stable.URL,
				Version:     decoded.Stable.Version,
				BetaURL:     decoded.Beta.URL,
				BetaVersion: decoded.Beta.Version,
			}

			mu.Lock()
			client.firmwares[model] = fw

			// Store alt variants (e.g. ZB/Zigbee builds).
			for altName, alt := range decoded.Alt {
				if alt.Stable.Version != "" {
					client.firmwares[altName] = RemoteFirmware{
						Model:       altName,
						URL:         alt.Stable.URL,
						Version:     alt.Stable.Version,
						BetaURL:     alt.Beta.URL,
						BetaVersion: alt.Beta.Version,
					}
				}
			}

			mu.Unlock()
		}(gen2Model)
	}

	wg.Wait()

	if fetchErr != nil {
		return nil, fetchErr
	}

	return client.firmwares, nil
}

// DownloadFirmware returns the local destination of the firmware that
// has been requested to download for a particular model.
func (client *APIClient) DownloadFirmware(remoteFirmware RemoteFirmware, betaVersion bool, downloadDir string) (string, error) {
	id := remoteFirmware.StableID()
	url := remoteFirmware.URL
	version := remoteFirmware.Version
	if betaVersion {
		id = remoteFirmware.BetaID()
		url = remoteFirmware.BetaURL
		version = remoteFirmware.BetaVersion
	}

	if localPath, ok := client.firmwareCache.Load(id); ok {
		return localPath.(string), nil
	}

	response, err := client.httpClient.Get(url)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download firmware from %s: HTTP %d", url, response.StatusCode)
	}

	err = os.MkdirAll(downloadDir, 0700)
	if err != nil {
		return "", err
	}

	extension := path.Ext(url)
	if extension == "" {
		extension = ".zip"
	}

	filename := strings.Replace(strings.Join([]string{remoteFirmware.Model, version, extension}, ""), "/", "-", -1)

	fullPath := filepath.Join(downloadDir, filename)
	out, err := os.Create(fullPath)

	if err != nil {
		return "", err
	}

	defer out.Close()

	_, err = io.Copy(out, response.Body)
	if err != nil {
		return "", err
	}

	log.Debugf("Downloaded firmware %v for model %s to %v", version, remoteFirmware.Model, fullPath)

	client.firmwareCache.Store(id, fullPath)

	return fullPath, nil
}

// GetVersion returns the most recent firmware version available for a model
func (client *APIClient) GetLatestFirmwareAvailable(model string) (RemoteFirmware, error) {
	firmwares, err := client.FetchVersions()
	if err != nil {
		return RemoteFirmware{}, err
	}

	if firmware, ok := firmwares[model]; ok {
		return firmware, nil
	}

	// Try device alias: some devices report variant names (e.g. "S2PMG4ZB")
	// that share firmware with a base model (e.g. "2PMG4").
	if canonical, ok := gen2PlusDeviceAliases[model]; ok {
		if firmware, ok := firmwares[canonical]; ok {
			return firmware, nil
		}
	}

	// Try reverse lookup: the device may report an API name (e.g. "S1G3")
	// while firmware is cached under the internal name (e.g. "1G3").
	for internal, api := range gen2PlusAPINames {
		if api == model {
			if firmware, ok := firmwares[internal]; ok {
				return firmware, nil
			}
		}
	}

	return RemoteFirmware{}, fmt.Errorf("remote firmware for model %s not found", model)
}

// GetURL returns the most recent firmware download URL available for a model
func (client *APIClient) GetURL(model string) (string, error) {
	firmwares, err := client.FetchVersions()
	if err != nil {
		return "", err
	}

	version := firmwares[model].URL
	if client.includeBetas && firmwares[model].BetaURL != "" {
		version = firmwares[model].BetaURL
	}

	return version, nil
}
